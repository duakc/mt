package sh

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCaptured(shell Shell) (*Cmd, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	c := NewShell(shell)
	c.Stdout = buf
	c.Stderr = buf
	c.Stdin = nil
	return c, buf
}

func TestNew_DefaultsToDefaultShell(t *testing.T) {
	c := New()
	assert.Equal(t, ShellUseDefault, c.Shell)
}

func TestNewShell_SetsShell(t *testing.T) {
	c := NewShell(ShellBash)
	assert.Equal(t, ShellBash, c.Shell)
}

func TestNew_DefaultsStdioToOS(t *testing.T) {
	c := New()
	assert.Equal(t, os.Stdin, c.Stdin)
	assert.Equal(t, os.Stdout, c.Stdout)
	assert.Equal(t, os.Stderr, c.Stderr)
}

func TestCmd_Env_AppendsKV(t *testing.T) {
	c := New()
	ret := c.Env("FOO", "bar")
	assert.Same(t, c, ret, "Env should return *Cmd itself")
	assert.Contains(t, c.ExtendEnv, "FOO=bar")
}

func TestCmd_Env_MultipleCallsAccumulate(t *testing.T) {
	c := New()
	c.Env("A", "1").Env("B", "2")
	assert.Contains(t, c.ExtendEnv, "A=1")
	assert.Contains(t, c.ExtendEnv, "B=2")
}

func TestCmd_Envs_AppendsSlice(t *testing.T) {
	c := New()
	ret := c.Envs([]string{"X=1", "Y=2"})
	assert.Same(t, c, ret)
	assert.Contains(t, c.ExtendEnv, "X=1")
	assert.Contains(t, c.ExtendEnv, "Y=2")
}

func TestCmd_Deattach_SetsDiscardIO(t *testing.T) {
	c := New()
	ret := c.Deattach()
	assert.Same(t, c, ret)
	assert.Nil(t, c.Stdin)
	assert.Equal(t, io.Discard, c.Stdout)
	assert.Equal(t, io.Discard, c.Stderr)
}

func TestCmd_CD_SetsWorkDir(t *testing.T) {
	c := New()
	ret := c.CD(".")
	assert.Same(t, c, ret)
	assert.NotEmpty(t, c.WorkDir, "WorkDir should be set")
}

func TestCmd_CD_PanicsOnGetWdError(t *testing.T) {
	// documentation only
	t.Log("CD() will panic when os.Getwd() failed")
}

func TestCmd_BecomeUser_SetsOption(t *testing.T) {
	c := New()
	ret := c.BecomeUser("alice")
	assert.Same(t, c, ret)
	require.NotNil(t, c.Become)
	assert.Equal(t, BecomeUseDefault, c.Become.Method)
	assert.Equal(t, "alice", c.Become.User)
	assert.Empty(t, c.Become.Group)
}

func TestCmd_BecomeFull_SetsOption(t *testing.T) {
	c := New()
	ret := c.BecomeFull(BecomeUseSudo, "bob", "wheel")
	assert.Same(t, c, ret)
	require.NotNil(t, c.Become)
	assert.Equal(t, BecomeUseSudo, c.Become.Method)
	assert.Equal(t, "bob", c.Become.User)
	assert.Equal(t, "wheel", c.Become.Group)
}

func TestShellError_Error_WithoutBecome(t *testing.T) {
	inner := errors.New("exit status 1")
	se := &ShellError{
		ShellPath: "sh",
		ShellArgs: []string{"-c"},
		Err:       inner,
	}
	msg := se.Error()
	assert.Contains(t, msg, "sh")
	assert.Contains(t, msg, "-c")
	assert.NotContains(t, msg, "sudo")
}

func TestShellError_Error_WithBecome(t *testing.T) {
	inner := errors.New("exit status 1")
	se := &ShellError{
		ShellPath: "bash",
		ShellArgs: []string{"-c"},
		Become: BecomeOption{
			Method: BecomeUseSudo,
			User:   "root",
		},
		Err: inner,
	}
	msg := se.Error()
	assert.Contains(t, msg, "bash")
	assert.Contains(t, msg, "sudo")
}

func TestShellError_Unwrap_ReturnsInner(t *testing.T) {
	inner := errors.New("original error")
	se := &ShellError{Err: inner}
	assert.Equal(t, inner, se.Unwrap())
	assert.True(t, errors.Is(se, inner))
}

func TestShellError_ImplementsError(t *testing.T) {
	var _ error = &ShellError{}
}

func TestHasProgramInPath_NonExistentProgram(t *testing.T) {
	assert.False(t, hasProgramInPath("__this_program_does_not_exist_mt_test__"))
}
