package p2pd

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/learning-at-home/go-libp2p-daemon/internal/utils"
	pb "github.com/learning-at-home/go-libp2p-daemon/pb"

	ggio "github.com/gogo/protobuf/io"
	ma "github.com/multiformats/go-multiaddr"
)

const DefaultTimeout = 60 * time.Second

func (d *Daemon) handleConn(c net.Conn) {
	defer c.Close()

	r := ggio.NewDelimitedReader(c, d.persistentConnMsgMaxSize)
	w := ggio.NewDelimitedWriter(c)

	for {
		var req pb.Request

		err := r.ReadMsg(&req)
		if err != nil {
			if err != io.EOF {
				log.Debugw("error reading message", "error", err)
			}
			return
		}

		log.Debugw("request", "type", req.GetType())

		switch req.GetType() {
		case pb.Request_IDENTIFY:
			res := d.doIdentify(&req)
			err := w.WriteMsg(res)
			if err != nil {
				log.Debugw("error writing response", "error", err)
				return
			}

		case pb.Request_CONNECT:
			res := d.doConnect(&req)
			err := w.WriteMsg(res)
			if err != nil {
				log.Debugw("error writing response", "error", err)
				return
			}

		case pb.Request_STREAM_OPEN:
			res, s := d.doStreamOpen(&req)
			err := w.WriteMsg(res)
			if err != nil {
				log.Debugw("error writing response", "error", err)
				if s != nil {
					s.Reset()
				}
				return
			}

			if s != nil {
				d.doStreamPipe(c, s)
				return
			}

		case pb.Request_STREAM_HANDLER:
			res := d.doStreamHandler(&req)
			err := w.WriteMsg(res)
			if err != nil {
				log.Debugw("error writing response", "error", err)
				return
			}

		case pb.Request_REMOVE_STREAM_HANDLER:
			res := d.doRemoveStreamHandler(&req)
			err := w.WriteMsg(res)
			if err != nil {
				log.Debugw("error writing response", "error", err)
				return
			}

		case pb.Request_DHT:
			res, ch, cancel := d.doDHT(&req)
			err := w.WriteMsg(res)
			if err != nil {
				log.Debugw("error writing response", "error", err)
				if ch != nil {
					cancel()
				}
				return
			}

			if ch != nil {
				for res := range ch {
					err = w.WriteMsg(res)
					if err != nil {
						log.Debugw("error writing response", "error", err)
						cancel()
						return
					}
				}

				err = w.WriteMsg(dhtResponseEnd())
				if err != nil {
					log.Debugw("error writing response", "error", err)
					return
				}
			}

		case pb.Request_LIST_PEERS:
			res := d.doListPeers(&req)
			err := w.WriteMsg(res)
			if err != nil {
				log.Debugw("error writing response", "error", err)
				return
			}

		case pb.Request_CONNMANAGER:
			res := d.doConnManager(&req)
			err := w.WriteMsg(res)
			if err != nil {
				log.Debugw("error writing response", "error", err)
				return
			}

		case pb.Request_DISCONNECT:
			res := d.doDisconnect(&req)
			err := w.WriteMsg(res)
			if err != nil {
				log.Debugw("error writing response", "error", err)
				return
			}

		case pb.Request_PUBSUB:
			res, sub := d.doPubsub(&req)
			err := w.WriteMsg(res)
			if err != nil {
				log.Debugw("error writing response", "error", err)
				if sub != nil {
					sub.Cancel()
				}
				return
			}

			if sub != nil {
				d.doPubsubPipe(sub, r, w)
				return
			}

		case pb.Request_PERSISTENT_CONN_UPGRADE:
			d.handlePersistentConn(r, w)
			return

		default:
			log.Debugw("unexpected request type", "type", req.GetType())
			return
		}
	}
}

func (d *Daemon) doIdentify(req *pb.Request) *pb.Response {
	id := []byte(d.ID())
	addrs := d.Addrs()
	baddrs := make([][]byte, len(addrs))
	for x, addr := range addrs {
		baddrs[x] = addr.Bytes()
	}

	res := okResponse()
	res.Identify = &pb.IdentifyResponse{Id: id, Addrs: baddrs}
	return res
}

func (d *Daemon) doConnect(req *pb.Request) *pb.Response {
	if req.Connect == nil {
		return errorResponseString("Malformed request; missing parameters")
	}

	ctx, cancel := d.requestContext(req.Connect.GetTimeout())
	defer cancel()

	pid, err := peer.IDFromBytes(req.Connect.Peer)
	if err != nil {
		log.Debugw("error parsing peer ID", "error", err)
		return errorResponse(err)
	}

	addrs := make([]ma.Multiaddr, len(req.Connect.Addrs))
	for x, bs := range req.Connect.Addrs {
		addr, err := ma.NewMultiaddrBytes(bs)
		if err != nil {
			log.Debugw("Error parsing multiaddr", "error", err)
			return errorResponse(err)
		}
		addrs[x] = addr
	}

	pi := peer.AddrInfo{ID: pid, Addrs: addrs}

	log.Debug("connecting", "to", pid)
	err = d.host.Connect(ctx, pi)
	if err != nil {
		log.Debugw("error opening connection", "to", pid, "error", err)
		return errorResponse(err)
	}

	return okResponse()
}

func (d *Daemon) doDisconnect(req *pb.Request) *pb.Response {
	if req.Disconnect == nil {
		return errorResponseString("Malformed request; missing parameters")
	}

	p, err := peer.IDFromBytes(req.Disconnect.GetPeer())
	if err != nil {
		return errorResponse(err)
	}

	err = d.host.Network().ClosePeer(p)
	if err != nil {
		return errorResponse(err)
	}

	return okResponse()
}

func (d *Daemon) doStreamOpen(req *pb.Request) (*pb.Response, network.Stream) {
	if req.StreamOpen == nil {
		return errorResponseString("Malformed request; missing parameters"), nil
	}

	ctx, cancel := d.requestContext(req.StreamOpen.GetTimeout())
	defer cancel()

	pid, err := peer.IDFromBytes(req.StreamOpen.Peer)
	if err != nil {
		log.Debugw("Error parsing peer ID", "error", err)
		return errorResponse(err), nil
	}

	protos := make([]protocol.ID, len(req.StreamOpen.Proto))
	for x, str := range req.StreamOpen.Proto {
		protos[x] = protocol.ID(str)
	}

	log.Debugf("opening stream", "to", pid)
	ctx = network.WithUseTransient(ctx, "allow testing over relayed connections")
	s, err := d.host.NewStream(ctx, pid, protos...)
	if err != nil {
		log.Debugw("error opening stream", "to", pid, "error", err)
		return errorResponse(err), nil
	}

	res := okResponse()
	res.StreamInfo = makeStreamInfo(s)
	return res, s
}

func (d *Daemon) doStreamHandler(req *pb.Request) *pb.Response {
	if req.StreamHandler == nil {
		return errorResponseString("Malformed request; missing parameters")
	}

	d.mx.Lock()
	defer d.mx.Unlock()

	maddr, err := ma.NewMultiaddrBytes(req.StreamHandler.Addr)
	if err != nil {
		return errorResponse(err)
	}
	for _, sp := range req.StreamHandler.Proto {
		p := protocol.ID(sp)
		round_robin, ok := d.handlers[p]
		if !ok {
			d.handlers[p] = utils.NewRoundRobin(true)
			d.handlers[p].Append(maddr)
			d.host.SetStreamHandler(p, d.handleStream)
		} else if !req.StreamHandler.GetBalanced() {
			return errorResponseString(fmt.Sprintf("handler for protocol %s already set", p))
		} else {
			round_robin.Append(maddr)
		}
		log.Debugw("set stream handler", "protocol", sp, "to", maddr)
	}

	return okResponse()
}

func (d *Daemon) doRemoveStreamHandler(req *pb.Request) *pb.Response {
	if req.RemoveStreamHandler == nil {
		return errorResponseString("Malformed request; missing parameters")
	}

	d.mx.Lock()
	defer d.mx.Unlock()

	maddr, err := ma.NewMultiaddrBytes(req.RemoveStreamHandler.Addr)
	if err != nil {
		return errorResponse(err)
	}
	for _, sp := range req.RemoveStreamHandler.Proto {
		p := protocol.ID(sp)
		round_robin, ok := d.handlers[p]
		if !ok {
			return errorResponseString(fmt.Sprintf("handler for protocol %s does not exist", p))
		}

		ok = round_robin.Remove(maddr)
		if !ok {
			return errorResponseString(fmt.Sprintf("handler for protocol %s with maddr %s does not exist", p, maddr.String()))
		}
		if round_robin.Len() == 0 {
			d.host.RemoveStreamHandler(p)
			delete(d.handlers, p)
		}
		log.Debugw("removed stream handler", "protocol", sp, "to", maddr)
	}

	return okResponse()
}

func (d *Daemon) doListPeers(req *pb.Request) *pb.Response {
	conns := d.host.Network().Conns()
	peers := make([]*pb.PeerInfo, len(conns))
	for x, conn := range conns {
		peers[x] = &pb.PeerInfo{
			Id:    []byte(conn.RemotePeer()),
			Addrs: [][]byte{conn.RemoteMultiaddr().Bytes()},
		}
	}

	res := okResponse()
	res.Peers = peers
	return res
}

func (d *Daemon) requestContext(utime int64) (context.Context, func()) {
	timeout := DefaultTimeout
	if utime > 0 {
		timeout = time.Duration(utime) * time.Second
	}

	return context.WithTimeout(d.ctx, timeout)
}

func okResponse() *pb.Response {
	return &pb.Response{
		Type: pb.Response_OK.Enum(),
	}
}

func errorResponse(err error) *pb.Response {
	errstr := err.Error()
	return &pb.Response{
		Type:  pb.Response_ERROR.Enum(),
		Error: &pb.ErrorResponse{Msg: &errstr},
	}
}

func errorResponseString(err string) *pb.Response {
	return &pb.Response{
		Type:  pb.Response_ERROR.Enum(),
		Error: &pb.ErrorResponse{Msg: &err},
	}
}

func makeStreamInfo(s network.Stream) *pb.StreamInfo {
	proto := string(s.Protocol())
	return &pb.StreamInfo{
		Peer:  []byte(s.Conn().RemotePeer()),
		Addr:  s.Conn().RemoteMultiaddr().Bytes(),
		Proto: &proto,
	}
}
