package p2pclient

import (
	"errors"
	"fmt"
	"io"
	"sync"

	ggio "github.com/gogo/protobuf/io"
	logging "github.com/ipfs/go-log"
	pb "github.com/libp2p/go-libp2p-daemon/pb"
	multiaddr "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
)

var log = logging.Logger("p2pclient")

// MessageSizeMax is cribbed from github.com/libp2p/go-libp2p-net
const MessageSizeMax = 1 << 22 // 4 MB

// Client is the struct that manages a connection to a libp2p daemon.
type Client struct {
	control manet.Conn

	mhandlers sync.Mutex
	handlers  map[string]io.Closer
}

// NewClient creates a new libp2p daemon client, connecting to a daemon
// listening at the provided multiaddr.Multiaddr
func NewClient(addr multiaddr.Multiaddr) (*Client, error) {
	conn, err := manet.Dial(addr)
	if err != nil {
		return nil, err
	}
	cli := &Client{control: conn}
	return cli, nil
}

// Close shuts down the control connection and all active handlers. Users are
// responsible for closing the streams they've opened with the daemon.
func (c *Client) Close() error {
	c.mhandlers.Lock()
	defer c.mhandlers.Unlock()

	if err := c.control.Close(); err != nil {
		return err
	}

	for proto, listener := range c.handlers {
		if err := listener.Close(); err != nil {
			return fmt.Errorf("closing listener for %s: %s", proto, err)
		}
	}

	return nil
}

// Identify queries the daemon for its peer ID and listen addresses.
func (c *Client) Identify() ([]byte, []multiaddr.Multiaddr, error) {
	r := ggio.NewDelimitedReader(c.control, MessageSizeMax)
	w := ggio.NewDelimitedWriter(c.control)

	req := &pb.Request{Type: pb.Request_IDENTIFY.Enum()}
	if err := w.WriteMsg(req); err != nil {
		return nil, nil, err
	}

	res := &pb.Response{}
	if err := r.ReadMsg(res); err != nil {
		return nil, nil, err
	}

	if err := res.GetError(); err != nil {
		return nil, nil, errors.New(err.GetMsg())
	}

	idres := res.GetIdentify()
	addrs := make([]multiaddr.Multiaddr, 0, len(idres.Addrs))
	for i, addrbytes := range idres.Addrs {
		addr, err := multiaddr.NewMultiaddrBytes(addrbytes)
		if err != nil {
			log.Errorf("failed to parse multiaddr in position %d in response to identify request", i)
			continue
		}
		addrs = append(addrs, addr)
	}

	return idres.Id, addrs, nil
}

// Connect establishes a connection to a peer after populating the Peerstore
// entry for said peer with a list of addresses.
func (c *Client) Connect(p []byte, addrs []multiaddr.Multiaddr) error {
	r := ggio.NewDelimitedReader(c.control, MessageSizeMax)
	w := ggio.NewDelimitedWriter(c.control)

	addrbytes := make([][]byte, len(addrs))
	for i, addr := range addrs {
		addrbytes[i] = addr.Bytes()
	}

	req := &pb.Request{
		Type: pb.Request_CONNECT.Enum(),
		Connect: &pb.ConnectRequest{
			Peer:  p,
			Addrs: addrbytes,
		},
	}

	if err := w.WriteMsg(req); err != nil {
		return err
	}

	res := &pb.Response{}
	if err := r.ReadMsg(res); err != nil {
		return err
	}

	if err := res.GetError(); err != nil {
		return errors.New(err.GetMsg())
	}

	return nil
}
