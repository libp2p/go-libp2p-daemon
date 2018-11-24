SHELL := /bin/sh

include config.mk

.PHONY : all java-daemon go-daemon daemon-shared-object deps gx clean
.DEFAULT_GOAL : go-daemon

all: deps go-daemon java-daemon daemon-shared-object

java-daemon:
	cd $(BDIR) && make $@

daemon-shared-object:
	cd $(BDIR) && make $@

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

