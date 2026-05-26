//go:build unix

package sh_test

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/duakc/mt/sh"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCaptured(shell sh.Shell) (*sh.Cmd, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	c := sh.NewShell(shell)
	c.Stdout = buf
	c.Stderr = buf
	c.Stdin = nil
	return c, buf
}

func TestRun_EchoCommand_Unix(t *testing.T) {
	c, buf := newCaptured(sh.ShellSh)
	err := c.Run("echo hello")
	require.NoError(t, err)
	assert.Equal(t, "hello\n", buf.String())
}

func TestRun_EnvIsPassedToChild_Unix(t *testing.T) {
	c, buf := newCaptured(sh.ShellSh)
	c.Env("_MT_TEST_VAR", "sentinel_value")
	err := c.Run("echo $_MT_TEST_VAR")
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "sentinel_value")
}

func TestRun_WorkDirIsRespected_Unix(t *testing.T) {
	tmpDir := t.TempDir()
	c, buf := newCaptured(sh.ShellSh)
	c.WorkDir = tmpDir
	err := c.Run("pwd")
	require.NoError(t, err)
	assert.Contains(t, strings.TrimSpace(buf.String()), strings.TrimSuffix(tmpDir, "/"))
}

func TestRun_FailingCommandReturnsExitCode(t *testing.T) {
	c, _ := newCaptured(sh.ShellSh)
	err := c.Run("exit 42")
	require.Error(t, err)
	assert.Equal(t, 42, sh.ExitCode(err))
}

func TestRunContext_CancelStopsCommand(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	c, _ := newCaptured(sh.ShellSh)
	err := c.RunContext(ctx, "sleep 10")
	require.Error(t, err, "expected an error after context timeout")
}

func TestRun_PackageLevelHelpers(t *testing.T) {
	// Redirect stdout to /dev/null so passing `true` / failing `false` don't pollute test output.
	old := os.Stdout
	devNull, _ := os.Open(os.DevNull)
	os.Stdout = devNull
	defer func() { os.Stdout = old; devNull.Close() }()

	assert.NoError(t, sh.Run("true"))
	assert.Error(t, sh.Run("false"))
}
