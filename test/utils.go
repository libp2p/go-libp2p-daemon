package test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	p2pd "github.com/libp2p/go-libp2p-daemon"
	"github.com/libp2p/go-libp2p-daemon/p2pclient"
	pb "github.com/libp2p/go-libp2p-daemon/pb"
	peer "github.com/libp2p/go-libp2p-peer"
	peertest "github.com/libp2p/go-libp2p-peer/test"
	ma "github.com/multiformats/go-multiaddr"
)

func createTempDir(t *testing.T) (string, string, func()) {
	root := os.TempDir()
	dir, err := ioutil.TempDir(root, "p2pd")
	if err != nil {
		t.Fatalf("creating temp dir: %s", err)
	}
	daemonPath := dir + "/daemon.sock"
	clientPath := dir + "/client.sock"
	closer := func() {
		os.RemoveAll(dir)
	}
	return daemonPath, clientPath, closer
}

func createDaemon(t *testing.T, daemonPath string) (*p2pd.Daemon, func()) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	daemon, err := p2pd.NewDaemon(ctx, daemonPath)
	if err != nil {
		t.Fatal(err)
	}
	return daemon, cancelCtx
}

func createClient(t *testing.T, daemonPath, clientPath string) (*p2pclient.Client, func()) {
	client, err := p2pclient.NewClient(daemonPath, clientPath)
	if err != nil {
		t.Fatal(err)
	}
	closer := func() {
		client.Close()
	}
	return client, closer
}

func createDaemonClientPair(t *testing.T) (*p2pd.Daemon, *p2pclient.Client, func()) {
	daemonPath, clientPath, dirCloser := createTempDir(t)
	daemon, closeDaemon := createDaemon(t, daemonPath)
	client, closeClient := createClient(t, daemonPath, clientPath)

	closer := func() {
		closeDaemon()
		closeClient()
		dirCloser()
	}
	return daemon, client, closer
}

func createMockDaemonClientPair(t *testing.T) (*mockdaemon, *p2pclient.Client, func()) {
	daemonPath, clientPath, dirCloser := createTempDir(t)
	client, clientCloser := createClient(t, daemonPath, clientPath)
	daemon := newMockDaemon(t, daemonPath, clientPath)
	closer := func() {
		daemon.Close()
		clientCloser()
		dirCloser()
	}
	return daemon, client, closer
}

func randPeerID(t *testing.T) peer.ID {
	id, err := peertest.RandPeerID()
	if err != nil {
		t.Fatalf("peer id: %s", err)
	}
	return id
}

func randPeerIDs(t *testing.T, n int) []peer.ID {
	ids := make([]peer.ID, n)
	for i := 0; i < n; i++ {
		ids[i] = randPeerID(t)
	}
	return ids
}

func wrapDhtResponse(dht *pb.DHTResponse) *pb.Response {
	return &pb.Response{
		Type: pb.Response_OK.Enum(),
		Dht:  dht,
	}
}

func peerInfoResponse(t *testing.T, id peer.ID) *pb.DHTResponse {
	addr, err := ma.NewMultiaddr(fmt.Sprintf("/p2p-circuit/p2p/%s", id.Pretty()))
	if err != nil {
		t.Fatal(err)
	}
	return &pb.DHTResponse{
		Type: pb.DHTResponse_VALUE.Enum(),
		Peer: &pb.PeerInfo{
			Id:    []byte(id),
			Addrs: [][]byte{addr.Bytes()},
		},
	}
}
