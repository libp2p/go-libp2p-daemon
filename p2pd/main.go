package main

import "C"
import (
	identify "github.com/libp2p/go-libp2p/p2p/protocol/identify"

	p2pd "github.com/libp2p/go-libp2p-daemon"
)

func main() {
	identify.ClientVersion = "p2pd/0.1"
	config := p2pd.Initialize()
	p2pd.Start(config)
}

//export startDaemon
func startDaemon(args *C.char) {
	argsGoString := C.GoString(args)
	config := p2pd.ProcessArgs(&argsGoString)
	p2pd.Start(config)
}

//export stopDaemon
func stopDaemon() {
	p2pd.Stop()
}
