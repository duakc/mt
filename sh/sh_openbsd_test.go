package sh

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnixShellFromString_OpenBSD(t *testing.T) {
	assert.Equal(t, ShellKsh, unixShellFromString("unknown"))
}
