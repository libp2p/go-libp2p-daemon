SHELL := /bin/sh

include config.mk

.PHONY : all java-daemon java-client go-client go-daemon deps gx clean
.DEFAULT_GOAL : go-daemon

all: deps go-daemon go-client go-bindings java-daemon java-client

java-daemon:
	cd $(BDIR) && make $@

java-client:
	cd $(BDIR) && make $@

go-bindings:
	cd $(BDIR) && make $@

go-client:
	cd $(CDIR) && go install ./...

go-daemon:
	cd $(DDIR) && go install ./...

deps: gx
	gx --verbose install --global
	gx-go rewrite

gx:
	go get github.com/whyrusleeping/gx
	go get github.com/whyrusleeping/gx-go

clean:
	gx-go uw
	cd $(BDIR) && make $@

