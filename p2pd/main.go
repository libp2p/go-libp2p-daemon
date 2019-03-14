package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	libp2p "github.com/libp2p/go-libp2p"
	relay "github.com/libp2p/go-libp2p-circuit"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	p2pd "github.com/libp2p/go-libp2p-daemon"
	ps "github.com/libp2p/go-libp2p-pubsub"
	quic "github.com/libp2p/go-libp2p-quic-transport"
	identify "github.com/libp2p/go-libp2p/p2p/protocol/identify"
	multiaddr "github.com/multiformats/go-multiaddr"
	"github.com/prometheus/client_golang/prometheus"
)

func main() {
	identify.ClientVersion = "p2pd/0.1"

	maddrString := flag.String("listen", "/unix/tmp/p2pd.sock", "daemon control listen multiaddr")
	quiet := flag.Bool("q", false, "be quiet")
	id := flag.String("id", "", "peer identity; private key file")
	bootstrap := flag.Bool("b", false, "connects to bootstrap peers and bootstraps the dht if enabled")
	bootstrapPeers := flag.String("bootstrapPeers", "", "comma separated list of bootstrap peers; defaults to the IPFS DHT peers")
	dht := flag.Bool("dht", false, "Enables the DHT in full node mode")
	dhtClient := flag.Bool("dhtClient", false, "Enables the DHT in client mode")
	connMgr := flag.Bool("connManager", false, "Enables the Connection Manager")
	connMgrLo := flag.Int("connLo", 256, "Connection Manager Low Water mark")
	connMgrHi := flag.Int("connHi", 512, "Connection Manager High Water mark")
	connMgrGrace := flag.Duration("connGrace", 120, "Connection Manager grace period (in seconds)")
	QUIC := flag.Bool("quic", false, "Enables the QUIC transport")
	natPortMap := flag.Bool("natPortMap", false, "Enables NAT port mapping")
	pubsub := flag.Bool("pubsub", false, "Enables pubsub")
	pubsubRouter := flag.String("pubsubRouter", "gossipsub", "Specifies the pubsub router implementation")
	pubsubSign := flag.Bool("pubsubSign", true, "Enables pubsub message signing")
	pubsubSignStrict := flag.Bool("pubsubSignStrict", false, "Enables pubsub strict signature verification")
	gossipsubHeartbeatInterval := flag.Duration("gossipsubHeartbeatInterval", 0, "Specifies the gossipsub heartbeat interval")
	gossipsubHeartbeatInitialDelay := flag.Duration("gossipsubHeartbeatInitialDelay", 0, "Specifies the gossipsub initial heartbeat delay")
	relayEnabled := flag.Bool("relay", true, "Enables circuit relay")
	relayActive := flag.Bool("relayActive", false, "Enables active mode for relay")
	relayHop := flag.Bool("relayHop", false, "Enables hop for relay")
	relayDiscovery := flag.Bool("relayDiscovery", false, "Enables passive discovery for relay")
	autoRelay := flag.Bool("autoRelay", false, "Enables autorelay")
	autonat := flag.Bool("autonat", false, "Enables the AutoNAT service")
	hostAddrs := flag.String("hostAddrs", "", "comma separated list of multiaddrs the host should listen on")
	announceAddrs := flag.String("announceAddrs", "", "comma separated list of multiaddrs the host should announce to the network")
	noListen := flag.Bool("noListenAddrs", false, "sets the host to listen on no addresses")
	metricsAddr := flag.String("metricsAddr", "", "an address to bind the metrics handler to")
	flag.Parse()

	var opts []libp2p.Option

	maddr, err := multiaddr.NewMultiaddr(*maddrString)
	if err != nil {
		log.Fatal(err)
	}

	if *id != "" {
		key, err := p2pd.ReadIdentity(*id)
		if err != nil {
			log.Fatal(err)
		}

		opts = append(opts, libp2p.Identity(key))
	}

	if *hostAddrs != "" {
		addrs := strings.Split(*hostAddrs, ",")
		opts = append(opts, libp2p.ListenAddrStrings(addrs...))
	}

	if *announceAddrs != "" {
		addrs := strings.Split(*announceAddrs, ",")
		maddrs := make([]multiaddr.Multiaddr, 0, len(addrs))
		for _, a := range addrs {
			maddr, err := multiaddr.NewMultiaddr(a)
			if err != nil {
				log.Fatal(err)
			}
			maddrs = append(maddrs, maddr)
		}
		opts = append(opts, libp2p.AddrsFactory(func([]multiaddr.Multiaddr) []multiaddr.Multiaddr {
			return maddrs
		}))
	}

	if *connMgr {
		cm := connmgr.NewConnManager(*connMgrLo, *connMgrHi, *connMgrGrace)
		opts = append(opts, libp2p.ConnectionManager(cm))
	}

	if *QUIC {
		opts = append(opts,
			libp2p.DefaultTransports,
			libp2p.Transport(quic.NewTransport),
		)

		// if we explicitly specify a transport, we must also explicitly specify the listen addrs
		if *hostAddrs == "" {
			opts = append(opts,
				libp2p.ListenAddrStrings(
					"/ip4/0.0.0.0/tcp/0",
					"/ip4/0.0.0.0/udp/0/quic",
					"/ip6/::1/tcp/0",
					"/ip6/::1/udp/0/quic",
				))
		}
	}

	if *natPortMap {
		opts = append(opts, libp2p.NATPortMap())
	}

	if *relayEnabled {
		var relayOpts []relay.RelayOpt
		if *relayActive {
			relayOpts = append(relayOpts, relay.OptActive)
		}
		if *relayHop {
			relayOpts = append(relayOpts, relay.OptHop)
		}
		if *relayDiscovery {
			relayOpts = append(relayOpts, relay.OptDiscovery)
		}
		opts = append(opts, libp2p.EnableRelay(relayOpts...))
	}

	if *autoRelay {
		if !(*dht || *dhtClient) {
			log.Fatal("DHT must be enabled in order to enable autorelay")
		}
		if !*relayEnabled {
			log.Fatal("Relay must be enabled to enable autorelay")
		}
		opts = append(opts, libp2p.EnableAutoRelay())
	}

	if *noListen {
		opts = append(opts, libp2p.NoListenAddrs)
	}

	d, err := p2pd.NewDaemon(context.Background(), maddr, *dht, *dhtClient, opts...)
	if err != nil {
		log.Fatal(err)
	}

	if *autonat {
		var opts []libp2p.Option
		// allow the AutoNAT service to dial back quic addrs.
		if *QUIC {
			opts = append(opts,
				libp2p.DefaultTransports,
				libp2p.Transport(quic.NewTransport),
			)
		}
		err := d.EnableAutoNAT(opts...)
		if err != nil {
			log.Fatal(err)
		}
	}

	if *pubsub {
		if *gossipsubHeartbeatInterval > 0 {
			ps.GossipSubHeartbeatInterval = *gossipsubHeartbeatInterval
		}

		if *gossipsubHeartbeatInitialDelay > 0 {
			ps.GossipSubHeartbeatInitialDelay = *gossipsubHeartbeatInitialDelay
		}

		err = d.EnablePubsub(*pubsubRouter, *pubsubSign, *pubsubSignStrict)
		if err != nil {
			log.Fatal(err)
		}
	}

	if *bootstrapPeers != "" {
		for _, s := range strings.Split(*bootstrapPeers, ",") {
			ma, err := multiaddr.NewMultiaddr(s)
			if err != nil {
				log.Fatalf("error parsing bootstrap peer %q: %v", s, err)
			}
			p2pd.BootstrapPeers = append(p2pd.BootstrapPeers, ma)
		}
	}

	if *bootstrap {
		err = d.Bootstrap()
		if err != nil {
			log.Fatal(err)
		}
	}

	if !*quiet {
		fmt.Printf("Control socket: %s\n", maddr.String())
		fmt.Printf("Peer ID: %s\n", d.ID().Pretty())
		fmt.Printf("Peer Addrs:\n")
		for _, addr := range d.Addrs() {
			fmt.Printf("%s\n", addr.String())
		}
		if *bootstrap && *bootstrapPeers != "" {
			fmt.Printf("Bootstrap peers:\n")
			for _, p := range p2pd.BootstrapPeers {
				fmt.Printf("%s\n", p)
			}
		}
	}

	if *metricsAddr != "" {
		http.Handle("/metrics", prometheus.Handler())
		go log.Println(http.ListenAndServe(*metricsAddr, nil))
	}

	select {}
}
