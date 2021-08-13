package utils

import (
	"sync"

	ggio "github.com/gogo/protobuf/io"
	"github.com/gogo/protobuf/proto"
)

func NewSafeWriter(w ggio.WriteCloser) *safeWriter {
	return &safeWriter{w: w}
}

type safeWriter struct {
	w ggio.WriteCloser
	m sync.Mutex
}

func (sw *safeWriter) WriteMsg(msg proto.Message) error {
	sw.m.Lock()
	defer sw.m.Unlock()
	return sw.w.WriteMsg(msg)
}

func (sw *safeWriter) Close() error {
	sw.m.Lock()
	defer sw.m.Unlock()
	return sw.w.Close()
}
