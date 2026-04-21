//go:build linux

package sh

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnixShellFromString_Linux(t *testing.T) {
	cases := []struct {
		path string
		want Shell
	}{
		{"/bin/bash", ShellBash},
		{"/usr/bin/bash", ShellBash},
		// nix
		{"/run/current-system/sw/bin/bash", ShellBash},
		{"/bin/zsh", ShellZsh},
		{"/bin/fish", ShellFish},
		{"/bin/dash", ShellDash},
		{"/bin/sh", ShellSh},
		{"/bin/ash", ShellAsh},
		{"/bin/mksh", ShellMksh},
		{"/bin/csh", ShellCsh},
		{"/bin/tcsh", ShellTcsh},
		{"/bin/rksh", ShellRksh},
		{"/bin/ksh", ShellKsh},

		{"not_a_real_shell", ShellSh},
		{"/usr/bin/unknown_shell_xyz", ShellBash},
	}
	for _, c := range cases {
		assert.Equalf(t, c.want, unixShellFromString(c.path), "path=%q", c.path)
	}
}

func TestDefaultBecomeMethod_LinuxNeverCmd(t *testing.T) {
	if hasProgramInPath("sudo") {
		assert.Equal(t, BecomeUseSudo, DefaultBecomeMethod(),
			"once the system has sudo , use sudo prefer")
	}
}
