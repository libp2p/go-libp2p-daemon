package p2pd

import (
	"context"
	"time"

	pb "github.com/libp2p/go-libp2p-daemon/pb"

	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
)

func (d *Daemon) doDHT(req *pb.Request) (*pb.Response, <-chan *pb.DHTResponse, func()) {
	if d.dht == nil {
		return errorResponseString("DHT not enabled"), nil, nil
	}

	if req.Dht == nil {
		return errorResponseString("Malformed request; missing parameters"), nil, nil
	}

	switch *req.Dht.Type {
	case pb.DHTRequest_FIND_PEER:
		return d.doDHTFindPeer(req.Dht)

	case pb.DHTRequest_FIND_PEERS_CONNECTED_TO_PEER:
		return d.doDHTFindPeersConnectedToPeer(req.Dht)

	case pb.DHTRequest_FIND_PROVIDERS:
		return d.doDHTFindProviders(req.Dht)

	case pb.DHTRequest_GET_CLOSEST_PEERS:
		return d.doDHTGetClosestPeers(req.Dht)

	case pb.DHTRequest_GET_PUBLIC_KEY:
		return d.doDHTGetPublicKey(req.Dht)

	case pb.DHTRequest_GET_VALUE:
		return d.doDHTGetValue(req.Dht)

	case pb.DHTRequest_SEARCH_VALUE:
		return d.doDHTSearchValue(req.Dht)

	case pb.DHTRequest_PUT_VALUE:
		return d.doDHTPutValue(req.Dht)

	case pb.DHTRequest_PROVIDE:
		return d.doDHTProvide(req.Dht)

	default:
		log.Debugf("Unexpected DHT request type: %d", *req.Dht.Type)
		return errorResponseString("Unexpected request"), nil, nil
	}
}

func (d *Daemon) doDHTFindPeer(req *pb.DHTRequest) (*pb.Response, <-chan *pb.DHTResponse, func()) {
	if req.Peer == nil {
		return errorResponseString("Malformed request; missing peer parameter"), nil, nil
	}

	p, err := peer.IDFromBytes(req.Peer)
	if err != nil {
		return errorResponse(err), nil, nil
	}

	ctx, cancel := d.dhtRequestContext(req)
	defer cancel()

	pi, err := d.dht.FindPeer(ctx, p)
	if err != nil {
		return errorResponse(err), nil, nil
	}

	return dhtOkResponse(dhtResponsePeer(pi)), nil, nil
}

func (d *Daemon) doDHTFindPeersConnectedToPeer(req *pb.DHTRequest) (*pb.Response, <-chan *pb.DHTResponse, func()) {
	if req.Peer == nil {
		return errorResponseString("Malformed request; missing peer parameter"), nil, nil
	}

	p, err := peer.IDFromBytes(req.Peer)
	if err != nil {
		return errorResponse(err), nil, nil
	}

	ctx, cancel := d.dhtRequestContext(req)

	ch, err := d.dht.FindPeersConnectedToPeer(ctx, p)
	if err != nil {
		cancel()
		return errorResponse(err), nil, nil
	}

	rch := make(chan *pb.DHTResponse)
	go func() {
		defer cancel()
		defer close(rch)
		for pi := range ch {
			select {
			case rch <- dhtResponsePeer(*pi):
			case <-ctx.Done():
				return
			}
		}
	}()

	return dhtOkResponse(dhtResponseBegin()), rch, cancel
}

func (d *Daemon) doDHTFindProviders(req *pb.DHTRequest) (*pb.Response, <-chan *pb.DHTResponse, func()) {
	return errorResponseString("XXX Implement me!"), nil, nil
}

func (d *Daemon) doDHTGetClosestPeers(req *pb.DHTRequest) (*pb.Response, <-chan *pb.DHTResponse, func()) {
	return errorResponseString("XXX Implement me!"), nil, nil
}

func (d *Daemon) doDHTGetPublicKey(req *pb.DHTRequest) (*pb.Response, <-chan *pb.DHTResponse, func()) {
	return errorResponseString("XXX Implement me!"), nil, nil
}

func (d *Daemon) doDHTGetValue(req *pb.DHTRequest) (*pb.Response, <-chan *pb.DHTResponse, func()) {
	return errorResponseString("XXX Implement me!"), nil, nil
}

func (d *Daemon) doDHTSearchValue(req *pb.DHTRequest) (*pb.Response, <-chan *pb.DHTResponse, func()) {
	return errorResponseString("XXX Implement me!"), nil, nil
}

func (d *Daemon) doDHTPutValue(req *pb.DHTRequest) (*pb.Response, <-chan *pb.DHTResponse, func()) {
	return errorResponseString("XXX Implement me!"), nil, nil
}

func (d *Daemon) doDHTProvide(req *pb.DHTRequest) (*pb.Response, <-chan *pb.DHTResponse, func()) {
	return errorResponseString("XXX Implement me!"), nil, nil
}

func (d *Daemon) dhtRequestContext(req *pb.DHTRequest) (context.Context, func()) {
	timeout := 60 * time.Second
	if req.GetTimeout() > 0 {
		timeout = time.Duration(*req.Timeout)
	}

	return context.WithTimeout(d.ctx, timeout)
}

func dhtResponseBegin() *pb.DHTResponse {
	return &pb.DHTResponse{
		Type: pb.DHTResponse_BEGIN.Enum(),
	}
}

func dhtResponseEnd() *pb.DHTResponse {
	return &pb.DHTResponse{
		Type: pb.DHTResponse_END.Enum(),
	}
}

func dhtResponsePeer(pi pstore.PeerInfo) *pb.DHTResponse {
	return &pb.DHTResponse{
		Type: pb.DHTResponse_VALUE.Enum(),
		Peer: peerInfo2pb(pi),
	}
}

func dhtOkResponse(r *pb.DHTResponse) *pb.Response {
	res := okResponse()
	res.Dht = r
	return res
}

func peerInfo2pb(pi pstore.PeerInfo) *pb.PeerInfo {
	addrs := make([][]byte, len(pi.Addrs))
	for x, addr := range pi.Addrs {
		addrs[x] = addr.Bytes()
	}

	return &pb.PeerInfo{
		Id:    []byte(pi.ID),
		Addrs: addrs,
	}
}
