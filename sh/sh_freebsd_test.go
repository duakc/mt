package sh

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnixShellFromString_FreeBSD(t *testing.T) {
	assert.Equal(t, ShellSh, unixShellFromString("/bin/sh"))
	assert.Equal(t, ShellSh, unixShellFromString("unknown"))
}
