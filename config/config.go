package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/multiformats/go-multiaddr"
)

type JSONMaddr struct {
	multiaddr.Multiaddr
}

func (jm *JSONMaddr) UnmarshalJSON(b []byte) error {
	ma, err := multiaddr.NewMultiaddr(string(b))
	if err != nil {
		return err
	}
	jma := ma.(JSONMaddr)
	jm = &jma
	return nil
}

type MaddrArray []multiaddr.Multiaddr

func (maa *MaddrArray) UnmarshalJSON(b []byte) error {
	maStrings := strings.Split(string(b), ",")
	*maa = make(MaddrArray, len(maStrings))
	for i, s := range strings.Split(string(b), ",") {
		ma, err := multiaddr.NewMultiaddr(s)
		if err != nil {
			return err
		}
		(*maa)[i] = ma
	}
	return nil
}

type bootstrap struct {
	Enabled bool
	Peers   MaddrArray
}

type connectionManager struct {
	Enabled       bool
	LowWaterMark  int
	HighWaterMark int
	GracePeriod   time.Duration
}

type gossipSubHeartbeat struct {
	Interval     time.Duration
	InitialDelay time.Duration
}

type pubSub struct {
	Enabled            bool
	Router             string
	Sign               bool
	SignStrict         bool
	GossipSubHeartbeat gossipSubHeartbeat
}

type relay struct {
	Enabled   bool
	Active    bool
	Hop       bool
	Discovery bool
	Auto      bool
}

const DHTFullMode = "full"
const DHTClientMode = "client"

type Config struct {
	ListenAddr        JSONMaddr
	Quiet             bool
	ID                string
	Bootstrap         bootstrap
	DHT               string
	ConnectionManager connectionManager
	QUIC              bool
	NatPortMap        bool
	PubSub            pubSub
	Relay             relay
	AutoNat           bool
	HostAddresses     MaddrArray
	AnnounceAddresses MaddrArray
	NoListen          bool
	MetricsAddress    string
}

func (c *Config) UnmarshalJSON(b []byte) error {
	// settings defaults
	type defaultConfig Config
	ndc := defaultConfig(NewDefaultConfig())
	dc := &ndc
	if err := json.Unmarshal(b, dc); err != nil {
		return err
	}
	*c = Config(*dc)

	// validation
	if err := c.Validate(); err != nil {
		return err
	}

	return nil
}

func (c *Config) Validate() error {
	if c.DHT != DHTClientMode && c.DHT != DHTFullMode && c.DHT != "" {
		return errors.New(fmt.Sprintf("unknown DHT mode %s", c.DHT))
	}
	if c.Relay.Auto == true && (c.Relay.Enabled == false || c.DHT == "") {
		return errors.New("can't have autorelay enabled without relay enabled and dht enabled")
	}
	return nil
}

func NewDefaultConfig() Config {
	defaultListen, _ := multiaddr.NewMultiaddr("/unix/tmp/p2pd.sock")
	return Config{
		ListenAddr: JSONMaddr{defaultListen},
		Quiet:      false,
		ID:         "",
		Bootstrap: bootstrap{
			Enabled: false,
			Peers:   make(MaddrArray, 0),
		},
		DHT: "",
		ConnectionManager: connectionManager{
			Enabled:       false,
			LowWaterMark:  256,
			HighWaterMark: 512,
			GracePeriod:   120,
		},
		QUIC:       false,
		NatPortMap: false,
		PubSub: pubSub{
			Enabled:    false,
			Router:     "gossipsub",
			Sign:       true,
			SignStrict: false,
			GossipSubHeartbeat: gossipSubHeartbeat{
				Interval:     0,
				InitialDelay: 0,
			},
		},
		Relay: relay{
			Enabled:   true,
			Hop:       false,
			Discovery: false,
			Auto:      false,
		},
		AutoNat:           false,
		HostAddresses:     make(MaddrArray, 0),
		AnnounceAddresses: make(MaddrArray, 0),
		NoListen:          false,
		MetricsAddress:    "",
	}
}
