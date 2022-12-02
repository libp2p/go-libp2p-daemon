package p2pclient

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p-daemon/internal/utils"
	pb "github.com/libp2p/go-libp2p-daemon/pb"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	ggio "github.com/gogo/protobuf/io"
)

type persistentConnectionResponseFuture chan *pb.PersistentConnectionResponse

type UnaryHandlerFunc func(context.Context, []byte) ([]byte, error)

func (u UnaryHandlerFunc) handle(ctx context.Context, w ggio.Writer, req *pb.PersistentConnectionResponse) {
	result, err := u(ctx, req.GetRequestHandling().Data)

	response := &pb.CallUnaryResponse{}
	if err == nil {
		response.Result = &pb.CallUnaryResponse_Response{
			Response: result,
		}
	} else {
		response.Result = &pb.CallUnaryResponse_Error{
			Error: []byte(err.Error()),
		}
	}

	w.WriteMsg(
		&pb.PersistentConnectionRequest{
			CallId: req.CallId,
			Message: &pb.PersistentConnectionRequest_UnaryResponse{
				UnaryResponse: response,
			},
		},
	)
}

func (c *Client) run(r ggio.Reader, w ggio.Writer) {
	for {
		var resp pb.PersistentConnectionResponse
		r.ReadMsg(&resp)

		callID, err := uuid.FromBytes(resp.CallId)
		if err != nil {
			log.Debugw("received response with bad call id:", "error", err)
			continue
		}

		switch resp.Message.(type) {
		case *pb.PersistentConnectionResponse_RequestHandling:
			proto := protocol.ID(*resp.GetRequestHandling().Proto)

			h, found := c.unaryHandlers.Load(proto)
			handler, ok := h.(UnaryHandlerFunc)
			if !ok {
				log.Fatal("could not load handler for %s: failed to cast it to unary handler\n", proto)
				return
			}

			if !found {
				w.WriteMsg(makeErrProtoNotFoundMsg(resp.CallId, string(proto)))
			}

			go func() {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				handler.handle(ctx, w, &resp)
			}()

		case *pb.PersistentConnectionResponse_DaemonError, *pb.PersistentConnectionResponse_CallUnaryResponse, *pb.PersistentConnectionResponse_Cancel, nil:
			go func() {
				rC, _ := c.callFutures.LoadOrStore(callID, make(persistentConnectionResponseFuture))
				rC.(persistentConnectionResponseFuture) <- &resp
			}()
		}
	}

}

func (c *Client) getPersistentWriter() ggio.WriteCloser {
	c.openPersistentConn.Do(
		func() {
			conn, err := c.newControlConn()
			if err != nil {
				panic(err)
			}

			w := utils.NewSafeWriter(ggio.NewDelimitedWriter(conn))
			w.WriteMsg(&pb.Request{Type: pb.Request_PERSISTENT_CONN_UPGRADE.Enum()})
			c.persistentConnWriter = w

			r := ggio.NewDelimitedReader(conn, network.MessageSizeMax)

			var msg pb.Response
			if err := r.ReadMsg(&msg); err != nil {
				panic(err)
			}

			if *msg.Type != pb.Response_OK {
				panic("failed to open persistent connection")
			}

			go c.run(r, c.persistentConnWriter)

		},
	)

	return c.persistentConnWriter
}

func (c *Client) getResponse(callID uuid.UUID) (*pb.PersistentConnectionResponse, error) {
	rc, _ := c.callFutures.LoadOrStore(callID, make(persistentConnectionResponseFuture))
	defer c.callFutures.Delete(callID)

	response := <-rc.(persistentConnectionResponseFuture)
	if dErr := response.GetDaemonError(); dErr != nil {
		return nil, newDaemonError(dErr)
	}

	return response, nil
}

func (c *Client) AddUnaryHandler(proto protocol.ID, handler UnaryHandlerFunc, balanced bool) error {
	w := c.getPersistentWriter()

	callID := uuid.New()

	w.WriteMsg(
		&pb.PersistentConnectionRequest{
			CallId: callID[:],
			Message: &pb.PersistentConnectionRequest_AddUnaryHandler{
				AddUnaryHandler: &pb.AddUnaryHandlerRequest{
					Proto:    (*string)(&proto),
					Balanced: &balanced,
				},
			},
		},
	)

	if _, err := c.getResponse(callID); err != nil {
		return err
	}

	c.unaryHandlers.Store(proto, handler)

	return nil
}

func (c *Client) RemoveUnaryHandler(proto protocol.ID) error {
	w := c.getPersistentWriter()

	callID := uuid.New()

	w.WriteMsg(
		&pb.PersistentConnectionRequest{
			CallId: callID[:],
			Message: &pb.PersistentConnectionRequest_RemoveUnaryHandler{
				RemoveUnaryHandler: &pb.RemoveUnaryHandlerRequest{
					Proto: (*string)(&proto),
				},
			},
		},
	)

	if _, err := c.getResponse(callID); err != nil {
		return err
	}

	c.unaryHandlers.Delete(proto)

	return nil
}

func (c *Client) CallUnaryHandler(
	ctx context.Context,
	peerID peer.ID,
	proto protocol.ID,
	payload []byte,
) ([]byte, error) {

	w := c.getPersistentWriter()

	callID := uuid.New()

	// both methods don't return any errors
	cid, err := callID.MarshalBinary()
	if err != nil {
		return nil, err
	}
	pid, err := peerID.MarshalBinary()
	if err != nil {
		return nil, err
	}

	done := make(chan struct{})
	w.WriteMsg(
		&pb.PersistentConnectionRequest{
			CallId: cid,
			Message: &pb.PersistentConnectionRequest_CallUnary{
				CallUnary: &pb.CallUnaryRequest{
					Peer:  pid,
					Proto: (*string)(&proto),
					Data:  payload,
				},
			},
		},
	)

	go func() {
		defer close(done)

		select {
		case <-done:
			return
		case <-ctx.Done():
			w.WriteMsg(
				&pb.PersistentConnectionRequest{
					CallId:  cid,
					Message: &pb.PersistentConnectionRequest_Cancel{Cancel: &pb.Cancel{}},
				},
			)
		}
	}()

	response, err := c.getResponse(callID)
	if err != nil {
		return nil, err
	}

	if response.GetCancel() != nil {
		return nil, ctx.Err()
	}

	result := response.GetCallUnaryResponse()
	if len(result.GetError()) != 0 {
		return nil, newP2PHandlerError(result)
	}

	select {
	case done <- struct{}{}:
		return result.GetResponse(), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func newDaemonError(dErr *pb.DaemonError) error {
	return &DaemonError{message: *dErr.Message}
}

type DaemonError struct {
	message string
}

func (de *DaemonError) Error() string {
	return fmt.Sprintf("Daemon failed with %s:", de.message)
}

func newP2PHandlerError(resp *pb.CallUnaryResponse) error {
	return &P2PHandlerError{message: string(resp.GetError())}
}

type P2PHandlerError struct {
	message string
}

func (he *P2PHandlerError) Error() string {
	return he.message
}

func makeErrProtoNotFoundMsg(callID []byte, proto string) *pb.PersistentConnectionRequest {
	return &pb.PersistentConnectionRequest{
		CallId: callID,
		Message: &pb.PersistentConnectionRequest_UnaryResponse{
			UnaryResponse: &pb.CallUnaryResponse{
				Result: &pb.CallUnaryResponse_Error{
					Error: []byte(fmt.Sprintf("handler for protocl %s not found", proto)),
				},
			},
		},
	}
}
