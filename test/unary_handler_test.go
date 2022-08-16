package test

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-daemon/p2pclient"
	ma "github.com/multiformats/go-multiaddr"
)

func TestConcurrentCalls(t *testing.T) {
	_, p1, cancel1 := createDaemonClientPair(t)
	_, p2, cancel2 := createDaemonClientPair(t)

	defer func() {
		cancel1()
		cancel2()
	}()

	peer1ID, peer1Addrs, err := p1.Identify()
	if err != nil {
		t.Fatal(err)
	}
	if err := p2.Connect(peer1ID, peer1Addrs); err != nil {
		t.Fatal(err)
	}

	var proto protocol.ID = "sqrt"
	if err := p1.AddUnaryHandler(proto, sqrtHandler, false); err != nil {
		t.Fatal(err)
	}

	count := 100

	var wg sync.WaitGroup
	var m sync.Map
	wg.Add(count)

	for i := 0; i < count; i++ {
		go func(i int) {
			defer wg.Done()

			reply, err := p2.CallUnaryHandler(context.Background(), peer1ID, proto, float64Bytes(float64(i)))
			if err != nil {
				panic(err)
			}

			if _, loaded := m.LoadOrStore(float64FromBytes(reply), ""); loaded {
				panic(err)
			}
		}(i)
	}

	wg.Wait()
}

func TestUnaryCalls(t *testing.T) {
	_, p1, cancel1 := createDaemonClientPair(t)
	_, p2, cancel2 := createDaemonClientPair(t)

	defer func() {
		cancel1()
		cancel2()
	}()

	peer1ID, peer1Addrs, err := p1.Identify()
	if err != nil {
		t.Fatal(err)
	}
	if err := p2.Connect(peer1ID, peer1Addrs); err != nil {
		t.Fatal(err)
	}

	var proto protocol.ID = "sqrt"
	if err := p1.AddUnaryHandler(proto, sqrtHandler, false); err != nil {
		t.Fatal(err)
	}

	t.Run(
		"test bad request",
		func(t *testing.T) {
			var handlerError *p2pclient.P2PHandlerError

			_, err := p2.CallUnaryHandler(context.Background(), peer1ID, proto, float64Bytes(-64))
			if !errors.As(err, &handlerError) {
				t.Fatal("remote should have returned error")
			}
			t.Logf("remote correctly returned error: '%v'\n", err)
		},
	)

	t.Run(
		"test correct request",
		func(t *testing.T) {
			reply, err := p2.CallUnaryHandler(context.Background(), peer1ID, proto, float64Bytes(64))
			if err != nil {
				t.Fatal(err)
			}
			result := float64FromBytes(reply)
			expected := math.Sqrt(64)

			if !almostEqual(result, expected) {
				t.Fatalf("remote returned unexpected result: %.2f != %.2f", result, expected)
			}

			t.Logf("remote returned: %f\n", result)
		},
	)

	t.Run(
		"test bad proto",
		func(t *testing.T) {
			var daemonError *p2pclient.DaemonError

			_, err := p2.CallUnaryHandler(context.Background(), peer1ID, "bad proto", make([]byte, 0))
			if !errors.As(err, &daemonError) {
				t.Fatal("expected error")
			}
			t.Logf("remote correctly returned error: '%v'\n", err)
		},
	)
}

func TestCancellation(t *testing.T) {
	_, p1, cancel1 := createDaemonClientPair(t)
	_, p2, cancel2 := createDaemonClientPair(t)

	t.Cleanup(func() {
		cancel1()
		cancel2()
	})

	peer1ID, peer1Addrs, err := p1.Identify()
	if err != nil {
		t.Fatal(err)
	}
	if err := p2.Connect(peer1ID, peer1Addrs); err != nil {
		t.Fatal(err)
	}

	var proto protocol.ID = "slow"
	if err := p1.AddUnaryHandler(proto, slowHandler, false); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = p2.CallUnaryHandler(ctx, peer1ID, proto, []byte("hi"))
	if err == nil {
		t.Fatal("handler is expected to be cancelled but finished successfully")
	}
}

func TestAddUnaryHandler(t *testing.T) {
	// create a single daemon and connect two clients to it
	dmaddr, c1maddr, dir1Closer := getEndpointsMaker(t)(t)
	_, c2maddr, dir2Closer := getEndpointsMaker(t)(t)

	daemon, closeDaemon := createDaemon(t, dmaddr)

	c1, closeClient1 := createClient(t, daemon.Listener().Multiaddr(), c1maddr)
	c2, closeClient2 := createClient(t, daemon.Listener().Multiaddr(), c2maddr)

	defer func() {
		closeClient1()
		closeClient2()

		closeDaemon()

		dir1Closer()
		dir2Closer()
	}()

	var proto protocol.ID = "sqrt"
	err := c1.AddUnaryHandler(proto, sqrtHandler, false)
	require.NoError(t, err)
	err = c2.AddUnaryHandler(proto, sqrtHandler, false)
	require.Error(t, err, "adding second unary handler with same name should have returned error")

	err = c1.Close()
	require.NoError(t, err)

	time.Sleep(time.Second)

	err = c2.AddUnaryHandler(proto, sqrtHandler, false)
	require.NoError(t, err, "closing client 1 should have cleaned up the proto list")
}

func TestRemoveUnaryHandler(t *testing.T) {
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

	var proto protocol.ID = "sqrt"
	err = c1.AddUnaryHandler(proto, sqrtHandler, true)
	require.NoError(t, err)
	err = c2.AddUnaryHandler(proto, sqrtHandler, true)
	require.NoError(t, err)
	_, err = c3.CallUnaryHandler(context.Background(), peer1ID, proto, float64Bytes(4))
	require.NoError(t, err)

	err = c1.RemoveUnaryHandler(proto)
	require.NoError(t, err)
	_, err = c3.CallUnaryHandler(context.Background(), peer1ID, proto, float64Bytes(4))
	require.NoError(t, err, "The handler was removed only on the 1st client, the 2nd client should respond")

	err = c2.RemoveUnaryHandler(proto)
	require.NoError(t, err)
	_, err = c3.CallUnaryHandler(context.Background(), peer1ID, proto, float64Bytes(4))
	require.Error(t, err, "Calling a handler removed on all clients should return an error")
}

func TestBalancedCall(t *testing.T) {
	dmaddr, c1maddr, dir1Closer := getEndpointsMaker(t)(t)
	_, c2maddr, dir2Closer := getEndpointsMaker(t)(t)

	handlerDaemon, closeDaemon := createDaemon(t, dmaddr)

	handlerClient1, closeClient1 := createClient(t, handlerDaemon.Listener().Multiaddr(), c1maddr)
	handlerClient2, closeClient2 := createClient(t, handlerDaemon.Listener().Multiaddr(), c2maddr)
	_, callerClient, callerClose := createDaemonClientPair(t)
	defer func() {
		closeClient1()
		closeClient2()

		closeDaemon()

		dir1Closer()
		dir2Closer()
		callerClose()
	}()

	if err := callerClient.Connect(handlerDaemon.ID(), handlerDaemon.Addrs()); err != nil {
		t.Fatal(err)
	}

	var proto protocol.ID = "test"
	done := make(chan int, 10)

	if err := handlerClient1.AddUnaryHandler(proto, getNumberedHandler(done, 1, t), true); err != nil {
		t.Fatal(err)
	}

	if err := handlerClient2.AddUnaryHandler(proto, getNumberedHandler(done, -1, t), true); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		_, err := callerClient.CallUnaryHandler(context.Background(), handlerDaemon.ID(), proto, []byte("test"))
		if err != nil {
			t.Fatal(err.Error())
		}
	}

	control := 0

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

func float64FromBytes(bytes []byte) float64 {
	bits := binary.LittleEndian.Uint64(bytes)
	float := math.Float64frombits(bits)
	return float
}

func float64Bytes(float float64) []byte {
	bits := math.Float64bits(float)
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, bits)
	return bytes
}

func slowHandler(ctx context.Context, data []byte) ([]byte, error) {
	time.Sleep(time.Second * 3)
	return nil, nil
}

func sqrtHandler(ctx context.Context, data []byte) ([]byte, error) {
	f := float64FromBytes(data)
	if f < 0 {
		return nil, fmt.Errorf("can't extract square root from negative")
	}

	result := math.Sqrt(f)
	return float64Bytes(result), nil
}

func getNumberedHandler(ch chan<- int, x int, t *testing.T) p2pclient.UnaryHandlerFunc {
	return func(ctx context.Context, data []byte) ([]byte, error) {
		t.Logf("numbered handler x = %d", x)
		ch <- x
		return []byte("test"), nil
	}
}

// Je reprends mon bien où je le trouve
// https://stackoverflow.com/questions/47969385/go-float-comparison
func almostEqual(a, b float64) bool {
	return math.Abs(a-b) <= 1e-9
}
