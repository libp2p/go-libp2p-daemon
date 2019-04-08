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
	signal.Notify(ch, syscall.SIGUSR1)
	for {
		select {
		case s := <-ch:
			switch s {
			case syscall.SIGUSR1:
				d.handleSIGUSR1()
			default:
				log.Warningf("unexpected signal %d", s)
			}
		case <-d.ctx.Done():
			return
		}
	}
}

func (d *Daemon) handleSIGUSR1() {
	// this is the state dump signal
	fmt.Println("dht routing table:")
	if d.dht != nil {
		d.dht.RoutingTable().Print()
	}

	fmt.Println("---")
	fmt.Println("connections and streams:")
	for _, c := range d.host.Network().Conns() {
		streams := c.GetStreams()
		protos, _ := d.host.Peerstore().GetProtocols(c.RemotePeer())
		protoVersion, _ := d.host.Peerstore().Get(c.RemotePeer(), "ProtocolVersion")
		agent, _ := d.host.Peerstore().Get(c.RemotePeer(), "AgentVersion")
		fmt.Printf("to=%s; multiaddr=%s; stream_cnt= %d\n, protocols=%v; protoversion=%s; agent=%s\n",
			c.RemotePeer().Pretty(), c.RemoteMultiaddr(), len(streams), protos, protoVersion, agent)
		for _, s := range streams {
			fmt.Println("\tprotocol: ", s.Protocol())
		}
	}
}
