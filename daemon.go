package p2pd

import (
	"context"
	"fmt"
	"sync"

	logging "github.com/ipfs/go-log"
	libp2p "github.com/libp2p/go-libp2p"
	discovery "github.com/libp2p/go-libp2p-discovery"
	host "github.com/libp2p/go-libp2p-host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	dhtopts "github.com/libp2p/go-libp2p-kad-dht/opts"
	peer "github.com/libp2p/go-libp2p-peer"
	proto "github.com/libp2p/go-libp2p-protocol"
	ps "github.com/libp2p/go-libp2p-pubsub"
	routing "github.com/libp2p/go-libp2p-routing"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	relay "github.com/libp2p/go-libp2p/p2p/host/relay"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
)

var log = logging.Logger("p2pd")

type Daemon struct {
	ctx      context.Context
	host     host.Host
	listener manet.Listener

	dht    *dht.IpfsDHT
	pubsub *ps.PubSub
	autorelay *relay.AutoRelayHost

	mx sync.Mutex
	// stream handlers: map of protocol.ID to multi-address
	handlers map[proto.ID]ma.Multiaddr
}

func NewDaemon(ctx context.Context, maddr ma.Multiaddr, dhtEnabled bool, dhtClient bool, opts ...libp2p.Option) (*Daemon, error) {
	d := &Daemon{
		ctx: ctx,
		handlers: make(map[proto.ID]ma.Multiaddr),
	}

	if dhtEnabled || dhtClient {
		var dhtOpts []dhtopts.Option
		if dhtClient {
			dhtOpts = append(dhtOpts, dhtopts.Client(true))
		}

		dhtRouting := d.DHTRoutingFactory(dhtOpts)
		opts = append(opts, libp2p.Routing(dhtRouting))
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

	return d, nil
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

func (d *Daemon) EnableAutoRelay() error {
	if d.dht == nil {
		return fmt.Errorf("DHT must be enabled for autorelay")
	}

	disc := discovery.NewRoutingDiscovery(d.dht)
	d.autorelay = relay.NewAutoRelayHost(d.ctx, d.host.(*bhost.BasicHost), disc)
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
