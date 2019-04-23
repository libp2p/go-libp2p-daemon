package p2pd

import (
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p-daemon/config"
	"os"
	"sync"

	logging "github.com/ipfs/go-log"
	libp2p "github.com/libp2p/go-libp2p"
	autonat "github.com/libp2p/go-libp2p-autonat-svc"
	host "github.com/libp2p/go-libp2p-host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	dhtopts "github.com/libp2p/go-libp2p-kad-dht/opts"
	peer "github.com/libp2p/go-libp2p-peer"
	proto "github.com/libp2p/go-libp2p-protocol"
	ps "github.com/libp2p/go-libp2p-pubsub"
	routing "github.com/libp2p/go-libp2p-routing"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
)

var log = logging.Logger("p2pd")

type Daemon struct {
	ctx      context.Context
	host     host.Host
	listener manet.Listener

	dht     *dht.IpfsDHT
	pubsub  *ps.PubSub
	autonat *autonat.AutoNATService

	mx sync.Mutex
	// stream handlers: map of protocol.ID to multi-address
	handlers map[proto.ID]ma.Multiaddr
}

func NewDaemon(ctx context.Context, maddr ma.Multiaddr, dhtMode string, opts ...libp2p.Option) (*Daemon, error) {
	d := &Daemon{
		ctx:      ctx,
		handlers: make(map[proto.ID]ma.Multiaddr),
	}

	if dhtMode != "" {
		var dhtOpts []dhtopts.Option
		if dhtMode == config.DHTClientMode {
			dhtOpts = append(dhtOpts, dhtopts.Client(true))
		}

		opts = append(opts, libp2p.Routing(d.DHTRoutingFactory(dhtOpts)))
	}

	h, err := libp2p.New(ctx, opts...)
	if err != nil {
		return nil, err
	}
	d.host = h

	l, err := manet.Listen(maddr)
	if err != nil {
		h.Close()
		return nil, err
	}
	d.listener = l

	go d.listen()
	go d.trapSignals()

	return d, nil
}

func (d *Daemon) Listener() manet.Listener {
	return d.listener
}

func (d *Daemon) DHTRoutingFactory(opts []dhtopts.Option) func(host.Host) (routing.PeerRouting, error) {
	makeRouting := func(h host.Host) (routing.PeerRouting, error) {
		dhtInst, err := dht.New(d.ctx, h, opts...)
		if err != nil {
			return nil, err
		}
		d.dht = dhtInst
		return dhtInst, nil
	}

	return makeRouting
}

func (d *Daemon) EnablePubsub(router string, sign, strict bool) error {
	var opts []ps.Option

	if sign {
		opts = append(opts, ps.WithMessageSigning(sign))

		if strict {
			opts = append(opts, ps.WithStrictSignatureVerification(strict))
		}
	}

	switch router {
	case "floodsub":
		pubsub, err := ps.NewFloodSub(d.ctx, d.host, opts...)
		if err != nil {
			return err
		}
		d.pubsub = pubsub
		return nil

	case "gossipsub":
		pubsub, err := ps.NewGossipSub(d.ctx, d.host, opts...)
		if err != nil {
			return err
		}
		d.pubsub = pubsub
		return nil

	default:
		return fmt.Errorf("unknown pubsub router: %s", router)
	}

}

func (d *Daemon) EnableAutoNAT(opts ...libp2p.Option) error {
	svc, err := autonat.NewAutoNATService(d.ctx, d.host, opts...)
	d.autonat = svc
	return err
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

func (d *Daemon) Close() error {
	if err := d.host.Close(); err != nil {
		return err
	}

	listenMaddr := d.listener.Multiaddr()
	if err := d.listener.Close(); err != nil {
		return err
	}

	for _, subMaddr := range ma.Split(listenMaddr) {
		if subMaddr.Protocols()[0].Code == ma.P_UNIX {
			socket, err := subMaddr.ValueForProtocol(ma.P_UNIX)
			if err != nil {
				return err
			}
			if err := os.Remove(socket); err != nil {
				return err
			}
		}
	}
	
	return nil
}
