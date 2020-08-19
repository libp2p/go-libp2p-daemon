// +build !windows,!plan9,!nacl,!js

package p2pd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func (d *Daemon) trapSignals() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGUSR1, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case s := <-ch:
			switch s {
			case syscall.SIGUSR1:
				d.handleSIGUSR1()
			case syscall.SIGINT, syscall.SIGTERM:
				d.Close()
				os.Exit(0x80 + int(s.(syscall.Signal)))
			default:
				log.Warnw("uncaught signal", "signal", s)
			}
		case <-d.ctx.Done():
			return
		}
	}
}

func (d *Daemon) handleSIGUSR1() {
	// this is our signal to dump diagnostics info.
	if d.dht != nil {
		fmt.Println("DHT Routing Table:")
		d.dht.RoutingTable().Print()
		fmt.Println()
		fmt.Println()
	}

	conns := d.host.Network().Conns()
	fmt.Printf("Connections and streams (%d):\n", len(conns))

	for _, c := range conns {
		protos, _ := d.host.Peerstore().GetProtocols(c.RemotePeer()) // error value here is useless

		protoVersion, err := d.host.Peerstore().Get(c.RemotePeer(), "ProtocolVersion")
		if err != nil {
			protoVersion = "(unknown)"
		}

		agent, err := d.host.Peerstore().Get(c.RemotePeer(), "AgentVersion")
		if err != nil {
			agent = "(unknown)"
		}

		streams := c.GetStreams()
		fmt.Printf("peer: %s, multiaddr: %s\n", c.RemotePeer().Pretty(), c.RemoteMultiaddr())
		fmt.Printf("\tprotoVersion: %s, agent: %s\n", protoVersion, agent)
		fmt.Printf("\tprotocols: %v\n", protos)
		fmt.Printf("\tstreams (%d):\n", len(streams))
		for _, s := range streams {
			fmt.Println("\t\tprotocol: ", s.Protocol())
		}
	}
}
