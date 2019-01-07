OS = $(shell uname -s | tr '[:upper:]' '[:lower:]')

ifeq ($(OS), linux)
	EXT = so
	OS_LFLAGS =
else ifeq ($(OS), darwin)
	EXT = dylib
	OS_LFLAGS = -mmacosx-version-min=$(shell defaults read loginwindow SystemVersionStampAsString) -framework CoreFoundation -framework Security
endif

DDIR = p2pd
BDIR = bindings