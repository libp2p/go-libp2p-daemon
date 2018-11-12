package test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-daemon/p2pclient"
	pb "github.com/libp2p/go-libp2p-daemon/pb"
)

func TestDHTFindPeer(t *testing.T) {
	daemon, client, closer := createMockDaemonClientPair(t)
	defer closer()
	id := randPeerID(t)

	infoc := make(chan p2pclient.PeerInfo)
	go func() {
		defer close(infoc)
		info, err := client.FindPeer(id)
		if err != nil {
			t.Fatal(err)
		}
		infoc <- info
	}()
	conn := daemon.ExpectConn(t)
	conn.ExpectDHTRequestType(t, pb.DHTRequest_FIND_PEER)
	findPeerResponse := wrapDhtResponse(peerInfoResponse(t, id))
	conn.SendMessage(t, findPeerResponse)
	select {
	case info := <-infoc:
		if info.ID != id {
			t.Fatalf("id %s didn't match expected %s", info.ID, id)
		}
		if len(info.Addrs) != 1 {
			t.Fatalf("expected 1 address, got %d", len(info.Addrs))
		}
		if !bytes.Equal(info.Addrs[0].Bytes(), findPeerResponse.Dht.Peer.Addrs[0]) {
			t.Fatal("address didn't match expected")
		}
	case <-time.After(testTimeout):
		t.Fatal("timed out waiting for peer info")
	}
}

func TestDHTFindPeersConnectedToPeer(t *testing.T) {
	daemon, client, closer := createMockDaemonClientPair(t)
	defer closer()
	ids := randPeerIDs(t, 3)

	infoc := make(chan p2pclient.PeerInfo)
	go func(out chan p2pclient.PeerInfo) {
		infoc, err := client.FindPeersConnectedToPeer(context.Background(), ids[0])
		if err != nil {
			t.Fatal(err)
		}
		for info := range infoc {
			out <- info
		}
		close(out)
	}(infoc)

	conn := daemon.ExpectConn(t)
	req := conn.ExpectDHTRequestType(t, pb.DHTRequest_FIND_PEERS_CONNECTED_TO_PEER)
	if !bytes.Equal(req.GetPeer(), []byte(ids[0])) {
		t.Fatal("request id didn't match expected id")
	}

	resps := make([]*pb.DHTResponse, 2)
	for i := 1; i < 3; i++ {
		resps[i-1] = peerInfoResponse(t, ids[i])
	}
	conn.SendStreamAsync(t, resps)

	i := 0
	for range infoc {
		i++
	}
	if i != 2 {
		t.Fatalf("expected 2 responses, got %d", i)
	}
}

func TestDHTFindProviders(t *testing.T) {
	daemon, client, closer := createMockDaemonClientPair(t)
	defer closer()
	ids := randPeerIDs(t, 3)

	infoc := make(chan p2pclient.PeerInfo)
	contentID := randCid(t)
	go func(out chan p2pclient.PeerInfo) {
		infoc, err := client.FindProviders(context.Background(), contentID)
		if err != nil {
			t.Fatal(err)
		}
		for info := range infoc {
			out <- info
		}
		close(out)
	}(infoc)

	conn := daemon.ExpectConn(t)
	req := conn.ExpectDHTRequestType(t, pb.DHTRequest_FIND_PROVIDERS)
	if !bytes.Equal(req.GetCid(), contentID.Bytes()) {
		t.Fatal("request cid didn't match expected cid")
	}

	resps := make([]*pb.DHTResponse, 2)
	for i := 1; i < 3; i++ {
		resps[i-1] = peerInfoResponse(t, ids[i])
	}
	conn.SendStreamAsync(t, resps)

	i := 0
	for range infoc {
		i++
	}
	if i != 2 {
		t.Fatalf("expected 2 responses, got %d", i)
	}
}
