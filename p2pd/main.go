package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	p2pd "github.com/libp2p/go-libp2p-daemon"
)

func main() {
	sock := flag.String("sock", "/tmp/p2pd.sock", "daemon control socket path")
	quiet := flag.Bool("q", false, "be quiet")
	flag.Parse()

	d, err := p2pd.NewDaemon(context.Background(), *sock)
	if err != nil {
		log.Fatal(err)
	}

	if !*quiet {
		fmt.Printf("Control socket: %s\n", *sock)
		fmt.Printf("Peer ID: %s\n", d.ID().Pretty())
		fmt.Printf("Peer Addrs:\n")
		for _, addr := range d.Addrs() {
			fmt.Printf("%s\n", addr.String())
		}
	}

	select {}
}
