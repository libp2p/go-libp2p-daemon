package p2pd

import (
	"context"

	pb "github.com/libp2p/go-libp2p-daemon/pb"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"

	cid "github.com/ipfs/go-cid"
)

const defaultProviderCount = 20

func (d *Daemon) doDHT(req *pb.Request) (*pb.Response, <-chan *pb.DHTResponse, func()) {
	if d.dht == nil {
		return errorResponseString("DHT not enabled"), nil, nil
	}

	if req.Dht == nil {
		return errorResponseString("Malformed request; missing parameters"), nil, nil
	}

	switch req.Dht.GetType() {
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
		log.Debugw("unexpected DHT request type", "type", req.Dht.GetType())
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

	return dhtOkResponse(dhtResponsePeerInfo(pi)), nil, nil
}

func (d *Daemon) doDHTFindPeersConnectedToPeer(req *pb.DHTRequest) (*pb.Response, <-chan *pb.DHTResponse, func()) {
	return errorResponseString("not supported"), nil, nil
}

func (d *Daemon) doDHTFindProviders(req *pb.DHTRequest) (*pb.Response, <-chan *pb.DHTResponse, func()) {
	if req.Cid == nil {
		return errorResponseString("Malformed request; missing cid parameter"), nil, nil
	}

	cid, err := cid.Cast(req.Cid)
	if err != nil {
		return errorResponse(err), nil, nil
	}

	count := defaultProviderCount
	if req.GetCount() > 0 {
		count = int(*req.Count)
	}

	ctx, cancel := d.dhtRequestContext(req)

	ch := d.dht.FindProvidersAsync(ctx, cid, count)

	rch := make(chan *pb.DHTResponse)
	go func() {
		defer cancel()
		defer close(rch)
		for pi := range ch {
			select {
			case rch <- dhtResponsePeerInfo(pi):
			case <-ctx.Done():
				return
			}
		}
	}()

	return dhtOkResponse(dhtResponseBegin()), rch, cancel
}

func (d *Daemon) doDHTGetClosestPeers(req *pb.DHTRequest) (*pb.Response, <-chan *pb.DHTResponse, func()) {
	if req.Key == nil {
		return errorResponseString("Malformed request; missing key parameter"), nil, nil
	}

	ctx, cancel := d.dhtRequestContext(req)

	keyString := string(req.Key)
	ch, err := d.dht.GetClosestPeers(ctx, keyString)
	if err != nil {
		cancel()
		return errorResponse(err), nil, nil
	}

	rch := make(chan *pb.DHTResponse)
	go func() {
		defer cancel()
		defer close(rch)
		for _, p := range ch {
			select {
			case rch <- dhtResponsePeerID(p):
			case <-ctx.Done():
				return
			}
		}
	}()

	return dhtOkResponse(dhtResponseBegin()), rch, cancel
}

func (d *Daemon) doDHTGetPublicKey(req *pb.DHTRequest) (*pb.Response, <-chan *pb.DHTResponse, func()) {
	if req.Peer == nil {
		return errorResponseString("Malformed request; missing peer parameter"), nil, nil
	}

	p, err := peer.IDFromBytes(req.Peer)
	if err != nil {
		return errorResponse(err), nil, nil
	}

	ctx, cancel := d.dhtRequestContext(req)
	defer cancel()

	key, err := d.dht.GetPublicKey(ctx, p)
	if err != nil {
		return errorResponse(err), nil, nil
	}

	res, err := dhtResponsePublicKey(key)
	if err != nil {
		return errorResponse(err), nil, nil
	}
	return dhtOkResponse(res), nil, nil
}

func (d *Daemon) doDHTGetValue(req *pb.DHTRequest) (*pb.Response, <-chan *pb.DHTResponse, func()) {
	if req.Key == nil {
		return errorResponseString("Malformed request; missing key parameter"), nil, nil
	}

	ctx, cancel := d.dhtRequestContext(req)
	defer cancel()

	keyString := string(req.Key)
	val, err := d.dht.GetValue(ctx, keyString)
	if err != nil {
		return errorResponse(err), nil, nil
	}

	return dhtOkResponse(dhtResponseValue(val)), nil, nil
}

func (d *Daemon) doDHTSearchValue(req *pb.DHTRequest) (*pb.Response, <-chan *pb.DHTResponse, func()) {
	if req.Key == nil {
		return errorResponseString("Malformed request; missing key parameter"), nil, nil
	}

	ctx, cancel := d.dhtRequestContext(req)

	keyString := string(req.Key)
	ch, err := d.dht.SearchValue(ctx, keyString)
	if err != nil {
		cancel()
		return errorResponse(err), nil, nil
	}

	rch := make(chan *pb.DHTResponse)
	go func() {
		defer cancel()
		defer close(rch)
		for val := range ch {
			select {
			case rch <- dhtResponseValue(val):
			case <-ctx.Done():
				return
			}
		}
	}()

	return dhtOkResponse(dhtResponseBegin()), rch, cancel
}

func (d *Daemon) doDHTPutValue(req *pb.DHTRequest) (*pb.Response, <-chan *pb.DHTResponse, func()) {
	if req.Key == nil {
		return errorResponseString("Malformed request; missing key parameter"), nil, nil
	}

	if req.Value == nil {
		return errorResponseString("Malformed request; missing value parameter"), nil, nil
	}

	ctx, cancel := d.dhtRequestContext(req)
	defer cancel()

	keyString := string(req.Key)
	err := d.dht.PutValue(ctx, keyString, req.Value)
	if err != nil {
		return errorResponse(err), nil, nil
	}

	return okResponse(), nil, nil
}

func (d *Daemon) doDHTProvide(req *pb.DHTRequest) (*pb.Response, <-chan *pb.DHTResponse, func()) {
	if req.Cid == nil {
		return errorResponseString("Malformed request; missing cid parameter"), nil, nil
	}

	cid, err := cid.Cast(req.Cid)
	if err != nil {
		return errorResponse(err), nil, nil
	}

	ctx, cancel := d.dhtRequestContext(req)
	defer cancel()

	err = d.dht.Provide(ctx, cid, true)
	if err != nil {
		return errorResponse(err), nil, nil
	}

	return okResponse(), nil, nil
}

func (d *Daemon) dhtRequestContext(req *pb.DHTRequest) (context.Context, func()) {
	return d.requestContext(req.GetTimeout())
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

func dhtResponsePeerInfo(pi peer.AddrInfo) *pb.DHTResponse {
	return &pb.DHTResponse{
		Type: pb.DHTResponse_VALUE.Enum(),
		Peer: peerInfo2pb(pi),
	}
}

func dhtResponsePeerID(p peer.ID) *pb.DHTResponse {
	return dhtResponseValue([]byte(p))
}

func dhtResponsePublicKey(key crypto.PubKey) (*pb.DHTResponse, error) {
	bytes, err := crypto.MarshalPublicKey(key)
	if err != nil {
		return nil, err
	}
	return dhtResponseValue(bytes), nil
}

func dhtResponseValue(val []byte) *pb.DHTResponse {
	return &pb.DHTResponse{
		Type:  pb.DHTResponse_VALUE.Enum(),
		Value: val,
	}
}

func dhtOkResponse(r *pb.DHTResponse) *pb.Response {
	res := okResponse()
	res.Dht = r
	return res
}

func peerInfo2pb(pi peer.AddrInfo) *pb.PeerInfo {
	addrs := make([][]byte, len(pi.Addrs))
	for x, addr := range pi.Addrs {
		addrs[x] = addr.Bytes()
	}

	return &pb.PeerInfo{
		Id:    []byte(pi.ID),
		Addrs: addrs,
	}
}
