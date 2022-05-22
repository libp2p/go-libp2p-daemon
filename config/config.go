package config

import (
	"encoding/json"
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
	*jm = JSONMaddr{ma}
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

type Bootstrap struct {
	Enabled bool
	Peers   MaddrArray
}

type ConnectionManager struct {
	Enabled       bool
	LowWaterMark  int
	HighWaterMark int
	GracePeriod   time.Duration
}

type GossipSubHeartbeat struct {
	Interval     time.Duration
	InitialDelay time.Duration
}

type PubSub struct {
	Enabled            bool
	Router             string
	Sign               bool
	SignStrict         bool
	GossipSubHeartbeat GossipSubHeartbeat
}

type Relay struct {
	Enabled  bool
	Auto     bool
	HopLimit int
}

type DHT struct {
	Mode string
}

type PProf struct {
	Enabled bool
	Port    uint
}

type Security struct {
	Noise bool
	TLS   bool
}

const DHTFullMode = "full"
const DHTClientMode = "client"
const DHTServerMode = "server"

type Config struct {
	ListenAddr        JSONMaddr
	Quiet             bool
	ID                string
	Bootstrap         Bootstrap
	DHT               DHT
	ConnectionManager ConnectionManager
	NatPortMap        bool
	PubSub            PubSub
	Relay             Relay
	AutoNat           bool
	HostAddresses     MaddrArray
	AnnounceAddresses MaddrArray
	NoListen          bool
	MetricsAddress    string
	PProf             PProf
	Security          Security
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
	if c.DHT.Mode != DHTClientMode && c.DHT.Mode != DHTFullMode && c.DHT.Mode != DHTServerMode && c.DHT.Mode != "" {
		return fmt.Errorf("unknown DHT mode %s", c.DHT)
	}
	if c.Relay.Auto && (!c.Relay.Enabled || c.DHT.Mode == "") {
		return fmt.Errorf("can't have autorelay enabled without Relay enabled and DHT enabled")
	}
	return nil
}

func NewDefaultConfig() Config {
	defaultListen, _ := multiaddr.NewMultiaddr("/unix/tmp/p2pd.sock")
	return Config{
		ListenAddr: JSONMaddr{defaultListen},
		Quiet:      false,
		ID:         "",
		Bootstrap: Bootstrap{
			Enabled: false,
			Peers:   make(MaddrArray, 0),
		},
		DHT: DHT{
			Mode: "",
		},
		ConnectionManager: ConnectionManager{
			Enabled:       false,
			LowWaterMark:  256,
			HighWaterMark: 512,
			GracePeriod:   120 * time.Second,
		},
		NatPortMap: false,
		PubSub: PubSub{
			Enabled:    false,
			Router:     "gossipsub",
			Sign:       true,
			SignStrict: true,
			GossipSubHeartbeat: GossipSubHeartbeat{
				Interval:     0,
				InitialDelay: 0,
			},
		},
		Relay: Relay{
			Enabled:  true,
			Auto:     false,
			HopLimit: 0,
		},
		AutoNat:           false,
		HostAddresses:     make(MaddrArray, 0),
		AnnounceAddresses: make(MaddrArray, 0),
		NoListen:          false,
		MetricsAddress:    "",
		PProf: PProf{
			Enabled: false,
			Port:    0,
		},
		Security: Security{
			Noise: true,
			TLS:   true,
		},
	}
}
