package sh_test

import (
	"testing"

	"github.com/duakc/mt/sh"

	"github.com/stretchr/testify/assert"
)

func TestShellCommand_Windows(t *testing.T) {
	cases := []struct {
		shell    sh.Shell
		wantCmd  string
		wantArgs []string
	}{
		{sh.ShellCmd, "cmd.exe", []string{"/c"}},
		{sh.ShellPowerShell, "powershell.exe", []string{"-Command"}},
	}
	for _, tc := range cases {
		t.Run(tc.shell.String(), func(t *testing.T) {
			t.Parallel()
			cmd, args := sh.ShellCommand(tc.shell)
			assert.Equal(t, tc.wantCmd, cmd)
			assert.Equal(t, tc.wantArgs, args)
		})
	}
}

func TestDefaultBecomeMethod_WindowsExcludesUnixEscalators(t *testing.T) {
	m := sh.DefaultBecomeMethod()
	for _, banned := range []sh.BecomeMethod{sh.BecomeUseDoas, sh.BecomeUsePkexec, sh.BecomeUseSu} {
		assert.NotEqual(t, banned, m, "Windows should not return %v", banned)
	}
}
