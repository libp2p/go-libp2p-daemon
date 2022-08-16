package p2pd

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"

	ggio "github.com/gogo/protobuf/io"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-daemon/internal/utils"
	pb "github.com/libp2p/go-libp2p-daemon/pb"
)

func (d *Daemon) handlePersistentConn(r ggio.Reader, unsafeW ggio.WriteCloser) {
	w := utils.NewSafeWriter(unsafeW)

	var streamHandlers []string
	defer func() {
		d.mx.Lock()
		defer d.mx.Unlock()
		for _, proto := range streamHandlers {
			p := protocol.ID(proto)
			d.registeredUnaryProtocols[p].Remove(w)
			if d.registeredUnaryProtocols[p].Len() == 0 {
				d.host.RemoveStreamHandler(p)
				delete(d.registeredUnaryProtocols, p)
			}
		}
	}()

	if d.cancelTerminateTimer != nil {
		d.cancelTerminateTimer()
	}

	d.terminateWG.Add(1)
	defer d.terminateWG.Done()

	d.terminateOnce.Do(func() { go d.awaitTermination() })

	if err := w.WriteMsg(&pb.Response{Type: pb.Response_OK.Enum()}); err != nil {
		log.Debugw("error writing message", "error", err)
		return
	}

	for {
		var req pb.PersistentConnectionRequest
		if err := r.ReadMsg(&req); err != nil {
			if err != io.EOF {
				log.Debugw("error reading message", "error", err)
			}
			return
		}

		go d.handlePersistentConnRequest(req, w, &streamHandlers)
	}
}

func (d *Daemon) handlePersistentConnRequest(req pb.PersistentConnectionRequest, w ggio.WriteCloser, streamHandlers *[]string) {
	callID, err := uuid.FromBytes(req.CallId)
	if err != nil {
		log.Debugw("bad call id: ", "error", err)
		return
	}

	switch req.Message.(type) {
	case *pb.PersistentConnectionRequest_AddUnaryHandler:
		resp := d.doAddUnaryHandler(w, callID, req.GetAddUnaryHandler())

		d.mx.Lock()
		if _, ok := resp.Message.(*pb.PersistentConnectionResponse_DaemonError); !ok {
			*streamHandlers = append(
				*streamHandlers,
				*req.GetAddUnaryHandler().Proto,
			)
		}
		d.mx.Unlock()

		if err := w.WriteMsg(resp); err != nil {
			log.Debugw("error writing message", "error", err)
			return
		}

	case *pb.PersistentConnectionRequest_RemoveUnaryHandler:
		resp := d.doRemoveUnaryHandler(w, callID, req.GetRemoveUnaryHandler())

		d.mx.Lock()
		if _, ok := resp.Message.(*pb.PersistentConnectionResponse_DaemonError); !ok {
			for index, proto := range *streamHandlers {
				if proto == *req.GetRemoveUnaryHandler().Proto {
					*streamHandlers = append((*streamHandlers)[:index], (*streamHandlers)[index+1:]...)
					break
				}
			}
		}
		d.mx.Unlock()

		if err := w.WriteMsg(resp); err != nil {
			log.Debugw("error writing message", "error", err)
			return
		}

	case *pb.PersistentConnectionRequest_CallUnary:
		ctx, cancel := context.WithCancel(context.Background())
		d.cancelUnary.Store(callID, cancel)
		defer cancel()

		defer d.cancelUnary.Delete(callID)

		resp := d.doUnaryCall(ctx, callID, &req)

		if err := w.WriteMsg(resp); err != nil {
			log.Debugw("error reading message", "error", err)
			return
		}

	case *pb.PersistentConnectionRequest_UnaryResponse:
		d.sendReponseToRemote(&req)

	case *pb.PersistentConnectionRequest_Cancel:
		cf, found := d.cancelUnary.Load(callID)
		if !found {
			return
		}

		cf.(context.CancelFunc)()
	}
}

func (d *Daemon) doAddUnaryHandler(w ggio.Writer, callID uuid.UUID, req *pb.AddUnaryHandlerRequest) *pb.PersistentConnectionResponse {
	d.mx.Lock()
	defer d.mx.Unlock()

	p := protocol.ID(*req.Proto)
	round_robin, ok := d.registeredUnaryProtocols[p]
	if !ok {
		d.registeredUnaryProtocols[p] = utils.NewRoundRobin()
		d.registeredUnaryProtocols[p].Append(w)
		d.host.SetStreamHandler(p, d.persistentStreamHandler)
	} else if !req.GetBalanced() {
		return errorUnaryCallString(
			callID,
			fmt.Sprintf("handler for protocol %s already set", *req.Proto),
		)
	} else {
		round_robin.Append(w)
	}

	return okUnaryCallResponse(callID)
}

func (d *Daemon) doRemoveUnaryHandler(w ggio.Writer, callID uuid.UUID, req *pb.RemoveUnaryHandlerRequest) *pb.PersistentConnectionResponse {
	d.mx.Lock()
	defer d.mx.Unlock()

	p := protocol.ID(*req.Proto)
	round_robin, ok := d.registeredUnaryProtocols[p]
	if !ok {
		return errorUnaryCallString(
			callID,
			fmt.Sprintf("handler for protocol %s does not exist", *req.Proto),
		)
	}

	ok = round_robin.Remove(w)
	if !ok {
		return errorUnaryCallString(
			callID,
			fmt.Sprintf("handler for protocol %s was not created in this persistent connection", *req.Proto),
		)
	}
	if round_robin.Len() == 0 {
		d.host.RemoveStreamHandler(p)
		delete(d.registeredUnaryProtocols, p)
	}
	return okUnaryCallResponse(callID)
}

func (d *Daemon) doUnaryCall(ctx context.Context, callID uuid.UUID, req *pb.PersistentConnectionRequest) *pb.PersistentConnectionResponse {
	pid, err := peer.IDFromBytes(req.GetCallUnary().Peer)
	if err != nil {
		return errorUnaryCall(callID, err)
	}

	remoteStream, err := d.host.NewStream(
		ctx,
		pid,
		protocol.ID(*req.GetCallUnary().Proto),
	)
	if err != nil {
		return errorUnaryCall(callID, err)
	}
	defer remoteStream.Close()

	select {
	case response := <-d.exchangeMessages(ctx, remoteStream, req):
		return response

	case <-ctx.Done():
		return okUnaryCallCancelled(callID)
	}
}

func (d *Daemon) exchangeMessages(ctx context.Context, s network.Stream, req *pb.PersistentConnectionRequest) <-chan *pb.PersistentConnectionResponse {
	callID, _ := uuid.FromBytes(req.CallId)
	rc := make(chan *pb.PersistentConnectionResponse)

	go func() {
		defer close(rc)

		if err := ggio.NewDelimitedWriter(s).WriteMsg(req); ctx.Err() != nil {
			return
		} else if err != nil {
			rc <- errorUnaryCall(callID, err)
			return
		}

		remoteResp := &pb.PersistentConnectionRequest{}
		if err := ggio.NewDelimitedReader(s, d.persistentConnMsgMaxSize).ReadMsg(remoteResp); ctx.Err() != nil {
			return
		} else if err != nil {
			rc <- errorUnaryCall(callID, err)
			return
		}

		resp := okUnaryCallResponse(callID)
		resp.Message = &pb.PersistentConnectionResponse_CallUnaryResponse{
			CallUnaryResponse: remoteResp.GetUnaryResponse(),
		}

		select {
		case rc <- resp:
			return

		case <-ctx.Done():
			return
		}
	}()

	return rc
}

// notifyWhenClosed writers to a semaphor channel if the given io.Reader fails to
// read before the context was cancelled
func notifyWhenClosed(ctx context.Context, r io.Reader) <-chan struct{} {
	event := make(chan struct{})

	go func() {
		defer close(event)

		buff := make([]byte, 1)
		if _, err := r.Read(buff); err != nil {
			select {
			case event <- struct{}{}:
			case <-ctx.Done():
			}
		}
	}()

	return event
}

// getPersistentStreamHandler returns a libp2p stream handler tied to a
// given persistent client stream
func (d *Daemon) persistentStreamHandler(s network.Stream) {
	defer s.Close()

	p := s.Protocol()

	d.mx.Lock()
	cws, ok := d.registeredUnaryProtocols[p]
	var cw ggio.Writer
	if ok {
		cw = cws.Next().(ggio.Writer)
	}
	d.mx.Unlock()

	if !ok {
		log.Debugw("unexpected persistent stream", "protocol", p)
		return
	}

	req := &pb.PersistentConnectionRequest{}
	if err := ggio.NewDelimitedReader(s, d.persistentConnMsgMaxSize).ReadMsg(req); err != nil {
		log.Debugw("failed to read proto from incoming p2p stream", "error", err)
		return
	}

	if req.GetCallUnary() == nil {
		log.Debug("proto is expected to include callUnary but does not have it")
		return
	}

	// now the peer field stores the caller's peer id
	req.GetCallUnary().Peer = []byte(s.Conn().RemotePeer())

	callID, err := uuid.FromBytes(req.CallId)
	if err != nil {
		log.Debugw("bad call id in p2p handler", "error", err)
		return
	}

	rc := make(chan *pb.PersistentConnectionRequest)
	d.responseWaiters.Store(callID, rc)
	defer d.responseWaiters.Delete(callID)

	ctx, cancel := context.WithCancel(d.ctx)
	defer cancel()

	resp := &pb.PersistentConnectionResponse{
		CallId: req.CallId,
		Message: &pb.PersistentConnectionResponse_RequestHandling{
			RequestHandling: req.GetCallUnary(),
		},
	}

	if err := cw.WriteMsg(resp); err != nil {
		log.Debugw("failed to write message to client", "error", err)
		return
	}

	select {
	case <-notifyWhenClosed(ctx, s):
		if err := cw.WriteMsg(
			&pb.PersistentConnectionResponse{
				CallId: callID[:],
				Message: &pb.PersistentConnectionResponse_Cancel{
					Cancel: &pb.Cancel{},
				},
			},
		); err != nil {
			log.Debugw("failed to write to client", "error", err)
		}
	case response := <-rc:
		w := ggio.NewDelimitedWriter(s)
		if err := w.WriteMsg(response); err != nil {
			log.Debugw("failed to write message to remote", "error", err)
		}
	}
}

func (d *Daemon) sendReponseToRemote(req *pb.PersistentConnectionRequest) {
	callID, err := uuid.FromBytes(req.CallId)
	if err != nil {
		log.Debugf("failed to unmarshal call id from bytes: %v", err)
		return
	}

	rc, found := d.responseWaiters.Load(callID)
	if !found {
		log.Debugf("could not find request awaiting response for following call id: %s", callID.String())
		return
	}

	rc.(chan *pb.PersistentConnectionRequest) <- req
}

func errorUnaryCall(callID uuid.UUID, err error) *pb.PersistentConnectionResponse {
	return errorUnaryCallString(callID, err.Error())
}

func errorUnaryCallString(callID uuid.UUID, errMsg string) *pb.PersistentConnectionResponse {
	return &pb.PersistentConnectionResponse{
		CallId: callID[:],
		Message: &pb.PersistentConnectionResponse_DaemonError{
			DaemonError: &pb.DaemonError{Message: &errMsg},
		},
	}
}

func okUnaryCallResponse(callID uuid.UUID) *pb.PersistentConnectionResponse {
	return &pb.PersistentConnectionResponse{CallId: callID[:]}
}

func okUnaryCallCancelled(callID uuid.UUID) *pb.PersistentConnectionResponse {
	return &pb.PersistentConnectionResponse{
		CallId: callID[:],
		Message: &pb.PersistentConnectionResponse_Cancel{
			Cancel: &pb.Cancel{},
		},
	}
}
