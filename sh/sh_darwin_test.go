package sh

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnixShellFromString_Darwin(t *testing.T) {
	cases := []struct {
		path string
		want Shell
	}{
		{"/bin/zsh", ShellZsh},
		{"/bin/bash", ShellBash},
		{"/bin/sh", ShellSh},
		{"/bin/ksh", ShellKsh},
		{"unknown_shell", ShellZsh},
	}
	for _, c := range cases {
		assert.Equalf(t, c.want, unixShellFromString(c.path), "path=%q", c.path)
	}
}
