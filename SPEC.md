# libp2p Daemon

The libp2p daemon is a standalone binary meant to make it easy to bring
peer-to-peer networking to new languages without fully porting libp2p and all
of its complexities.

_At the moment, this is a living document. As such, it will be susceptible to
changes until stabilization._

## Structure

### Overview

There are two pieces to the libp2p daemon:

- __Daemon__: A golang daemon that manages libp2p hosts and proxies streams to
  the end user.
- __Client__: A library written in any language that controls the daemon over
  a protocol specified in this document, allowing end users to enjoy the
  benefits of peer-to-peer networking without implementing a full libp2p stack.

### Technical Details

The libp2p daemon and client will communicate with each other over an HTTP API
for the time being. In the future, this may likely evolve into a simpler,
lighter weight protocol. Both the daemon and the client will run HTTP servers,
facilitating bidirectional communication.

In the initial implementation, communication between end users and the daemon
will take place over unix sockets. Each unix socket will correspond to a libp2p
stream, operating on a given protocol.

Future implementations may attempt to take advantage of shared memory (shmem)
or other IPC constructs.

## HTTP API Specification

### Data Types

_Keys surrounded in <> brackets are optional. Values surrounded in [] are
arrays._

- `Error`
  ```
  {
    error: <message>,
  }
  ```

- `Direction`
  ```
  INCOMING or OUTGOING
  ```

- `Stream`
  ```
  {
    "localAddr": <multiaddr>,
    "localID": <peer ID>,
    "remoteAddr": <multiaddr>,
    "remoteID": <peer ID>,
    "createdAt": <time>,
    "direction": <Direction>,
    "socketPath": <path>,
    "protocol": <string>,
  }
  ```

- `Peer`
  ```
  {
    "peerID": <b58 formatted peer id>,
    "addresses": [<multiaddr>],
    "<connected>": <bool>,
    "<streams>": [<Stream>],
  }
  ```

- `Address`
  ```
  {
    "address": <multiaddr>,
  }
  ```

### API Routes

The API has two namespaces, daemon and client.

#### Daemon

- `POST /api/daemon/peers` __Adds a peer to the Host's Peerstore.__
  - Request `Peer`
  - Response
    - 204 _Successfully added addresses for peer_
    - 500 _Failed adding addresses for peer_
      `Error`
- `GET /api/daemon/peers` _Returns all known peers in the Host's Peerstore_
  - Response
    `[Peer]`
- `GET /api/daemon/peers/<id>`
  - Response
    - 404 _Peer not in store_
    - 200 _Peer information_
      `Peer`
- `POST /api/daemon/peers/<id>/streams` _Creates a new stream for a peer_
  - Request _Includes either an empty body or an `Address` to connect to._
  - Response `Stream`

#### Client

- `POST /api/client/peers/<id>/streams` _Notify the client of an incoming stream
  request._
  - Request `Stream`
  - Response 204
