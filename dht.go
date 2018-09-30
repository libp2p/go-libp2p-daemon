package p2pd

import (
	"errors"

	pb "github.com/libp2p/go-libp2p-daemon/pb"
)

func (d *Daemon) doDHT(req *pb.Request) (*pb.Response, <-chan *pb.DHTResponse, func()) {
	if d.dht == nil {
		return errorResponse(errors.New("DHT not enabled")), nil, nil
	}

	if req.Dht == nil {
		return errorResponse(errors.New("Malformed request; missing parameters")), nil, nil
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
		return errorResponse(errors.New("Unexpected request")), nil, nil
	}
}

func (d *Daemon) doDHTFindPeer(req *pb.DHTRequest) (*pb.Response, <-chan *pb.DHTResponse, func()) {
	return errorResponse(errors.New("XXX Implement me!")), nil, nil
}

func (d *Daemon) doDHTFindPeersConnectedToPeer(req *pb.DHTRequest) (*pb.Response, <-chan *pb.DHTResponse, func()) {
	return errorResponse(errors.New("XXX Implement me!")), nil, nil
}

func (d *Daemon) doDHTFindProviders(req *pb.DHTRequest) (*pb.Response, <-chan *pb.DHTResponse, func()) {
	return errorResponse(errors.New("XXX Implement me!")), nil, nil
}

func (d *Daemon) doDHTGetClosestPeers(req *pb.DHTRequest) (*pb.Response, <-chan *pb.DHTResponse, func()) {
	return errorResponse(errors.New("XXX Implement me!")), nil, nil
}

func (d *Daemon) doDHTGetPublicKey(req *pb.DHTRequest) (*pb.Response, <-chan *pb.DHTResponse, func()) {
	return errorResponse(errors.New("XXX Implement me!")), nil, nil
}

func (d *Daemon) doDHTGetValue(req *pb.DHTRequest) (*pb.Response, <-chan *pb.DHTResponse, func()) {
	return errorResponse(errors.New("XXX Implement me!")), nil, nil
}

func (d *Daemon) doDHTSearchValue(req *pb.DHTRequest) (*pb.Response, <-chan *pb.DHTResponse, func()) {
	return errorResponse(errors.New("XXX Implement me!")), nil, nil
}

func (d *Daemon) doDHTPutValue(req *pb.DHTRequest) (*pb.Response, <-chan *pb.DHTResponse, func()) {
	return errorResponse(errors.New("XXX Implement me!")), nil, nil
}

func (d *Daemon) doDHTProvide(req *pb.DHTRequest) (*pb.Response, <-chan *pb.DHTResponse, func()) {
	return errorResponse(errors.New("XXX Implement me!")), nil, nil
}
