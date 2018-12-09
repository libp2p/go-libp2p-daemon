package p2pclient

import "C"
import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	peer "github.com/libp2p/go-libp2p-peer"
	multiaddr "github.com/multiformats/go-multiaddr"
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

func Initialize() ClientConfig {
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

func Start(config ClientConfig) {

	client, err := NewClient(*config.pathd, *config.pathc)
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
		fmt.Println("Listening...")
		protos := []string{"/test"}
		done := make(chan struct{})
		client.NewStreamHandler(protos, func(info *StreamInfo, conn io.ReadWriteCloser) {
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

func ProcessArgs(args *string) ClientConfig {
	//replace default config options with configs passed from external source
	argsArray := strings.Split(*args, "|")
	os.Args = argsArray
	//call initialize() to get config
	config := Initialize()
	return config
}
