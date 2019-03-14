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
	done := make(chan struct{})
	go func() {
		_, err := client.Subscribe(ctx, "test")
		if err != nil {
			t.Fatal(err)
		}
		done <- struct{}{}
	}()
	<-done
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

	if err = sender.Connect(id, addrs); err != nil {
		t.Fatal(err)
	}

	progress := make(chan struct{})
	done := make(chan struct{})
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		msgs, err := receiver.Subscribe(ctx, "test")
		if err != nil {
			t.Fatal(err)
		}
		progress <- struct{}{}

		select {
		case msg := <-msgs:
			msgstr := string(msg.Data)
			if msgstr != "foobar" {
				t.Fatalf("expected \"foobar\", got %s", msgstr)
			}
			done <- struct{}{}
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for message")
		}
	}()

	go func() {
		<-progress
		if err := sender.Publish("test", []byte("foobar")); err != nil {
			t.Fatal(err)
		}
	}()

	<-done
}
