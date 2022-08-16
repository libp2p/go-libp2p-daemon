package test

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

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
	err := connect(c1, d2)
	require.NoError(t, err)

	testprotos := []string{"/test"}
	done := make(chan struct{})
	err = c1.NewStreamHandler(testprotos, makeExpectStringHandler(t, done), false)
	require.NoError(t, err)

	err = callExpectStringHandler(c2, d1.ID(), testprotos, done)
	require.NoError(t, err)
}

func TestRemovingStreams(t *testing.T) {
	d1, c1, cancel1 := createDaemonClientPair(t)
	c2maddr, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/0")
	require.NoError(t, err)
	c2, cancel2 := createClient(t, d1.Listener().Multiaddr(), c2maddr)

	_, c3, cancel3 := createDaemonClientPair(t)

	defer func() {
		cancel1()
		cancel2()
		cancel3()
	}()

	peer1ID, peer1Addrs, err := c1.Identify()
	require.NoError(t, err)
	err = c3.Connect(peer1ID, peer1Addrs)
	require.NoError(t, err)

	testprotos := []string{"/test"}
	done := make(chan struct{})
	err = c1.NewStreamHandler(testprotos, makeExpectStringHandler(t, done), true)
	require.NoError(t, err)
	err = c2.NewStreamHandler(testprotos, makeExpectStringHandler(t, done), true)
	require.NoError(t, err)
	err = callExpectStringHandler(c3, peer1ID, testprotos, done)
	require.NoError(t, err)

	err = c1.RemoveStreamHandler(testprotos)
	require.NoError(t, err)
	err = callExpectStringHandler(c3, peer1ID, testprotos, done)
	require.NoError(t, err, "The handler was removed only on the 1st client, the 2nd client should respond")

	err = c2.RemoveStreamHandler(testprotos)
	require.NoError(t, err)
	err = callExpectStringHandler(c3, peer1ID, testprotos, done)
	require.Error(t, err, "Calling a handler removed on all clients should return an error")
}

func makeExpectStringHandler(t *testing.T, done chan struct{}) p2pclient.StreamHandlerFunc {
	return func(info *p2pclient.StreamInfo, conn io.ReadWriteCloser) {
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
	}
}

func callExpectStringHandler(client *p2pclient.Client, peerID peer.ID, testprotos []string, done chan struct{}) error {
	_, conn, err := client.NewStream(peerID, testprotos)
	if err != nil {
		return err
	}
	defer conn.Close()

	n, err := conn.Write([]byte("test"))
	if err != nil {
		return err
	}
	if n != 4 {
		return errors.New("wrote wrong # of bytes")
	}

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		return errors.New("timed out waiting for stream result")
	}
	return nil
}

func TestBalancedStreams(t *testing.T) {
	handlerDaemon, handlerClient1, closer1 := createDaemonClientPair(t)
	defer closer1()
	_, cmaddr, dirCloser := getEndpointsMaker(t)(t)
	handlerClient2, closer2 := createClient(t, handlerDaemon.Listener().Multiaddr(), cmaddr)
	defer func() {
		closer2()
		dirCloser()
	}()
	_, callerClient, callerCloser := createDaemonClientPair(t)
	defer callerCloser()

	if err := connect(callerClient, handlerDaemon); err != nil {
		t.Fatal(err)
	}

	testprotos := []string{"/test"}

	done := make(chan int)
	makeHandler := func(x int) p2pclient.StreamHandlerFunc {
		return func(info *p2pclient.StreamInfo, conn io.ReadWriteCloser) {
			defer conn.Close()
			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			if err != nil {
				t.Fatal(err)
			}
			if n != 4 {
				t.Fatalf("expected to read 4 bytes, %d: %s", n, string(buf))
			}
			if string(buf[0:4]) != "test" {
				t.Fatalf(`expected "test", got "%s"`, string(buf))
			}
			time.Sleep(50 * time.Millisecond)
			done <- x
		}
	}
	err := handlerClient1.NewStreamHandler(testprotos, makeHandler(1), true)
	if err != nil {
		t.Fatal(err)
	}
	err = handlerClient2.NewStreamHandler(testprotos, makeHandler(-1), true)
	if err != nil {
		t.Fatal(err)
	}

	control := 0
	for i := 0; i < 10; i++ {
		_, conn, err := callerClient.NewStream(handlerDaemon.ID(), testprotos)
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
		conn.Close()
	}

	for i := 0; i < 10; i++ {
		select {
		case x := <-done:
			control += x
		case <-time.After(1 * time.Second):
			t.Fatal("timed out waiting for stream result")
		}
	}
	if control != 0 {
		t.Fatalf("daemon did not balanced handlers %d", control)
	}
}
