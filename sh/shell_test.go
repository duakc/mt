package sh

import (
	"testing"

	"github.com/duakc/mt/gosys"

	"github.com/stretchr/testify/assert"
)

func TestShellString(t *testing.T) {
	cases := []struct {
		shell Shell
		want  string
	}{
		{ShellSh, "sh"},
		{ShellCmd, "cmd"},
		{ShellPowerShell, "powershell"},
		{ShellBash, "bash"},
		{ShellZsh, "zsh"},
		{ShellFish, "fish"},
		{ShellDash, "dash"},
		{ShellAsh, "ash"},
		{ShellMksh, "mksh"},
		{ShellCsh, "csh"},
		{ShellTcsh, "tcsh"},
		{ShellRksh, "rksh"},
		{ShellKsh, "ksh"},
		// unknown shell should fall back to sh
		{Shell(255), "sh"},
	}

	for _, c := range cases {
		if c.shell == ShellUseDefault {
			continue
		}
		assert.Equalf(t, c.want, c.shell.String(), "Shell(%d).String()", c.shell)
	}
}

func TestShellCommand_UnixShells(t *testing.T) {
	cases := []struct {
		shell    Shell
		wantCmd  string
		wantArgs []string
	}{
		{ShellSh, "sh", []string{"-c"}},
		{ShellBash, "bash", []string{"-c"}},
		{ShellZsh, "zsh", []string{"-c"}},
		{ShellFish, "fish", []string{"-c"}},
		{ShellDash, "dash", []string{"-c"}},
		{ShellAsh, "ash", []string{"-c"}},
		{ShellMksh, "mksh", []string{"-c"}},
		{ShellCsh, "csh", []string{"-c"}},
		{ShellTcsh, "tcsh", []string{"-c"}},
		{ShellRksh, "rksh", []string{"-c"}},
		{ShellKsh, "ksh", []string{"-c"}},
	}
	for _, c := range cases {
		gotCmd, gotArgs := ShellCommand(c.shell)
		assert.Equalf(t, c.wantCmd, gotCmd, "ShellCommand(%s) cmd", c.shell)
		assert.Equalf(t, c.wantArgs, gotArgs, "ShellCommand(%s) args", c.shell)
	}
}

func TestUnixShellFromString_EmptyPath(t *testing.T) {
	assert.Equal(t, ShellSh, unixShellFromString(""))
}

func TestDefaultShell_IsValid(t *testing.T) {
	s := DefaultShell()
	assert.NotEqual(t, ShellUseDefault, s, "DefaultShell() can not return ShellUseDefault, caused a loop")
	assert.NotEmpty(t, s.String())
}

func TestDefaultShell_IsIdempotent(t *testing.T) {
	assert.Equal(t, DefaultShell(), DefaultShell())
}

func TestDefaultShell_PlatformRange(t *testing.T) {
	s := DefaultShell()
	switch {
	case gosys.IsWindows:
		assert.True(t, s == ShellCmd || s == ShellPowerShell)
	case gosys.IsLinux || gosys.IsDarwin || gosys.IsFreebsd ||
		gosys.IsOpenbsd || gosys.IsNetbsd || gosys.IsDragonfly ||
		gosys.IsAndroid || gosys.IsHurd:
		assert.NotEqual(t, ShellCmd, s, "ShellCmd can not be use on unix-liked system")
		assert.NotEqual(t, ShellPowerShell, s, "ShellPowerShell can not be use on unix-liked system")
	}
}
