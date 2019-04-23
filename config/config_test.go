package config

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/multiformats/go-multiaddr"
)

func TestDefaultConfig(t *testing.T) {
	const inputJson = "{}"
	var c Config
	if err := json.Unmarshal([]byte(inputJson), &c); err != nil {
		t.Fatal(err)
	}

	defaultListen, err := multiaddr.NewMultiaddr("/unix/tmp/p2pd.sock")
	if err != nil {
		t.Fatal(err)
	}
	if c.ListenAddr.String() != defaultListen.String() {
		t.Fatal(fmt.Sprintf("Expected %s, got %s", defaultListen.String(), c.ListenAddr.String()))
	}
}
