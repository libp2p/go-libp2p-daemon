package p2pd

import (
	"context"
	"net"
	"sync"

	logging "github.com/ipfs/go-log"
	libp2p "github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p-host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	dhtopts "github.com/libp2p/go-libp2p-kad-dht/opts"
	peer "github.com/libp2p/go-libp2p-peer"
	proto "github.com/libp2p/go-libp2p-protocol"
	rhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	ma "github.com/multiformats/go-multiaddr"
)

var log = logging.Logger("p2pd")

type Daemon struct {
	ctx      context.Context
	host     host.Host
	listener net.Listener

	dht *dht.IpfsDHT

	mx sync.Mutex
	// stream handlers: map of protocol.ID to unix socket path
	handlers map[proto.ID]string
}

func NewDaemon(ctx context.Context, path string, opts ...libp2p.Option) (*Daemon, error) {
	h, err := libp2p.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	l, err := net.Listen("unix", path)
	if err != nil {
		h.Close()
		return nil, err
	}

	d := &Daemon{
		ctx:      ctx,
		host:     h,
		listener: l,
		handlers: make(map[proto.ID]string),
	}

	go d.listen()

	return d, nil
}

func (d *Daemon) EnableDHT(client bool) error {
	var opts []dhtopts.Option

	if client {
		opts = append(opts, dhtopts.Client(true))
	}

	dht, err := dht.New(d.ctx, d.host, opts...)
	if err != nil {
		return err
	}

	d.dht = dht
	d.host = rhost.Wrap(d.host, d.dht)

	return nil
}

func (d *Daemon) ID() peer.ID {
	return d.host.ID()
}

func (d *Daemon) Addrs() []ma.Multiaddr {
	return d.host.Addrs()
}

func (d *Daemon) listen() {
	for {
		c, err := d.listener.Accept()
		if err != nil {
			log.Errorf("error accepting connection: %s", err.Error())
		}

		log.Debug("incoming connection")
		go d.handleConn(c)
	}
}
