package p2pd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	ps "github.com/libp2p/go-libp2p-pubsub"
	quic "github.com/libp2p/go-libp2p-quic-transport"
)

// DaemonConfig defines the configuration options
type DaemonConfig struct {
	sock                           *string
	quiet                          *bool
	id                             *string
	bootstrap                      *bool
	bootstrapPeers                 *string
	dht                            *bool
	dhtClient                      *bool
	connMgr                        *bool
	connMgrLo                      *int
	connMgrHi                      *int
	connMgrGrace                   *int
	QUIC                           *bool
	natPortMap                     *bool
	pubsub                         *bool
	pubsubRouter                   *string
	pubsubSign                     *bool
	pubsubSignStrict               *bool
	gossipsubHeartbeatInterval     *int
	gossipsubHeartbeatInitialDelay *int
	args                           []string
}

func Initialize() DaemonConfig {
	config := DaemonConfig{
		sock:                           flag.String("sock", "/tmp/p2pd.sock", "daemon control socket path"),
		quiet:                          flag.Bool("q", false, "be quiet"),
		id:                             flag.String("id", "", "peer identity; private key file"),
		bootstrap:                      flag.Bool("b", false, "connects to bootstrap peers and bootstraps the dht if enabled"),
		bootstrapPeers:                 flag.String("bootstrapPeers", "", "comma separated list of bootstrap peers; defaults to the IPFS DHT peers"),
		dht:                            flag.Bool("dht", true, "Enables the DHT in full node mode"),
		dhtClient:                      flag.Bool("dhtClient", true, "Enables the DHT in client mode"),
		connMgr:                        flag.Bool("connManager", false, "Enables the Connection Manager"),
		connMgrLo:                      flag.Int("connLo", 256, "Connection Manager Low Water mark"),
		connMgrHi:                      flag.Int("connHi", 512, "Connection Manager High Water mark"),
		connMgrGrace:                   flag.Int("connGrace", 120, "Connection Manager grace period (in seconds)"),
		QUIC:                           flag.Bool("quic", false, "Enables the QUIC transport"),
		natPortMap:                     flag.Bool("natPortMap", false, "Enables NAT port mapping"),
		pubsub:                         flag.Bool("pubsub", false, "Enables pubsub"),
		pubsubRouter:                   flag.String("pubsubRouter", "gossipsub", "Specifies the pubsub router implementation"),
		pubsubSign:                     flag.Bool("pubsubSign", true, "Enables pubsub message signing"),
		pubsubSignStrict:               flag.Bool("pubsubSignStrict", false, "Enables pubsub strict signature verification"),
		gossipsubHeartbeatInterval:     flag.Int("gossipsubHeartbeatInterval", 0, "Specifies the gossipsub heartbeat interval"),
		gossipsubHeartbeatInitialDelay: flag.Int("gossipsubHeartbeatInitialDelay", 0, "Specifies the gossipsub initial heartbeat delay"),
	}
	flag.Parse()
	config.args = flag.Args()
	// delete control socket if it already exists
	if _, err := os.Stat(*config.sock); !os.IsNotExist(err) {
		err = os.Remove(*config.sock)
		if err != nil {
			log.Fatal(err)
		}
	}
	return config
}

func Start(config DaemonConfig) {
	var opts []libp2p.Option

	if *config.id != "" {
		key, err := ReadIdentity(*config.id)
		if err != nil {
			log.Fatal(err)
		}

		opts = append(opts, libp2p.Identity(key))
	}

	if *config.connMgr {
		cm := connmgr.NewConnManager(*config.connMgrLo, *config.connMgrHi, time.Duration(*config.connMgrGrace))
		opts = append(opts, libp2p.ConnectionManager(cm))
	}

	if *config.QUIC {
		opts = append(opts,
			libp2p.DefaultTransports,
			libp2p.Transport(quic.NewTransport),
			libp2p.ListenAddrStrings(
				"/ip4/0.0.0.0/tcp/0",
				"/ip4/0.0.0.0/udp/0/quic",
				"/ip6/::1/tcp/0",
				"/ip6/::1/udp/0/quic",
			))
	}

	if *config.natPortMap {
		opts = append(opts, libp2p.NATPortMap())
	}

	d, err := NewDaemon(context.Background(), *config.sock, opts...)
	if err != nil {
		log.Fatal(err)
	}

	if *config.pubsub {
		if *config.gossipsubHeartbeatInterval > 0 {
			ps.GossipSubHeartbeatInterval = time.Duration(*config.gossipsubHeartbeatInterval)
		}

		if *config.gossipsubHeartbeatInitialDelay > 0 {
			ps.GossipSubHeartbeatInitialDelay = time.Duration(*config.gossipsubHeartbeatInitialDelay)
		}

		err = d.EnablePubsub(*config.pubsubRouter, *config.pubsubSign, *config.pubsubSignStrict)
		if err != nil {
			log.Fatal(err)
		}
	}

	if *config.dht || *config.dhtClient {
		err = d.EnableDHT(*config.dhtClient)
		if err != nil {
			log.Fatal(err)
		}
	}

	if *config.bootstrapPeers != "" {
		BootstrapPeers = strings.Split(*config.bootstrapPeers, ",")
	}

	if *config.bootstrap {
		err = d.Bootstrap()
		if err != nil {
			log.Fatal(err)
		}
	}

	if !*config.quiet {
		fmt.Printf("Control socket: %s\n", *config.sock)
		fmt.Printf("Peer ID: %s\n", d.ID().Pretty())
		fmt.Printf("Peer Addrs:\n")
		for _, addr := range d.Addrs() {
			fmt.Printf("%s\n", addr.String())
		}
		if *config.bootstrap && *config.bootstrapPeers != "" {
			fmt.Printf("Bootstrap peers:\n")
			for _, p := range BootstrapPeers {
				fmt.Printf("%s\n", p)
			}
		}
	}

	select {}
}

func Stop() {
	p, _ := os.FindProcess(os.Getpid())
	p.Signal(os.Interrupt)
}

func ProcessArgs(args *string) DaemonConfig {
	//replace default config options with configs passed from external source
	argsArray := strings.Split(*args, "|")
	os.Args = argsArray
	//call initialize() to get config
	config := Initialize()
	return config
}
