# libp2p Daemon Peerstore Protocol

The libp2p daemon Peerstore protocol allows clients to interact with the libp2p daemon's Peerstore.

_At the moment, this is a living document. As such, it will be susceptible to
changes until stabilization._

## Protocol Specification

### Data Types

The data structures are defined in [pb/p2pd.proto](../pb/p2pd.proto). All messages
are varint-delimited. For the DHT queries, the relevant data types are:

- `PeerstoreRequest`
- `PeerstoreResponse`

All Peerstore requests will be wrapped in a `Request` message with `Type: PEERSTORE`.
Peerstore responses from the daemon will be wrapped in a `Response` with the
`PeerstoreResponse` field populated. Some responses will be basic `Response` messages to convey whether or not there was an error.

`PeerstoreRequest` messages have a `Type` parameter that specifies the specific operation
the client wishes to execute.

### Protocol Requests

*Protocols described in pseudo-go. Items of the form [item, ...] are lists of
many items.*

#### Errors

Any response that may be an error, will take the form of:

```
Response{
  Type: ERROR,
  ErrorResponse: {
    Msg: <error message>,
  },
}
```

#### `ADD_PROTOCOLS`
Clients can issue a `ADD_PROTOCOLS` request to add protocols to the known list for a given peer.

**Client**
```
Request{
  Type: PEERSTORE,
  PeerstoreRequest: PeerstoreRequest{
    Type: ADD_PROTOCOLS,
    Id: <peer id>,
    Protos: [<protocol string>, ...],
  },
}
```

**Daemon**
*Can return an error*

```
Response{
  Type: OK
}
```

#### `GET_PROTOCOLS`
Clients can issue a `GET_PROTOCOLS` request to get the known list of protocols for a given peer.

**Client**
```
Request{
  Type: PEERSTORE,
  PeerstoreRequest: PeerstoreRequest{
    Type: GET_PROTOCOLS,
    Id: <peer id>,
  },
}
```

**Daemon**
*Can return an error*

```
Response{
  Type: OK,
  PeerstoreResponse: PeerstoreResponse{
    Protos: [<protocol string>, ...],
  },
}
```

#### `GET_PEER_INFO`
Clients can issue a `GET_PEER_INFO` request to get the PeerInfo for a given peer id.

**Client**
```
Request{
  Type: PEERSTORE,
  PeerstoreRequest: PeerstoreRequest{
    Type: GET_PEER_INFO,
    Id: <peer id>,
  },
}
```

**Daemon**
*Can return an error*

```
Response{
  Type: OK,
  PeerstoreResponse: PeerstoreResponse{
    Peer: PeerInfo{
      Id: <peer id>,
      Addrs: [<addr>, ...],
    },
  },
}
```
