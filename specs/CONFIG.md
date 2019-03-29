# libp2p Daemon Config Spec

The libp2p daemon is an interface to libp2p, and as such has a large number of 
configuration options. For complex deployments, these options can become 
unwieldy on the command line. For this, we allow the ability to provide a
configuration spec via json from a file or from stdin

_At the moment, this is a living document. As such, it will be susceptible to
changes until stabilization._

## Considerations

As this is a spec that must be shared between multiple implementations, it is 
probably best that the structure be generated from some shared source of truth,
instead of depending on discipline to keep the implementation consistent 
between language implementations. For instance, the config structure could be 
generated as a protobuf message.

## Command Line

There are two ways to provide a JSON configuration to the daemon. Both methods 
cause all other configuration command line options to be ignored.

### Read from a file
`p2pd -f ./conf.json`

### Read from stdin
`cat ./conf.json | p2pd -i`


## Schema

* `Field Name`
    * Description
    * Type
    * `default`

* `Listen`
    * Daemon control listen multiaddr
    * Maddr String
    * `"/unix/tmp/p2pd.sock"`
* `Quiet`
    * Be Quiet
    * Boolean
    * `false`
* `ID`
    * Peer identity; private key file
    * String
    * `""`
* `Bootstrap`
    * `Enabled`
        * Connects to bootstrap peers and bootstraps the dht if enabled
        * Boolean
        * `false`
    * `Peers`
        * List of bootstrap peers; defaults to the IPFS DHT peers
        * Array[Maddr String]
        * `[]`
* `DHT`
    * `Enabled`
        * Enables the DHT in full node mode
        * Boolean
        * `false`
    * `ClientMode`
        * Enables the DHT in client mode
        * Boolean
        * `false`
* `ConnectionManager`
    * `Enabled`
        * Enables the Connection Manager
        * Boolean
        * `false`
    * `LowWaterMark`
        * Connection Manager Low Water mark
        * Integer
        * `256`
    * `HighWaterMark`
        * Connection Manager High Water mark
        * Integer
        * `512`
    * `GracePeriod`
        * Connection Manager grace period (in seconds)
        * Integer
        * `120`
* `QUIC` 
    * Enables the QUIC transport
    * Boolean
    * `false`
* `NatPortMap`
    * Enables NAT port mapping
    * Boolean
    * `false`
* `PubSub`
    * `Enabled`
        * Enables pubsub
        * Boolean
        * `false`
    * `Router`
        * Specifies the pubsub router implementation
        * String
        * `"gossipsub"`
    * `Sign`
        * Enables pubsub message signing
        * Boolean
        * `true`
    * `SignStrict`
        * Enables pubsub strict signature verification
        * Boolean
        * `false`
    * `GossipSubHeartbeat`
        * `Interval`
            * Specifies the gossipsub heartbeat interval
            * Integer
            * `0`
        * `InitialDelay`
            * Specifies the gossipsub initial heartbeat delay
            * Integer
            * `0`
* `Relay`
    * `Enabled`
        * Enables circuit relay
        * Boolean
        * `true`
    * `Active`
        * Enables active mode for relay
        * Boolean
        * `false`
    * `Hop`
        * Enables hop for relay
        * Boolean
        * `false`
    * `Discovery`
        * Enables passive discovery for relay
        * Boolean
        * `false`
    * `Auto`
        * Enables autorelay
        * Boolean
        * `false`
* `AutoNat`
    * Enables the AutoNAT service
    * Boolean
    * `false`
* `HostAddresses`
    * List of multiaddrs the host should listen on
    * Array[Maddr String]
    * `[]`
* `AnnounceAddresses`
    * List of multiaddrs the host should announce to the network
    * Array[Maddr String]
    * `[]`
* `NoListen`
    * Sets the host to listen on no addresses
    * Boolean
    * `false`
* `MetricsAddress`
    * An address to bind the metrics handler to
    * Maddr String
    * `"""`
    
### Default Example

```json
{
  "ListenAddr": "/unix/tmp/p2pd.sock",
  "Quiet": false,
  "ID": "",
  "Bootstrap": {
    "Enabled": false,
    "Peers": []
  },
  "DHT": {
    "Enabled": false,
    "ClientMode": false
  },
  "ConnectionManager": {
    "Enabled": false,
    "LowWaterMark": 256,
    "HighWaterMark": 512,
    "GracePeriod": 120
  },
  "QUIC": false,
  "NatPortMap": false,
  "PubSub": {
    "Enabled": false,
    "Router": "gossipsub",
    "Sign": true,
    "SignStrict": true,
    "GossipSubHeartbeat": {
      "Interval": 0,
      "InitialDelay": 0
    }
  },
  "Relay": {
    "Enabled": true,
    "Active": false,
    "Hop": false,
    "Discovery": false,
    "Auto": false
  },
  "AutoNat": false,
  "HostAddresses": [],
  "AnnounceAddresses": [],
  "NoListen": false,
  "MetricsAddress": ""
}
```
