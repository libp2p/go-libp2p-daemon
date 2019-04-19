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
Please see the json [schema file](config.schema.json).

### Complete (Default Options) Example

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
    "Mode": ""
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
    "SignStrict": false,
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
  "MetricsAddress": "",
  "PProf": {
    "Enabled": false
  }
}
```
