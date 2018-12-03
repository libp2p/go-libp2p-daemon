package p2pd

import (
	pb "github.com/libp2p/go-libp2p-daemon/pb"

	ggio "github.com/gogo/protobuf/io"
	ps "github.com/libp2p/go-libp2p-pubsub"
)

func (d *Daemon) doPubsub(req *pb.Request) (*pb.Response, *ps.Subscription) {
	if d.pubsub == nil {
		return errorResponseString("PubSub not enabled"), nil
	}

	if req.Pubsub == nil {
		return errorResponseString("Malformed request; missing parameters"), nil
	}

	switch *req.Pubsub.Type {
	case pb.PSRequest_GET_TOPICS:
		return d.doPubsubGetTopics(req.Pubsub)

	case pb.PSRequest_LIST_PEERS:
		return d.doPubsubListPeers(req.Pubsub)

	case pb.PSRequest_PUBLISH:
		return d.doPubsubPublish(req.Pubsub)

	case pb.PSRequest_SUBSCRIBE:
		return d.doPubsubSubscribe(req.Pubsub)

	default:
		log.Debugf("Unexpected pubsub request type: %d", *req.Pubsub.Type)
		return errorResponseString("Unexpected request"), nil
	}
}

func (d *Daemon) doPubsubGetTopics(req *pb.PSRequest) (*pb.Response, *ps.Subscription) {
	return nil, nil
}

func (d *Daemon) doPubsubListPeers(req *pb.PSRequest) (*pb.Response, *ps.Subscription) {
	return nil, nil
}

func (d *Daemon) doPubsubPublish(req *pb.PSRequest) (*pb.Response, *ps.Subscription) {
	return nil, nil
}

func (d *Daemon) doPubsubSubscribe(req *pb.PSRequest) (*pb.Response, *ps.Subscription) {
	return nil, nil
}

func (d *Daemon) doPubsubPipe(sub *ps.Subscription, r ggio.ReadCloser, w ggio.WriteCloser) {

}
