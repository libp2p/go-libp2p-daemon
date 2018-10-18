package p2pclient

import (
	"context"
	"errors"
	"fmt"
	"net"

	ggio "github.com/gogo/protobuf/io"
	pb "github.com/libp2p/go-libp2p-daemon/pb"
	peer "github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"
)

// PeerInfo wraps the PeerInfo message from our protobuf with richer types.
type PeerInfo struct {
	// Id is the peer's ID
	ID peer.ID
	// Addrs are the peer's listen addresses.
	Addrs []ma.Multiaddr
}

func convertPbPeerInfo(pbi *pb.PeerInfo) (*PeerInfo, error) {
	if pbi == nil {
		return nil, errors.New("null peerinfo")
	}

	id, err := peer.IDFromBytes(pbi.GetId())
	if err != nil {
		return nil, err
	}

	addrs := make([]ma.Multiaddr, 0, len(pbi.Addrs))
	for _, addrbytes := range pbi.Addrs {
		addr, err := ma.NewMultiaddrBytes(addrbytes)
		if err != nil {
			return nil, err
		}
		addrs = append(addrs, addr)
	}

	pi := &PeerInfo{
		ID:    id,
		Addrs: addrs,
	}

	return pi, nil
}

func newDHTReq(req *pb.DHTRequest) *pb.Request {
	return &pb.Request{
		Type: pb.Request_DHT.Enum(),
		Dht:  req,
	}
}

func readDhtResponseStream(ctx context.Context, control net.Conn) <-chan *pb.Response {
	out := make(chan *pb.Response)

	go func() {
		defer close(out)
		defer control.Close()

		r := ggio.NewDelimitedReader(control, MessageSizeMax)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				msg := &pb.Response{}
				if err := r.ReadMsg(msg); err != nil {
					log.Errorf("reading FindPeer response: %s", err)
					return
				}

				out <- msg
				if msg.GetType() == pb.Response_ERROR || msg.Dht.GetType() == pb.DHTResponse_END {
					return
				}
			}
		}
	}()

	return out
}

// FindPeer queries the daemon for a peer's address.
func (c *Client) FindPeer(peer peer.ID) (*PeerInfo, error) {
	control, err := c.newControlConn()
	if err != nil {
		return nil, err
	}

	out := make(chan *PeerInfo, 10)
	defer close(out)
	defer control.Close()

	req := newDHTReq(&pb.DHTRequest{
		Type: pb.DHTRequest_FIND_PEER.Enum(),
		Peer: []byte(peer),
	})

	w := ggio.NewDelimitedWriter(control)
	if err = w.WriteMsg(req); err != nil {
		control.Close()
		return nil, err
	}

	r := ggio.NewDelimitedReader(control, MessageSizeMax)
	msg := &pb.Response{}
	if err = r.ReadMsg(msg); err != nil {
		return nil, err
	}
	if msg.GetType() == pb.Response_ERROR {
		err = fmt.Errorf("error from daemon in findpeer: %s", msg.GetError().GetMsg())
		return nil, err
	}

	dht := msg.GetDht()
	if dht == nil {
		return nil, errors.New("dht response was not populated in findpeer")
	}

	info, err := convertPbPeerInfo(dht.GetPeer())
	if err != nil {
		return nil, err
	}

	return info, nil
}

// FindPeersConnectedToPeer queries the DHT for peers that have an active
// connection to a given peer.
func (c *Client) FindPeersConnectedToPeer(ctx context.Context, peer peer.ID) (<-chan *PeerInfo, error) {
	control, err := c.newControlConn()
	if err != nil {
		return nil, err
	}

	out := make(chan *PeerInfo, 10)
	w := ggio.NewDelimitedWriter(control)
	req := newDHTReq(&pb.DHTRequest{
		Type: pb.DHTRequest_FIND_PEERS_CONNECTED_TO_PEER.Enum(),
		Peer: []byte(peer),
	})

	if err := w.WriteMsg(req); err != nil {
		return nil, err
	}

	go func() {
		defer close(out)
		respc := readDhtResponseStream(ctx, control)

		for resp := range respc {
			if resp.GetType() != pb.Response_OK {
				log.Errorf("error from daemon in findpeersconnectedtopeer: %s", resp.GetError().GetMsg())
				return
			}

			info, err := convertPbPeerInfo(resp.Dht.GetPeer())
			if err != nil {
				continue
			}

			out <- info
		}
	}()

	return out, nil
}
