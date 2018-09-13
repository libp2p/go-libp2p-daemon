package p2pd

import (
	"io"
	"net"
	"sync"

	pb "github.com/libp2p/go-libp2p-daemon/pb"

	ggio "github.com/gogo/protobuf/io"
	inet "github.com/libp2p/go-libp2p-net"
)

func (d *Daemon) doStreamPipe(c net.Conn, s inet.Stream) {
	var wg sync.WaitGroup
	wg.Add(2)

	pipe := func(dst io.Writer, src io.Reader) {
		_, err := io.Copy(dst, src)
		if err != nil && err != io.EOF {
			log.Debugf("stream error: %s", err.Error())
		}
		wg.Done()
	}

	go pipe(c, s)
	go pipe(s, c)

	wg.Wait()
	s.Close()
}

func (d *Daemon) handleStream(s inet.Stream) {
	defer s.Close()
	p := s.Protocol()

	d.mx.Lock()
	path, ok := d.handlers[p]
	d.mx.Unlock()

	if !ok {
		log.Debugf("unexpected stream: %s", p)
		return
	}

	c, err := net.Dial("unix", path)
	if err != nil {
		log.Debugf("error dialing handler at %s: %s", path, err.Error())
		return
	}
	defer c.Close()

	w := ggio.NewDelimitedWriter(c)
	msg := pb.StreamAccept{
		Peer: []byte(s.Conn().RemotePeer()),
		Addr: s.Conn().RemoteMultiaddr().Bytes(),
	}
	err = w.WriteMsg(&msg)
	if err != nil {
		log.Debugf("error accepting stream: %s", err.Error())
		return
	}

	d.doStreamPipe(c, s)

}
