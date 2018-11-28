package main

import "C"
import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	peer "github.com/libp2p/go-libp2p-peer"
	identify "github.com/libp2p/go-libp2p/p2p/protocol/identify"
	multiaddr "github.com/multiformats/go-multiaddr"

	"github.com/libp2p/go-libp2p-daemon/p2pclient/go"
)

// ClientConfig defines the configuration options
type ClientConfig struct {
	pathd   *string
	pathc   *string
	command *string
	args    []string
}

type Command int

const (
	Identify         Command = 0
	Connect          Command = 1
	ListenForMessage Command = 2
	SendMessage      Command = 3
)

func (c Command) String() string {
	commands := [...]string{
		"Identify",
		"Connect",
		"ListenForMessage",
		"SendMessage",
	}
	return commands[c]
}

func main() {
	identify.ClientVersion = "p2pc/0.1"
	config := initialize()
	start(config)
}

func initialize() ClientConfig {
	config := ClientConfig{
		pathd:   flag.String("pathd", "/tmp/p2pd.sock", "daemon control socket path"),
		pathc:   flag.String("pathc", "/tmp/p2pc.sock", "client control socket path"),
		command: flag.String("command", "Identify", "command to send to the daemon"),
	}
	flag.Parse()
	config.args = flag.Args()
	// delete control socket if it already exists
	if _, err := os.Stat(*config.pathc); !os.IsNotExist(err) {
		err = os.Remove(*config.pathc)
		if err != nil {
			log.Fatal(err)
		}
	}
	return config
}

func start(config ClientConfig) {

	client, err := p2pclient.NewClient(*config.pathd, *config.pathc)
	defer os.Remove(*config.pathc)

	if err != nil {
		log.Fatal(err)
	}

	switch *config.command {

	case Identify.String():
		id, addrs, err := client.Identify()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Daemon ID: %s\n", id.Pretty())
		fmt.Printf("Peer addresses: %v\n", addrs)

	case Connect.String():
		id, err := peer.IDB58Decode(config.args[0])
		var addrs []multiaddr.Multiaddr
		addrs = make([]multiaddr.Multiaddr, len(config.args[1:]))
		for i, arg := range config.args[1:] {
			addr, _ := multiaddr.NewMultiaddr(arg)
			addrs[i] = addr
		}
		err = client.Connect(id, addrs)
		if err != nil {
			log.Fatal(err)
		}

		pi, err := client.FindPeer(id)
		fmt.Printf("ID: %s has multiaddr: %v", pi.ID, pi.Addrs)

	case ListenForMessage.String():
		protos := []string{"/test"}
		done := make(chan struct{})
		client.NewStreamHandler(protos, func(info *p2pclient.StreamInfo, conn io.ReadWriteCloser) {
			defer conn.Close()
			buf := make([]byte, 1024)
			_, err := conn.Read(buf)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(string(buf))
			done <- struct{}{}
		})
		select {}

	case SendMessage.String():
		protos := []string{"/test"}
		recipientID, err := peer.IDB58Decode(config.args[0])
		_, conn, err := client.NewStream(recipientID, protos)
		if err != nil {
			log.Fatal(err)
		}
		_, err = conn.Write([]byte(config.args[1]))
		if err != nil {
			log.Fatal(err)
		}

	default:

	}
}

//export startClient
func startClient(args *C.char) {
	//replace default config options with configs passed from external source
	argsGoString := C.GoString(args)
	argsArray := strings.Split(argsGoString, "|")
	os.Args = argsArray
	//call initialize() to get config
	config := initialize()
	start(config)
}
