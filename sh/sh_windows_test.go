package sh

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShellCommand_Windows(t *testing.T) {
	cmd, args := ShellCommand(ShellCmd)
	assert.Equal(t, "cmd.exe", cmd)
	assert.Equal(t, []string{"/c"}, args)

	cmd, args = ShellCommand(ShellPowerShell)
	assert.Equal(t, "powershell.exe", cmd)
	assert.Equal(t, []string{"-Command"}, args)
}

func TestDefaultBecomeMethod_WindowsNeverDoas(t *testing.T) {
	m := DefaultBecomeMethod()
	assert.NotEqual(t, BecomeUseDoas, m, "Windows shouldn't use doas")
	assert.NotEqual(t, BecomeUsePkexec, m, "Windows shouldn't use pkexec")
	assert.NotEqual(t, BecomeUseSu, m, "Windows shouldn't use su")
}
