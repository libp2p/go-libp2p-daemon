package p2pd

import (
	"io"
	"net"
	"sync"

	"github.com/libp2p/go-libp2p-core/network"

	ggio "github.com/gogo/protobuf/io"
	manet "github.com/multiformats/go-multiaddr/net"
)

func (d *Daemon) doStreamPipe(c net.Conn, s network.Stream) {
	var wg sync.WaitGroup
	wg.Add(2)

	pipe := func(dst io.WriteCloser, src io.Reader) {
		_, err := io.Copy(dst, src)
		if err != nil && err != io.EOF {
			log.Debugw("stream error", "error", err)
			s.Reset()
		}
		dst.Close()
		wg.Done()
	}

	go pipe(c, s)
	go pipe(s, c)

	wg.Wait()
}

func (d *Daemon) handleStream(s network.Stream) {
	p := s.Protocol()

	d.mx.Lock()
	maddr, ok := d.handlers[p]
	d.mx.Unlock()

	if !ok {
		log.Debugw("unexpected stream", "protocol", p)
		s.Reset()
		return
	}

	c, err := manet.Dial(maddr)
	if err != nil {
		log.Debugw("error dialing handler", "handler", maddr.String(), "error", err)
		s.Reset()
		return
	}
	defer c.Close()

	w := ggio.NewDelimitedWriter(c)
	msg := makeStreamInfo(s)
	err = w.WriteMsg(msg)
	if err != nil {
		log.Debugw("error accepting stream", "error", err)
		s.Reset()
		return
	}

	d.doStreamPipe(c, s)
}
