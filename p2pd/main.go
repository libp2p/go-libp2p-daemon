package main

import (
	"context"
	"fmt"
	"log"

	p2pd "github.com/libp2p/go-libp2p-daemon"
)

func main() {
	sock := "/tmp/p2pd.sock"
	d, err := p2pd.NewDaemon(context.Background(), sock)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Control socket: %s\n", sock)
	fmt.Printf("Peer ID: %s\n", d.ID().Pretty())
	fmt.Printf("Peer Addrs:\n")
	for _, addr := range d.Addrs() {
		fmt.Printf("%s\n", addr.String())
	}

	select {}
}
