package p2pd

import (
	"fmt"
	"runtime/debug"
	"strings"
)

const name = "go-libp2p-daemon"

var defaultUserAgent string

func init() {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		defaultUserAgent = "github.com/libp2p/" + name + "/unknown-fork"
		return
	}

	version := bi.Main.Version
	if version == "" {
		version = "(unknown)"
	}

	revision, dirty := "", ""
	for _, bs := range bi.Settings {
		switch bs.Key {
		case "vcs.revision":
			if len(bs.Value) >= 9 {
				revision = bs.Value[:9]
			} else {
				revision = bs.Value
			}
		case "vcs.modified":
			if bs.Value == "true" {
				dirty = "-dirty"
			}
		}
	}

	if strings.HasPrefix(version, "v") {
		defaultUserAgent = fmt.Sprintf("%s/%s", name, version)
	} else if revision != "" {
		defaultUserAgent = fmt.Sprintf("%s/%s%s", name, revision, dirty)
	} else {
		defaultUserAgent = fmt.Sprintf("%s@%s", bi.Main.Path, version)
	}
}

// UserAgent returns the dynamically generated user agent string.
func UserAgent() string {
	return defaultUserAgent
}
