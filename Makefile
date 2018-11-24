SHELL := /bin/bash
OS = $(shell uname -s | tr '[:upper:]' '[:lower:]')

ifeq ($(OS), linux)
	EXT = so
	OS_LFLAGS =
	JAVA_HOME = 
else ifeq ($(OS), darwin)
	EXT = dylib
	OS_LFLAGS = -mmacosx-version-min=$(shell defaults read loginwindow SystemVersionStampAsString) -framework CoreFoundation -framework Security
	JAVA_HOME = $(shell /usr/libexec/java_home)
endif

CC = gcc
CFLAGS = -O2 -fPIC
LFLAGS = $(OS_LFLAGS) -shared

JAVA_INCLUDES = -I$(JAVA_HOME)/include/$(OS) -I$(JAVA_HOME)/include
CLASS_PATH = .
vpath %.class $(CLASS_PATH)

DDIR := p2pd
CDIR := p2pc
JDIR := p2pclient/java
DNAME := p2pd


.DEFAULT_GOAL := go-daemon

java-daemon: lib$(DNAME).$(EXT)

lib$(DNAME).$(EXT): java-$(DNAME).o go-$(DNAME).a
	$(CC) $(LFLAGS) -o $(JDIR)/$@ $(JDIR)/*.o $(JDIR)/*.a

java-$(DNAME).o: java-$(DNAME).h $(DNAME).class go-$(DNAME).a
	$(CC) $(CFLAGS) -c $(JDIR)/java-$(DNAME).c $(JAVA_INCLUDES) -o $(JDIR)/$@

go-$(DNAME).a: 
	go build -o $(JDIR)/$@ -buildmode=c-archive $(DDIR)/main.go

java-$(DNAME).h:
	cd $(JDIR) && javac -h $(CLASS_PATH) $(DNAME).java && mv $(DNAME).h $@

$(DNAME).class: go-daemon
	cd $(JDIR) && javac $(DNAME).java

go-client: deps
	cd $(CDIR) && go install ./...

go-daemon: deps
	cd $(DDIR) && go install ./...

deps: gx
	gx --verbose install --global
	gx-go rewrite

gx:
	go get github.com/whyrusleeping/gx
	go get github.com/whyrusleeping/gx-go

clean:
	gx-go uw
	rm -f $(JDIR)/*.o \
	&& rm -f $(JDIR)/*.a \
	&& rm -f $(JDIR)/*.$(EXT) \
	&& rm -f $(JDIR)/*.class \
	&& rm -f $(JDIR)/*.h

