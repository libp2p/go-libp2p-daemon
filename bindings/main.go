package main

import "C"
import (
	p2pd "github.com/libp2p/go-libp2p-daemon"
	p2pc "github.com/libp2p/go-libp2p-daemon/p2pclient"
)

func main() {
}

//export startClient
func startClient(args *C.char) {
	argsGoString := C.GoString(args)
	config := p2pc.ProcessArgs(&argsGoString)
	p2pc.Start(config)
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
