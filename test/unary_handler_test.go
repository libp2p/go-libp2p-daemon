package test

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-daemon/p2pclient"
)

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
	if err := p1.AddUnaryHandler(proto, sqrtHandler); err != nil {
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
	if err := p1.AddUnaryHandler(proto, slowHandler); err != nil {
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

	if err := c1.AddUnaryHandler(proto, sqrtHandler); err != nil {
		t.Fatal(err)
	}

	if err := c2.AddUnaryHandler(proto, sqrtHandler); err == nil {
		t.Fatal("adding second unary handler with same name should have returned error")
	}

	if err := c1.Close(); err != nil {
		t.Fatal("closing client 1 should not have returned an error", err)
	}

	time.Sleep(time.Second)

	if err := c2.AddUnaryHandler(proto, sqrtHandler); err != nil {
		t.Fatal("closing client 1 should have cleaned up the proto list", err)
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

// Je reprends mon bien où je le trouve
// https://stackoverflow.com/questions/47969385/go-float-comparison
func almostEqual(a, b float64) bool {
	return math.Abs(a-b) <= 1e-9
}
