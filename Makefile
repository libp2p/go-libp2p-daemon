SHELL := /bin/bash
OS = $(shell uname -s | tr '[:upper:]' '[:lower:]')

ifeq ($(OS), linux)
	EXT = so
	OS_LFLAGS =
else ifeq ($(OS), darwin)
	EXT = dylib
	OS_LFLAGS = -mmacosx-version-min=$(shell defaults read loginwindow SystemVersionStampAsString) -framework CoreFoundation -framework Security
endif

CC = gcc
CFLAGS = -O2 -fPIC
LFLAGS = $(OS_LFLAGS) -shared

JAVA_HOME = $(shell java -XshowSettings:properties -version 2>&1 > /dev/null | grep 'java.home' | sed 's/\s*java.home = //')
JAVA_INCLUDES = -I$(JAVA_HOME)/include/$(OS) -I$(JAVA_HOME)/include
CLASS_PATH = .
vpath %.class $(CLASS_PATH)

DDIR = p2pd
CDIR = p2pc
JDIR = p2pclient/java
DNAME = p2pd
CNAME = p2pc

.PHONY : all java-daemon java-client go-client go-daemon deps gx clean
.DEFAULT_GOAL : go-daemon

all: deps go-daemon go-client java-daemon java-client

java-daemon: lib$(DNAME).$(EXT)

java-client: lib$(CNAME).$(EXT)

lib%.$(EXT): java-%.o go-%.a
	$(CC) $(LFLAGS) -o $(JDIR)/$@ $(JDIR)/$(firstword $^) $(JDIR)/$(lastword $^)

java-%.o: go-%.a java-%.h %.class 
	$(CC) $(CFLAGS) -c $(JDIR)/java-$*.c $(JAVA_INCLUDES) -o $(JDIR)/$@

go-%.a: 
	go build -o $(JDIR)/$@ -buildmode=c-archive $*/main.go

java-%.h:
	cd $(JDIR) && javac -h $(CLASS_PATH) $*.java && mv $*.h $@

%.class:
	cd $(JDIR) && javac $*.java

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
	rm -f $(JDIR)/*.o \
	&& rm -f $(JDIR)/*.a \
	&& rm -f $(JDIR)/*.$(EXT) \
	&& rm -f $(JDIR)/*.class \
	&& rm -f $(JDIR)/*.h

