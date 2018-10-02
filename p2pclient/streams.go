package p2pclient

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"

	ggio "github.com/gogo/protobuf/io"
	pb "github.com/libp2p/go-libp2p-daemon/pb"
	peer "github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"
)

// StreamInfo wraps the protobuf structure with friendlier types.
type StreamInfo struct {
	Peer  peer.ID
	Addr  ma.Multiaddr
	Proto string
}

func converStreamInfo(info *pb.StreamInfo) (*StreamInfo, error) {
	id, err := peer.IDFromBytes(info.Peer)
	if err != nil {
		return nil, err
	}
	addr, err := ma.NewMultiaddrBytes(info.Addr)
	if err != nil {
		return nil, err
	}
	streamInfo := &StreamInfo{
		Peer:  id,
		Addr:  addr,
		Proto: info.GetProto(),
	}
	return streamInfo, nil
}

type byteReaderConn struct {
	io.Reader
}

func (c *byteReaderConn) ReadByte() (byte, error) {
	b := make([]byte, 1)
	_, err := c.Reader.Read(b)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

func readHeader(r net.Conn) (*bytes.Buffer, error) {
	len, err := binary.ReadUvarint(&byteReaderConn{r})
	if err != nil {
		return nil, err
	}
	outbuf := make([]byte, 8)
	sz := binary.PutUvarint(outbuf, len)
	out := bytes.NewBuffer(outbuf[0:sz])
	n, err := io.CopyN(out, r, int64(len))
	if err != nil {
		return nil, err
	}
	if n != int64(len) {
		return nil, fmt.Errorf("read incorrect number of bytes in header: expected %d, got %d", len, n)
	}
	return out, nil
}

// NewStream initializes a new stream on one of the protocols in protos with
// the specified peer.
func (c *Client) NewStream(peer peer.ID, protos []string) (*StreamInfo, io.ReadWriteCloser, error) {
	control, err := c.newControlConn()
	if err != nil {
		return nil, nil, err
	}
	w := ggio.NewDelimitedWriter(control)

	req := &pb.Request{
		Type: pb.Request_STREAM_OPEN.Enum(),
		StreamOpen: &pb.StreamOpenRequest{
			Peer:  []byte(peer),
			Proto: protos,
		},
	}

	if err = w.WriteMsg(req); err != nil {
		control.Close()
		return nil, nil, err
	}

	headerbuf, err := readHeader(control)
	if err != nil {
		control.Close()
		return nil, nil, err
	}
	r := ggio.NewDelimitedReader(headerbuf, MessageSizeMax)
	res := &pb.Response{}
	if err = r.ReadMsg(res); err != nil {
		control.Close()
		return nil, nil, err
	}

	if err := res.GetError(); err != nil {
		control.Close()
		return nil, nil, errors.New(err.GetMsg())
	}

	info, err := converStreamInfo(res.GetStreamInfo())
	if err != nil {
		control.Close()
		return nil, nil, err
	}

	return info, control, nil
}

// Close stops the listener socket.
func (c *Client) Close() error {
	if c.listener != nil {
		return c.listener.Close()
	}
	return nil
}

func (c *Client) streamDispatcher() {
	for {
		conn, err := c.listener.Accept()
		if err != nil {
			log.Errorf("accepting incoming connection: %s", err)
			return
		}

		headerbuf, err := readHeader(conn)
		if err != nil {
			log.Errorf("reading stream header: %s", err)
			conn.Close()
			continue
		}
		r := ggio.NewDelimitedReader(headerbuf, MessageSizeMax)
		pbStreamInfo := &pb.StreamInfo{}
		if err = r.ReadMsg(pbStreamInfo); err != nil {
			log.Errorf("reading stream info: %s", err)
			conn.Close()
			continue
		}
		streamInfo, err := converStreamInfo(pbStreamInfo)
		if err != nil {
			log.Errorf("parsing stream info: %s", err)
			conn.Close()
			continue
		}

		c.mhandlers.Lock()
		defer c.mhandlers.Unlock()
		handler, ok := c.handlers[streamInfo.Proto]
		if !ok {
			conn.Close()
			continue
		}

		go handler(streamInfo, conn)
	}
}

func (c *Client) listen() error {
	l, err := net.Listen("unix", c.listenPath)
	if err != nil {
		return err
	}

	c.listener = l
	go c.streamDispatcher()

	return nil
}

// StreamHandlerFunc is the type of callbacks executed upon receiving a new stream
// on a given protocol.
type StreamHandlerFunc func(*StreamInfo, io.ReadWriteCloser)

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
