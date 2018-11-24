CLASS_PATH = .
CFLAGS = -O2 -fPIC
DIR = p2pclient/java
JAVA_INCLUDES = -I$(JAVA_HOME)/include/darwin -I$(JAVA_HOME)/include -I$(JAVA_HOME)/include/linux
vpath %.class $(CLASS_PATH)

.DEFAULT_GOAL := go-daemon

java-daemon: libp2pd.jnilib

libp2pd.jnilib : libp2pd.o
	ld -dylib -flat_namespace -undefined suppress -macosx_version_min 10.13.4 -o $(DIR)/$@ $(DIR)/*.o -L$(DIR) -lp2pd

libp2pd.o : libp2pd.a
	gcc $(CFLAGS) -c $(DIR)/p2pd.c $(JAVA_INCLUDES) -o $(DIR)/$@

libp2pd.a: p2pd.h
	go build -o $(DIR)/$@ -buildmode=c-archive p2pd/main.go

p2pd.h : p2pd.class
	cd $(DIR) && javac -h $(CLASS_PATH) $*.java

p2pd.class: go-daemon
	cd $(DIR) && javac p2pd.java

go-daemon: deps
	go install ./...

deps: gx
	gx --verbose install --global
	gx-go rewrite

gx:
	go get github.com/whyrusleeping/gx
	go get github.com/whyrusleeping/gx-go

clean:
	rm -f p2pclient/java/*.o \
	&& rm -f p2pclient/java/*.a \
	&& rm -f p2pclient/java/*.jnilib \
	&& rm -f p2pclient/java/*.class \
	&& rm -f p2pclient/java/*.h

