package test

import (
	"io"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"

	p2pd "github.com/libp2p/go-libp2p-daemon"
	"github.com/libp2p/go-libp2p-daemon/p2pclient"
	ma "github.com/multiformats/go-multiaddr"
)

func TestIdentify(t *testing.T) {
	d, c, closer := createDaemonClientPair(t)
	defer closer()
	cid, caddrs, err := c.Identify()
	if err != nil {
		t.Fatal(err)
	}
	if cid != d.ID() {
		t.Fatal("peer id not equal to query result")
	}
	daddrs := d.Addrs()
	if len(caddrs) != len(daddrs) {
		t.Fatalf("peer addresses different lengths; daemon=%d, client=%d", len(daddrs), len(caddrs))
	}
	addrset := make(map[string]struct{})
	for _, addr := range daddrs {
		addrset[addr.String()] = struct{}{}
	}
	for _, addr := range caddrs {
		if _, ok := addrset[addr.String()]; !ok {
			t.Fatalf("address %s present in client result not present in daemon addresses", addr.String())
		}
	}
}

func connect(c *p2pclient.Client, d *p2pd.Daemon) error {
	return c.Connect(d.ID(), d.Addrs())
}

func TestConnect(t *testing.T) {
	d1, c1, closer1 := createDaemonClientPair(t)
	defer closer1()
	d2, c2, closer2 := createDaemonClientPair(t)
	defer closer2()
	if err := connect(c1, d2); err != nil {
		t.Fatal(err)
	}
	if err := connect(c2, d1); err != nil {
		t.Fatal(err)
	}
	if err := c1.Connect(peer.ID("foobar"), d2.Addrs()); err == nil {
		t.Fatal("expected connection to invalid peer id to fail")
	}
}

func TestConnectFailsOnBadAddress(t *testing.T) {
	_, c1, closer1 := createDaemonClientPair(t)
	defer closer1()
	d2, _, closer2 := createDaemonClientPair(t)
	defer closer2()
	addr, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/4000")
	addrs := []ma.Multiaddr{addr}
	if err := c1.Connect(d2.ID(), addrs); err == nil {
		t.Fatal("expected connection to invalid address to fail")
	}
}

func TestStreams(t *testing.T) {
	d1, c1, closer1 := createDaemonClientPair(t)
	defer closer1()
	d2, c2, closer2 := createDaemonClientPair(t)
	defer closer2()
	if err := connect(c1, d2); err != nil {
		t.Fatal(err)
	}
	testprotos := []string{"/test"}

	done := make(chan struct{})
	err := c1.NewStreamHandler(testprotos, func(info *p2pclient.StreamInfo, conn io.ReadWriteCloser) {
		defer conn.Close()
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		if n != 4 {
			t.Fatal("expected to read 4 bytes")
		}
		if string(buf[0:4]) != "test" {
			t.Fatalf(`expected "test", got "%s"`, string(buf))
		}
		done <- struct{}{}
	})
	if err != nil {
		t.Fatal(err)
	}

	_, conn, err := c2.NewStream(d1.ID(), testprotos)
	if err != nil {
		t.Fatal(err)
	}
	n, err := conn.Write([]byte("test"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 4 {
		t.Fatal("wrote wrong # of bytes")
	}

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for stream result")
	}
	conn.Close()
}
