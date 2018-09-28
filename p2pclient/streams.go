package p2pclient

import (
	"errors"
	"io"
	"net"

	ggio "github.com/gogo/protobuf/io"
	pb "github.com/libp2p/go-libp2p-daemon/pb"
)

// NewStream initializes a new stream on one of the protocols in protos with
// the specified peer.
func (c *Client) NewStream(peer []byte, protos []string) (*pb.StreamInfo, io.ReadWriteCloser, error) {
	control, err := c.newControlConn()
	if err != nil {
		return nil, nil, err
	}
	r := ggio.NewDelimitedReader(control, MessageSizeMax)
	w := ggio.NewDelimitedWriter(control)

	req := &pb.Request{
		Type: pb.Request_STREAM_OPEN.Enum(),
		StreamOpen: &pb.StreamOpenRequest{
			Peer:  peer,
			Proto: protos,
		},
	}

	if err := w.WriteMsg(req); err != nil {
		control.Close()
		return nil, nil, err
	}

	res := &pb.Response{}
	if err := r.ReadMsg(res); err != nil {
		control.Close()
		return nil, nil, err
	}

	if err := res.GetError(); err != nil {
		control.Close()
		return nil, nil, errors.New(err.GetMsg())
	}

	return res.GetStreamInfo(), control, nil
}

// Close stops the listener socket.
func (c *Client) Close() error {
	if c.listener != nil {
		return c.listener.Close()
	}
	return nil
}

func (c *Client) listen() error {
	l, err := net.Listen("unix", c.listenPath)
	if err != nil {
		return err
	}
	c.listener = l

	go func(c *Client) {
		for {
			conn, err := c.listener.Accept()
			if err != nil {
				log.Errorf("accepting incoming connection: %s", err)
				return
			}

			r := ggio.NewDelimitedReader(conn, MessageSizeMax)
			streamInfo := &pb.StreamInfo{}
			if err := r.ReadMsg(streamInfo); err != nil {
				log.Errorf("reading stream info: %s", err)
				conn.Close()
				continue
			}

			c.mhandlers.Lock()
			defer c.mhandlers.Unlock()
			handler, ok := c.handlers[streamInfo.GetProto()]
			if !ok {
				conn.Close()
				continue
			}

			go handler(streamInfo, conn)
		}
	}(c)

	return nil
}

// StreamHandlerFunc is the type of callbacks executed upon receiving a new stream
// on a given protocol.
type StreamHandlerFunc func(*pb.StreamInfo, io.ReadWriteCloser)

// NewStreamHandler establishes an inbound unix socket and starts a listener.
// All inbound connections to the listener are delegated to the provided
// handler.
func (c *Client) NewStreamHandler(protos []string, handler StreamHandlerFunc) error {
	control, err := c.newControlConn()
	if err != nil {
		return err
	}

	c.mhandlers.Lock()
	defer c.mhandlers.Unlock()

	w := ggio.NewDelimitedWriter(control)
	req := &pb.Request{
		Type: pb.Request_STREAM_HANDLER.Enum(),
		StreamHandler: &pb.StreamHandlerRequest{
			Path:  &c.listenPath,
			Proto: protos,
		},
	}
	if err := w.WriteMsg(req); err != nil {
		return err
	}

	for _, proto := range protos {
		c.handlers[proto] = handler
	}

	return nil
}
