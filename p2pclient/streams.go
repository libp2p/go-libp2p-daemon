package p2pclient

import (
	"errors"
	"fmt"
	"io"
	"net"

	ggio "github.com/gogo/protobuf/io"
	pb "github.com/libp2p/go-libp2p-daemon/pb"
)

// NewStream initializes a new stream on one of the protocols in protos with
// the specified peer.
func (c *Client) NewStream(peer []byte, protos []string) (*pb.StreamInfo, error) {
	r := ggio.NewDelimitedReader(c.control, MessageSizeMax)
	w := ggio.NewDelimitedWriter(c.control)

	req := &pb.Request{
		Type: pb.Request_STREAM_OPEN.Enum(),
		StreamOpen: &pb.StreamOpenRequest{
			Peer:  peer,
			Proto: protos,
		},
	}

	if err := w.WriteMsg(req); err != nil {
		return nil, err
	}

	res := &pb.Response{}
	if err := r.ReadMsg(res); err != nil {
		return nil, err
	}

	if err := res.GetError(); err != nil {
		return nil, errors.New(err.GetMsg())
	}

	return res.GetStreamInfo(), nil
}

// StreamHandlerFunc is the type of callbacks executed upon receiving a new stream
// on a given protocol.
type StreamHandlerFunc func(*pb.StreamInfo, io.ReadWriteCloser)

// NewStreamHandler establishes an inbound unix socket and starts a listener.
// All inbound connections to the listener are delegated to the provided
// handler.
func (c *Client) NewStreamHandler(proto string, path string, handler StreamHandlerFunc) (io.Closer, error) {
	var listener net.Listener
	var err error

	{
		c.mhandlers.Lock()
		defer c.mhandlers.Unlock()

		if _, ok := c.handlers[proto]; ok {
			return nil, fmt.Errorf("handler for protocol %s already registered", proto)
		}

		listener, err = net.Listen("unix", path)
		if err != nil {
			return nil, err
		}

		c.handlers[proto] = listener
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Errorf("error accepting on listener %s: %s", listener.Addr().String(), err)
				return
			}

			r := ggio.NewDelimitedReader(conn, MessageSizeMax)
			info := &pb.StreamInfo{}
			if err := r.ReadMsg(info); err != nil {
				log.Errorf("error parsing stream info: %s", err)
				conn.Close()
			}

			go handler(info, conn)
		}
	}()

	return listener, nil
}
