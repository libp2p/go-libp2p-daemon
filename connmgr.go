package p2pd

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"

	pb "github.com/libp2p/go-libp2p-daemon/pb"
)

func (d *Daemon) doConnManager(req *pb.Request) *pb.Response {
	if req.ConnManager == nil {
		return errorResponseString("Malformed request; missing parameters")
	}

	switch req.ConnManager.GetType() {
	case pb.ConnManagerRequest_TAG_PEER:
		p, err := peer.IDFromBytes(req.ConnManager.GetPeer())
		if err != nil {
			return errorResponse(err)
		}

		tag := req.ConnManager.GetTag()
		if tag == "" {
			return errorResponseString("Malformed request; missing tag parameter")
		}
		weight := req.ConnManager.GetWeight()

		d.host.ConnManager().TagPeer(p, tag, int(weight))
		return okResponse()

	case pb.ConnManagerRequest_UNTAG_PEER:
		p, err := peer.IDFromBytes(req.ConnManager.GetPeer())
		if err != nil {
			return errorResponse(err)
		}

		tag := req.ConnManager.GetTag()
		if tag == "" {
			return errorResponseString("Malformed request; missing tag parameter")
		}

		d.host.ConnManager().UntagPeer(p, tag)
		return okResponse()

	case pb.ConnManagerRequest_TRIM:
		ctx, cancel := context.WithTimeout(d.ctx, 60*time.Second)
		defer cancel()

		d.host.ConnManager().TrimOpenConns(ctx)
		return okResponse()

	default:
		log.Debugf("unexpected ConnManager request type", "type", req.ConnManager.GetType())
		return errorResponseString("Unexpected request")
	}
}
