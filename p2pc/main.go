package main

import "C"
import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	peer "github.com/libp2p/go-libp2p-peer"
	identify "github.com/libp2p/go-libp2p/p2p/protocol/identify"
	multiaddr "github.com/multiformats/go-multiaddr"

	"github.com/libp2p/go-libp2p-daemon/p2pclient/go"
)

// ClientConfig defines the configuration options
type ClientConfig struct {
	pathd   string
	pathc   string
	command string
	args    []string
}

func main() {
	//defined interactions between client and daemon
	commands := [...]string{"Identify", "Connect"}
	pathd := flag.String("pathd", "/tmp/p2pd.sock", "daemon control socket path")
	pathc := flag.String("pathc", "/tmp/p2pc.sock", "client control socket path")
	command := flag.String("command", commands[0], "command to send to the daemon")
	flag.Parse()

	config := ClientConfig{
		pathd:   *pathd,
		pathc:   *pathc,
		command: *command,
		args:    flag.Args(),
	}
	startClient(config)

}
func startClient(config ClientConfig) {
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
