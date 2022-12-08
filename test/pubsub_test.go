package test

import (
	"context"
	"testing"
	"time"
)

func TestPubsubGetTopicsAndSubscribe(t *testing.T) {
	_, client, closer := createDaemonClientPair(t)
	defer closer()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	_, err := client.Subscribe(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}

	topics, err := client.GetTopics()
	if err != nil {
		t.Fatal(err)
	}
	if len(topics) != 1 {
		t.Fatalf("expected 1 topic, found %d", len(topics))
	}
	if topics[0] != "test" {
		t.Fatalf("expected topic \"test\", found \"%s\"", topics[0])
	}
	cancel()
}

func TestPubsubMessages(t *testing.T) {
	_, sender, senderCloser := createDaemonClientPair(t)
	defer senderCloser()
	_, receiver, receiverCloser := createDaemonClientPair(t)
	defer receiverCloser()

	id, addrs, err := receiver.Identify()
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	msgs, err := receiver.Subscribe(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)

	if err = sender.Connect(id, addrs); err != nil {
		t.Fatal(err)
	}

	time.Sleep(2 * time.Second) // wait a second for the subscription to take effect

	go func() {
		if err := sender.Publish("test", []byte("foobar")); err != nil {
			t.Error(err)
		}
	}()

	select {
	case msg, ok := <-msgs:
		if !ok {
			t.Fatal("expected a message but was unsubscribed first")
		}
		msgstr := string(msg.Data)
		if msgstr != "foobar" {
			t.Fatalf("expected \"foobar\", got %s", msgstr)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for message")
	}
}
