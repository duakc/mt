package sh

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnixShellFromString_Android(t *testing.T) {
	assert.Equal(t, ShellAsh, unixShellFromString("/bin/ash"))
	assert.Equal(t, ShellAsh, unixShellFromString("unknown"))
}
