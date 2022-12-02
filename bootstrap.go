package p2pd

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
)

var BootstrapPeers = dht.DefaultBootstrapPeers

const BootstrapConnections = 4

func bootstrapPeerInfo() ([]peer.AddrInfo, error) {
	return peer.AddrInfosFromP2pAddrs(BootstrapPeers...)
}

func shufflePeerInfos(peers []peer.AddrInfo) {
	for i := range peers {
		j := rand.Intn(i + 1)
		peers[i], peers[j] = peers[j], peers[i]
	}
}

func (d *Daemon) Bootstrap() error {
	pis, err := bootstrapPeerInfo()
	if err != nil {
		return err
	}

	for _, pi := range pis {
		d.host.Peerstore().AddAddrs(pi.ID, pi.Addrs, peerstore.PermanentAddrTTL)
	}

	count := d.connectBootstrapPeers(pis, BootstrapConnections)
	if count == 0 {
		return fmt.Errorf("failed to connect to bootstrap peers")
	}

	go d.keepBootstrapConnections(pis)

	if d.dht != nil {
		return d.dht.Bootstrap(d.ctx)
	}

	return nil
}

func (d *Daemon) connectBootstrapPeers(pis []peer.AddrInfo, toconnect int) int {
	count := 0

	shufflePeerInfos(pis)

	ctx, cancel := context.WithTimeout(d.ctx, 60*time.Second)
	defer cancel()

	for _, pi := range pis {
		if d.host.Network().Connectedness(pi.ID) == network.Connected {
			continue
		}
		err := d.host.Connect(ctx, pi)
		if err != nil {
			log.Debugw("Error connecting to bootstrap peer", "peer", pi.ID, "error", err)
		} else {
			d.host.ConnManager().TagPeer(pi.ID, "bootstrap", 1)
			count++
			toconnect--
		}
		if toconnect == 0 {
			break
		}
	}

	return count

}

func (d *Daemon) keepBootstrapConnections(pis []peer.AddrInfo) {
	ticker := time.NewTicker(15 * time.Minute)
	for {
		<-ticker.C

		conns := d.host.Network().Conns()
		if len(conns) >= BootstrapConnections {
			continue
		}

		toconnect := BootstrapConnections - len(conns)
		d.connectBootstrapPeers(pis, toconnect)
	}
}
