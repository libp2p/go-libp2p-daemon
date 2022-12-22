package p2pd

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"


	"github.com/cenkalti/backoff/v4"
)

type AddrInfoChan chan peer.AddrInfo

func BeginAutoRelayFeeder(h host.Host, dht *dht.IpfsDHT, cfgPeering Peering, peerChan chan<- peer.AddrInfo) context.CancelFunc {
    ctx, cancel := context.WithCancel(context.Background())
    done := make(chan struct{})

    defer func() {
        if r := recover(); r != nil {
            fmt.Println("Recovering from unexpected error in AutoRelayFeeder:", r)
            debug.PrintStack()
        }
        //TODOYOZH DO WE REALLY NEED THIS?
    }()
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
            select {
            case <-t.C:
            case <-ctx.Done():
                return
            }

            // Always feed trusted IDs (Peering.Peers in the config)
            for _, trustedPeer := range cfgPeering.Peers {
                if len(trustedPeer.Addrs) == 0 {
                    continue
                }
                select {
                case peerChan <- trustedPeer:
                    fmt.Printf("I JUST WROTE TO CHANNEL TRUSTED PEER: %v\n", trustedPeer)
                case <-ctx.Done():
                    return
                }
            }

            // Additionally, feed closest peers discovered via DHT
            if dht == nil {
                /* noop due to missing dht.WAN. happens in some unit tests,//TODO YOZH FIX COMMENTS
                   not worth fixing as we will refactor this after go-libp2p 0.20 */
                continue
            }
            closestPeers, err := dht.GetClosestPeers(ctx, h.ID().String())
            if err != nil {
                // no-op: usually 'failed to find any peer in table' during startup
                continue
            }
            for _, p := range closestPeers {
                addrs := h.Peerstore().Addrs(p)
                if len(addrs) == 0 {
                    continue
                }
                dhtPeer := peer.AddrInfo{ID: p, Addrs: addrs}
                select {
                case peerChan <- dhtPeer:
                    fmt.Printf("I JUST WROTE TO CHANNEL DHT PEER: %v\n", dhtPeer)
                case <-ctx.Done():
                    return
                }
            }
        }
    }()

    return cancel
}

type Peering struct {
	// Peers lists the nodes to attempt to stay connected with.
	Peers []peer.AddrInfo
}

func ConfigureAutoRelay(opts []libp2p.Option, staticRelays []string) ([]libp2p.Option, chan peer.AddrInfo) {
    // note: this requires that the daemon runs autoRelayFeeder in backround
	if len(staticRelays) > 0 {
        if len(staticRelays) > 0 {
            static := make([]peer.AddrInfo, 0, len(staticRelays))
            for _, s := range staticRelays {
                var addr *peer.AddrInfo
                var err error
                addr, err = peer.AddrInfoFromString(s)
                if err != nil {
                    panic(err)
                }
                static = append(static, *addr)
            }
            opts = append(opts, libp2p.EnableAutoRelay(
                autorelay.WithStaticRelays(static),
                autorelay.WithCircuitV1Support(),
            ))
        }
        return opts, nil  // return nil for peerChan because we do not need peer source
	}

	peerChan := make(chan peer.AddrInfo)

    opts = append(opts, libp2p.EnableAutoRelay(
        autorelay.WithPeerSource(func(ctx context.Context, numPeers int) <-chan peer.AddrInfo {
				r := make(chan peer.AddrInfo)
				go func() {
					defer close(r)
					for ; numPeers != 0; numPeers-- {
						select {
						case v, ok := <-peerChan:
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
	return opts, peerChan

}

