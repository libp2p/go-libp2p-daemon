package p2pd

// This file is based on https://github.com/ipfs/kubo/blob/master/core/node/libp2p/relay.go

import (
	"context"
	"runtime/debug"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"

	"github.com/cenkalti/backoff/v4"
)

func parseRelays(addrStrings []string) []peer.AddrInfo {
	addrs := make([]peer.AddrInfo, 0, len(addrStrings))
	for _, s := range addrStrings {
		var addr *peer.AddrInfo
		var err error
		addr, err = peer.AddrInfoFromString(s)
		if err != nil {
			panic(err)
		}
		addrs = append(addrs, *addr)
	}
	return addrs
}

func MaybeConfigureAutoRelay(opts []libp2p.Option, relayDiscovery bool, trustedRelays []string) ([]libp2p.Option, chan peer.AddrInfo) {
	var peerSourceChan chan peer.AddrInfo // default(nil) means no peerSource

	if !relayDiscovery && len(trustedRelays) > 0 {
		log.Debugf("Running with static relays only: %v\n", trustedRelays)
		// static relays, no automatic discovery
		opts = append(opts, libp2p.EnableAutoRelay(
			autorelay.WithStaticRelays(parseRelays(trustedRelays)),
			autorelay.WithCircuitV1Support(),
		))
	} else if relayDiscovery {
		log.Debug("Running with automatic relay discovery\n")
		peerSourceChan = make(chan peer.AddrInfo)
		// requires daemon to BeginRelayDiscovery once it is initialized
		opts = append(opts, libp2p.EnableAutoRelay(
			autorelay.WithPeerSource(func(ctx context.Context, numPeers int) <-chan peer.AddrInfo {
				r := make(chan peer.AddrInfo)
				go func() {
					defer close(r)
					for ; numPeers != 0; numPeers-- {
						select {
						case v, ok := <-peerSourceChan:
							if !ok {
								return
							}
							select {
							case r <- v:
							case <-ctx.Done():
								return
							}
						case <-ctx.Done():
							return
						}
					}
				}()
				return r
			}, 0)))
	} else {
		log.Debug("Running without autorelay\n")
	}
	return opts, peerSourceChan
}

func BeginRelayDiscovery(h host.Host, dht *dht.IpfsDHT, trustedRelays []string, peerSourceChan chan<- peer.AddrInfo) context.CancelFunc {
	log.Debug("Began looking for potential relays in background\n")
	var trustedRelayAddrs = parseRelays(trustedRelays)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)

		// Feed peers more often right after the bootstrap, then backoff
		bo := backoff.NewExponentialBackOff()
		bo.InitialInterval = 15 * time.Second
		bo.Multiplier = 3
		bo.MaxInterval = 1 * time.Hour
		bo.MaxElapsedTime = 0 // never stop
		t := backoff.NewTicker(bo)
		defer t.Stop()
		for {
			func() { // gather peers once
				defer func() { // recover from errors
					if r := recover(); r != nil {
						log.Warnw("Recovering from unexpected error in AutoRelayFeeder,", "caught", r)
						debug.PrintStack()
					}
				}()

				// Always feed trusted IDs (Peering.Peers in the config)
				for _, trustedPeer := range trustedRelayAddrs {
					if len(trustedPeer.Addrs) == 0 {
						continue
					}
					select {
					case peerSourceChan <- trustedPeer:
						log.Debugf("Trying trusted peer as relay: %v\n", trustedPeer)
					case <-ctx.Done():
						return
					}
				}

				// Additionally, feed closest peers discovered via DHT
				if dht == nil {
					panic("Daemon asked to perform relay discovery but has not DHT. Please set -dht=1")
				}

				closestPeers, err := dht.GetClosestPeers(ctx, h.ID().String())
				if err != nil {
					// no-op: usually 'failed to find any peer in table' during startup
					return
				}
				for _, p := range closestPeers {
					addrs := h.Peerstore().Addrs(p)
					if len(addrs) == 0 {
						continue
					}
					dhtPeer := peer.AddrInfo{ID: p, Addrs: addrs}
					select {
					case peerSourceChan <- dhtPeer:
						log.Debugf("Trying dht peer as relay: %v\n", dhtPeer)
					case <-ctx.Done():
						return
					}
				}
			}()

			select {
			case <-t.C:
			case <-ctx.Done():
				return
			}
		}
	}()

	return cancel
}
