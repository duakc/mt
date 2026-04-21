//go:build unix

package sh

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasProgramInPath_ExistingProgram(t *testing.T) {
	assert.True(t, hasProgramInPath("sh"))
}

func TestRun_EchoCommand_Unix(t *testing.T) {
	c, buf := newCaptured(ShellSh)
	err := c.Run("echo hello")
	require.NoError(t, err)
	assert.Equal(t, "hello\n", buf.String())
}

func TestRun_EnvIsPassedToChild_Uinx(t *testing.T) {
	c, buf := newCaptured(ShellSh)
	c.Env("_MT_TEST_VAR", "sentinel_value")
	err := c.Run("echo $_MT_TEST_VAR")
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "sentinel_value")
}

func TestRun_WorkDirIsRespected_Unix(t *testing.T) {
	tmpDir := t.TempDir()
	c, buf := newCaptured(ShellSh)
	c.WorkDir = tmpDir
	err := c.Run("pwd")
	require.NoError(t, err)
	assert.Contains(t, strings.TrimSpace(buf.String()), strings.TrimSuffix(tmpDir, "/"))
}

func TestRun_FailingCommandReturnsError(t *testing.T) {
	c, _ := newCaptured(ShellSh)
	err := c.Run("exit 42")
	require.Error(t, err)
	require.Equal(t, int(42), ExitCode(err))
}

func TestRunContext_CancelStopsCommand(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	c, _ := newCaptured(ShellSh)
	err := c.RunContext(ctx, "sleep 10")
	require.Error(t, err, "an err required return after timeout")
}

func TestRun_PackageLevelHelpers(t *testing.T) {
	// redirect to null
	old := os.Stdout
	devNull, _ := os.Open(os.DevNull)
	os.Stdout = devNull
	defer func() { os.Stdout = old; devNull.Close() }()

	err := Run("true")
	assert.NoError(t, err)

	err = Run("false")
	assert.Error(t, err)
}
