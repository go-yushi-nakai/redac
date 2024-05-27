package redac

import (
	"fmt"
	"runtime/debug"
)

var (
	version  string
	revision string
)

func GetVersion() string {
	if version != "" || revision != "" {
		return fmt.Sprintf("%s(%s)", version, revision)
	}
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		return buildInfo.Main.Version
	}
	return ""
}
