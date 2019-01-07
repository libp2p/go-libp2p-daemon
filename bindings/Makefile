SHELL := /bin/sh

include ../config.mk

CC = gcc
CFLAGS = -O2 -fPIC
LFLAGS = $(OS_LFLAGS) -shared

JAVA_HOME = $(shell java -XshowSettings:properties -version 2>&1 > /dev/null | grep 'java.home' | sed 's/\s*java.home = //' | sed 's/\/jre//')
JAVA_INCLUDES = -I$(JAVA_HOME)/include/$(OS) -I$(JAVA_HOME)/include
CLASS_PATH = .
vpath %.class $(CLASS_PATH)

DNAME = p2pd

.PHONY : java-daemon daemon-control-so clean

java-daemon: $(DNAME).class 

daemon-control-so: lib$(DNAME).$(EXT)

lib%.$(EXT): java-%.o go-%.a
	$(CC) $(LFLAGS) -o $@ $^

java-%.o: go-%.a
	$(CC) $(CFLAGS) -c java/java-$*.c $(JAVA_INCLUDES) -o $@

go-p2p.a: 
	go build -o $@ -buildmode=c-archive main.go

go-%.a: 
	go build -o $@ -buildmode=c-archive ../$*/main.go

%.class:
	cd java/examples && javac $*.java && mv $@ ../../$@

clean:
	rm -f *.o \
	&& rm -f *.a \
	&& rm -f *.$(EXT) \
	&& rm -f *.class \
	&& rm -f *.h
