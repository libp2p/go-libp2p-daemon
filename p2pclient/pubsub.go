package p2pclient

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p-core/peer"

	ggio "github.com/gogo/protobuf/io"
	pb "github.com/libp2p/go-libp2p-daemon/pb"
)

func newPubsubReq(req *pb.PSRequest) *pb.Request {
	return &pb.Request{
		Type:   pb.Request_PUBSUB.Enum(),
		Pubsub: req,
	}
}

func (c *Client) doPubsub(psReq *pb.PSRequest) (*pb.PSResponse, error) {
	control, err := c.newControlConn()
	if err != nil {
		return nil, err
	}
	defer control.Close()

	w := ggio.NewDelimitedWriter(control)
	req := newPubsubReq(psReq)
	if err = w.WriteMsg(req); err != nil {
		return nil, err
	}

	r := ggio.NewDelimitedReader(control, MessageSizeMax)
	msg := &pb.Response{}
	if err = r.ReadMsg(msg); err != nil {
		return nil, err
	}

	if msg.GetType() == pb.Response_ERROR {
		err := fmt.Errorf("error from daemon in %s response: %s", req.GetType().String(), msg.GetError())
		log.Errorf(err.Error())
		return nil, err
	}

	return msg.GetPubsub(), nil

}

func (c *Client) streamPubsubRequest(ctx context.Context, psReq *pb.PSRequest) (<-chan *pb.PSMessage, error) {
	control, err := c.newControlConn()
	if err != nil {
		return nil, err
	}

	w := ggio.NewDelimitedWriter(control)
	req := newPubsubReq(psReq)
	if err = w.WriteMsg(req); err != nil {
		control.Close()
		return nil, err
	}

	r := ggio.NewDelimitedReader(control, MessageSizeMax)
	msg := &pb.Response{}
	if err = r.ReadMsg(msg); err != nil {
		control.Close()
		return nil, err
	}

	if msg.GetType() == pb.Response_ERROR {
		err := fmt.Errorf("error from daemon in %s response: %s", req.GetType().String(), msg.GetError())
		log.Errorf(err.Error())
		return nil, err
	}

	go func() {
		<-ctx.Done()
		control.Close()
	}()

	out := make(chan *pb.PSMessage)
	go func() {
		defer close(out)
		defer control.Close()

		for {
			msg := &pb.PSMessage{}
			if err := r.ReadMsg(msg); err != nil {
				log.Errorw("reading pubsub message", "error", err)
				return
			}
			out <- msg
		}
	}()

	return out, nil
}

func (c *Client) GetTopics() ([]string, error) {
	req := &pb.PSRequest{
		Type: pb.PSRequest_GET_TOPICS.Enum(),
	}

	res, err := c.doPubsub(req)
	if err != nil {
		return nil, err
	}

	return res.GetTopics(), nil
}

func (c *Client) ListPeers() ([]peer.ID, error) {
	req := &pb.PSRequest{
		Type: pb.PSRequest_LIST_PEERS.Enum(),
	}

	res, err := c.doPubsub(req)
	if err != nil {
		return nil, err
	}

	ids := make([]peer.ID, len(res.GetPeerIDs()))
	for i, idbytes := range res.GetPeerIDs() {
		id, err := peer.IDFromBytes(idbytes)
		if err != nil {
			return nil, err
		}
		ids[i] = id
	}

	return ids, nil
}

func (c *Client) Publish(topic string, data []byte) error {
	req := &pb.PSRequest{
		Type:  pb.PSRequest_PUBLISH.Enum(),
		Topic: &topic,
		Data:  data,
	}

	_, err := c.doPubsub(req)
	return err
}

func (c *Client) Subscribe(ctx context.Context, topic string) (<-chan *pb.PSMessage, error) {
	req := &pb.PSRequest{
		Type:  pb.PSRequest_SUBSCRIBE.Enum(),
		Topic: &topic,
	}

	return c.streamPubsubRequest(ctx, req)
}
