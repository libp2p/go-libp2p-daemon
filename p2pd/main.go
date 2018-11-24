package main

import "C"
import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	libp2p "github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	p2pd "github.com/libp2p/go-libp2p-daemon"
	peer "github.com/libp2p/go-libp2p-peer"
	ps "github.com/libp2p/go-libp2p-pubsub"
	quic "github.com/libp2p/go-libp2p-quic-transport"
	identify "github.com/libp2p/go-libp2p/p2p/protocol/identify"
	multiaddr "github.com/multiformats/go-multiaddr"

	"github.com/libp2p/go-libp2p-daemon/p2pclient/go"
)

// ClientConfig defines the configuration options for the client
type ClientConfig struct {
	pathd   string
	pathc   string
	command string
	args    []string
}

func main() {
	//defined interactions between client and daemon
	commands := [...]string{"Identify", "Connect"}
	client := flag.Bool("client", false, "run in client mode")
	pathd := flag.String("pathd", "/tmp/p2pd.sock", "daemon control socket path")
	pathc := flag.String("pathc", "/tmp/p2pc.sock", "client control socket path")
	command := flag.String("command", commands[0], "command to send to the daemon")
	flag.Parse()

	if *client {
		config := ClientConfig{
			pathd:   *pathd,
			pathc:   *pathc,
			command: *command,
			args:    flag.Args(),
		}
		startC(config)
	} else {
		startD()
	}
}
func startC(config ClientConfig) {
	identify.ClientVersion = "p2pc/0.1"

	client, err := p2pclient.NewClient(config.pathd, config.pathc)

	if err != nil {
		log.Fatal(err)
	}

	if config.command == "Identify" {
		id, addrs, err := client.Identify()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Daemon ID: %s\n", id.Pretty())
		fmt.Printf("Peer addresses: %v\n", addrs)
	} else if config.command == "Connect" {
		id, err := peer.IDB58Decode(config.args[0])
		var addrs []multiaddr.Multiaddr
		addrs = make([]multiaddr.Multiaddr, len(config.args[1:]))
		for i, arg := range config.args[1:] {
			addr, _ := multiaddr.NewMultiaddr(arg)
			addrs[i] = addr
		}
		err = client.Connect(id, addrs)
		if err != nil {
			fmt.Println(err)
		}

		pi, err := client.FindPeer(id)
		fmt.Printf("ID: %s has multiaddr: %v", pi.ID, pi.Addrs)

	} else if config.command == "ListenForMessage" {
		protos := []string{"/test"}
		done := make(chan struct{})
		client.NewStreamHandler(protos, func(info *p2pclient.StreamInfo, conn io.ReadWriteCloser) {
			defer conn.Close()
			buf := make([]byte, 1024)
			_, err := conn.Read(buf)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println(string(buf))
			done <- struct{}{}
		})
		select {}
	} else if config.command == "SendMessage" {
		protos := []string{"/test"}
		recipientID, err := peer.IDB58Decode(config.args[0])
		_, conn, err := client.NewStream(recipientID, protos)
		if err != nil {
			fmt.Println(err)
		}
		_, err = conn.Write([]byte(config.args[1]))
		if err != nil {
			fmt.Println(err)
		}

	}

	os.Remove(config.pathc)
}

//export startD
func startD() {
	identify.ClientVersion = "p2pd/0.1"

	sock := flag.String("sock", "/tmp/p2pd.sock", "daemon control socket path")
	quiet := flag.Bool("q", false, "be quiet")
	id := flag.String("id", "", "peer identity; private key file")
	bootstrap := flag.Bool("b", false, "connects to bootstrap peers and bootstraps the dht if enabled")
	bootstrapPeers := flag.String("bootstrapPeers", "", "comma separated list of bootstrap peers; defaults to the IPFS DHT peers")
	dht := flag.Bool("dht", true, "Enables the DHT in full node mode")
	dhtClient := flag.Bool("dhtClient", true, "Enables the DHT in client mode")
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
	flag.Parse()

	var opts []libp2p.Option

	if *id != "" {
		key, err := p2pd.ReadIdentity(*id)
		if err != nil {
			log.Fatal(err)
		}

		opts = append(opts, libp2p.Identity(key))
	}

	if *connMgr {
		cm := connmgr.NewConnManager(*connMgrLo, *connMgrHi, *connMgrGrace)
		opts = append(opts, libp2p.ConnectionManager(cm))
	}

	if *QUIC {
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

	if *natPortMap {
		opts = append(opts, libp2p.NATPortMap())
	}

	d, err := p2pd.NewDaemon(context.Background(), *sock, opts...)
	if err != nil {
		log.Fatal(err)
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

	if *dht || *dhtClient {
		err = d.EnableDHT(*dhtClient)
		if err != nil {
			log.Fatal(err)
		}
	}

	if *bootstrapPeers != "" {
		p2pd.BootstrapPeers = strings.Split(*bootstrapPeers, ",")
	}

	if *bootstrap {
		err = d.Bootstrap()
		if err != nil {
			log.Fatal(err)
		}
	}

	if !*quiet {
		fmt.Printf("Control socket: %s\n", *sock)
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

	select {}
}

//export stopD
func stopD() {
	p, _ := os.FindProcess(os.Getpid())
	p.Signal(os.Interrupt)
}
