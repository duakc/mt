package sh

import (
	"testing"

	"github.com/duakc/mt/gosys"
	"github.com/stretchr/testify/assert"
)

func TestUnixShellFromString_FallbackOnUnknownPlatform(t *testing.T) {
	isKnownUnix := gosys.IsLinux || gosys.IsDarwin || gosys.IsFreebsd ||
		gosys.IsOpenbsd || gosys.IsNetbsd || gosys.IsDragonfly ||
		gosys.IsAndroid || gosys.IsHurd
	if !isKnownUnix {
		assert.Equal(t, ShellSh, unixShellFromString("/bin/bash"))
	}
}
