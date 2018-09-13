package main

import (
	"context"
	"log"

	p2pd "github.com/libp2p/go-libp2p-daemon"
)

func main() {
	_, err := p2pd.NewDaemon(context.Background(), "/tmp/p2pd.sock")
	if err != nil {
		log.Fatal(err)
	}

	select {}
}
