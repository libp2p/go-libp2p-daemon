package p2pd

import (
	"fmt"
	"io"
	"net"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"

	ggio "github.com/gogo/protobuf/io"
	"github.com/libp2p/go-libp2p-core/protocol"
	pb "github.com/libp2p/go-libp2p-daemon/pb"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
)

func (d *Daemon) doPersistentConnect(req *pb.Request) *pb.Response {
	if req.PersistentConnect == nil {
		return errorResponseString("Malformed request; missing parameters")
	}

	addr, err := ma.NewMultiaddr(*req.PersistentConnect.ListenAddr)
	if err != nil {
		return errorResponseString(
			fmt.Sprintf("Failed to read multiaddr: %v", err),
		)
	}
	protocol := protocol.ID(*req.PersistentConnect.Proto)

	// make sure we keep one connection per protocol
	d.mx.Lock()
	defer d.mx.Unlock()

	if _, found := d.unaryProtocols[protocol]; found {
		return errorResponseString(
			fmt.Sprintf("Persistent connection for protocol `%s` already open", protocol),
		)
	}

	listener, err := manet.Listen(addr)
	if err != nil {
		return errorResponseString(
			fmt.Sprintf("Socket connetion failed: %v", err),
		)
	}
	conn, err := listener.Accept()
	if err != nil {
		return errorResponseString(
			fmt.Sprintf("Failed to accept connection: %v", err),
		)
	}

	d.unaryProtocols[protocol] = true

	go d.handlePersistentConn(protocol, conn)

	return okResponse()
}

func (d *Daemon) handlePersistentConn(pid protocol.ID, c net.Conn) {
	defer c.Close()
	defer func() {
		d.mx.Lock()
		d.unaryProtocols[pid] = false
		d.mx.Unlock()
	}()

	r := ggio.NewDelimitedReader(c, network.MessageSizeMax)
	w := ggio.NewDelimitedWriter(c)

	for {
		var req pb.Request
		err := r.ReadMsg(&req)
		if err != nil {
			if err != io.EOF {
				log.Debugw("error reading message", "error", err)
			}
			return
		}

		log.Debugw("request", "type", req.GetType())

		switch req.GetType() {
		case pb.Request_CALL_UNARY:
			resp := d.doUnaryCall(&req)
			if err := w.WriteMsg(resp); err != nil {
				log.Debugw("error reading message", "error", err)
				return
			}
		}
	}
}

func (d *Daemon) openUnaryStream(req *pb.Request) (*pb.Response, network.Stream) {
	if req.CallUnary == nil {
		return malformedRequestErrorResponse(), nil
	}

	ctx, cancel := d.requestContext(req.CallUnary.GetTimeout())
	defer cancel()

	pid, err := peer.IDFromBytes(req.CallUnary.Peer)
	if err != nil {
		return errorResponseString(
			fmt.Sprintf("Failed to parse peer id: %v", err),
		), nil
	}

	protos := make([]protocol.ID, len(req.CallUnary.Proto))
	for x, str := range req.CallUnary.Proto {
		protos[x] = protocol.ID(str)
	}

	s, err := d.host.NewStream(ctx, pid, protos...)
	if err != nil {
		return errorResponse(err), nil
	}

	res := okResponse()
	res.StreamInfo = makeStreamInfo(s)

	return res, s
}

func (d *Daemon) doUnaryCall(req *pb.Request) *pb.Response {
	if req.CallUnary == nil {
		return malformedRequestErrorResponse()
	}

	// pare peer id
	pid, err := peer.IDFromBytes(req.CallUnary.Peer)
	if err != nil {
		return errorResponseString(
			fmt.Sprintf("Failed to parse peer id: %v", err),
		)
	}

	// parse protocols
	protos := make([]protocol.ID, len(req.CallUnary.Proto))
	for x, str := range req.CallUnary.Proto {
		protos[x] = protocol.ID(str)
	}

	// parse timeout
	ctx, cancel := d.requestContext(req.CallUnary.GetTimeout())
	defer cancel()

	// TODO: cache streams
	s, err := d.host.NewStream(ctx, pid, protos...)
	if err != nil {
		return errorResponse(err)
	}
	defer s.Close()

	requestData := req.CallUnary.GetData()
	if _, err := s.Write(requestData); err != nil {
		return errorResponseString(
			fmt.Sprintf("Failed to write message: %s", err.Error()),
		)
	}

	resp := okResponse()
	resp.CallUnaryResponse = &pb.CallUnaryResponse{}
	if _, err := s.Read(resp.CallUnaryResponse.Result); err != nil {
		return errorResponseString(
			fmt.Sprintf("Failed to read message: %s", err.Error()),
		)
	}

	return resp
}
