package p2pd

import (
	"io"
	"net"
	"sync"

	ggio "github.com/gogo/protobuf/io"
	inet "github.com/libp2p/go-libp2p-net"
	"github.com/multiformats/go-multiaddr-net"
)

func (d *Daemon) doStreamPipe(c net.Conn, s inet.Stream) {
	var wg sync.WaitGroup
	wg.Add(2)

	pipe := func(dst io.WriteCloser, src io.Reader) {
		_, err := io.Copy(dst, src)
		if err != nil && err != io.EOF {
			log.Debugf("stream error: %s", err.Error())
			s.Reset()
		}
		dst.Close()
		wg.Done()
	}

	go pipe(c, s)
	go pipe(s, c)

	wg.Wait()
}

func (d *Daemon) handleStream(s inet.Stream) {
	p := s.Protocol()

	d.mx.Lock()
	maddr, ok := d.handlers[p]
	d.mx.Unlock()

	if !ok {
		log.Debugf("unexpected stream: %s", p)
		s.Reset()
		return
	}

	c, err := manet.Dial(maddr)
	if err != nil {
		log.Debugf("error dialing handler at %s: %s", maddr, err.Error())
		s.Reset()
		return
	}
	defer c.Close()

	w := ggio.NewDelimitedWriter(c)
	msg := makeStreamInfo(s)
	err = w.WriteMsg(msg)
	if err != nil {
		log.Debugf("error accepting stream: %s", err.Error())
		s.Reset()
		return
	}

	d.doStreamPipe(c, s)
}
