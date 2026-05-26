package sh_test

import (
	"testing"

	"github.com/duakc/mt/gosys"
	"github.com/duakc/mt/sh"

	"github.com/stretchr/testify/assert"
)

func TestShellString(t *testing.T) {
	cases := []struct {
		shell sh.Shell
		want  string
	}{
		{sh.ShellSh, "sh"},
		{sh.ShellCmd, "cmd"},
		{sh.ShellPowerShell, "powershell"},
		{sh.ShellBash, "bash"},
		{sh.ShellZsh, "zsh"},
		{sh.ShellFish, "fish"},
		{sh.ShellDash, "dash"},
		{sh.ShellAsh, "ash"},
		{sh.ShellMksh, "mksh"},
		{sh.ShellCsh, "csh"},
		{sh.ShellTcsh, "tcsh"},
		{sh.ShellRksh, "rksh"},
		{sh.ShellKsh, "ksh"},
		{sh.Shell(255), "sh"}, // unknown falls back to sh
	}
	for _, c := range cases {
		t.Run(c.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, c.want, c.shell.String())
		})
	}
}

func TestShellCommand_UnixShells(t *testing.T) {
	unixShells := []sh.Shell{
		sh.ShellSh, sh.ShellBash, sh.ShellZsh, sh.ShellFish,
		sh.ShellDash, sh.ShellAsh, sh.ShellMksh,
		sh.ShellCsh, sh.ShellTcsh, sh.ShellRksh, sh.ShellKsh,
	}
	for _, s := range unixShells {
		t.Run(s.String(), func(t *testing.T) {
			t.Parallel()
			cmd, args := sh.ShellCommand(s)
			assert.Equal(t, s.String(), cmd)
			assert.Equal(t, []string{"-c"}, args)
		})
	}
}

func TestDefaultShell_IsValid(t *testing.T) {
	s := sh.DefaultShell()
	assert.NotEqual(t, sh.ShellUseDefault, s, "DefaultShell() must not return ShellUseDefault — would cause a loop")
	assert.NotEmpty(t, s.String())
	assert.Equal(t, s, sh.DefaultShell(), "DefaultShell() must be idempotent")
}

func TestDefaultShell_MatchesPlatform(t *testing.T) {
	s := sh.DefaultShell()
	switch {
	case gosys.IsWindows:
		assert.True(t, s == sh.ShellCmd || s == sh.ShellPowerShell)
	case gosys.IsLinux || gosys.IsDarwin || gosys.IsFreebsd ||
		gosys.IsOpenbsd || gosys.IsNetbsd || gosys.IsDragonfly ||
		gosys.IsAndroid || gosys.IsHurd:
		assert.NotEqual(t, sh.ShellCmd, s, "ShellCmd is windows-only")
		assert.NotEqual(t, sh.ShellPowerShell, s, "ShellPowerShell is windows-only")
	}
}
