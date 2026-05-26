package sh_test

import (
	"errors"
	"io"
	"os"
	"testing"

	"github.com/duakc/mt/sh"

	"github.com/stretchr/testify/assert"
)

func TestNew_Defaults(t *testing.T) {
	c := sh.New()
	assert.Equal(t, sh.ShellUseDefault, c.Shell)
	assert.Equal(t, os.Stdin, c.Stdin)
	assert.Equal(t, os.Stdout, c.Stdout)
	assert.Equal(t, os.Stderr, c.Stderr)
}

func TestNewShell_SetsShell(t *testing.T) {
	c := sh.NewShell(sh.ShellBash)
	assert.Equal(t, sh.ShellBash, c.Shell)
}

func TestCmd_EnvBuilders(t *testing.T) {
	cases := []struct {
		name    string
		apply   func(*sh.Cmd)
		wantEnv []string
	}{
		{
			name:    "Env appends key=value",
			apply:   func(c *sh.Cmd) { c.Env("FOO", "bar") },
			wantEnv: []string{"FOO=bar"},
		},
		{
			name:    "Env is chainable and accumulates",
			apply:   func(c *sh.Cmd) { c.Env("A", "1").Env("B", "2") },
			wantEnv: []string{"A=1", "B=2"},
		},
		{
			name:    "Envs appends slice as-is",
			apply:   func(c *sh.Cmd) { c.Envs([]string{"X=1", "Y=2"}) },
			wantEnv: []string{"X=1", "Y=2"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := sh.New()
			tc.apply(c)
			for _, kv := range tc.wantEnv {
				assert.Contains(t, c.ExtendEnv, kv)
			}
		})
	}
}

func TestCmd_FluentSettersReturnReceiver(t *testing.T) {
	cases := []struct {
		name string
		call func(*sh.Cmd) *sh.Cmd
	}{
		{"Env", func(c *sh.Cmd) *sh.Cmd { return c.Env("K", "V") }},
		{"Envs", func(c *sh.Cmd) *sh.Cmd { return c.Envs([]string{"K=V"}) }},
		{"Deattach", func(c *sh.Cmd) *sh.Cmd { return c.Deattach() }},
		{"CD", func(c *sh.Cmd) *sh.Cmd { return c.CD(".") }},
		{"BecomeUser", func(c *sh.Cmd) *sh.Cmd { return c.BecomeUser("alice") }},
		{"BecomeFull", func(c *sh.Cmd) *sh.Cmd { return c.BecomeFull(sh.BecomeUseSudo, "bob", "wheel") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := sh.New()
			assert.Same(t, c, tc.call(c))
		})
	}
}

func TestCmd_Deattach_DiscardsIO(t *testing.T) {
	c := sh.New().Deattach()
	assert.Nil(t, c.Stdin)
	assert.Equal(t, io.Discard, c.Stdout)
	assert.Equal(t, io.Discard, c.Stderr)
}

func TestCmd_CD_SetsAbsoluteWorkDir(t *testing.T) {
	c := sh.New().CD(".")
	assert.NotEmpty(t, c.WorkDir)
}

func TestCmd_Become(t *testing.T) {
	cases := []struct {
		name  string
		apply func(*sh.Cmd)
		want  sh.BecomeOption
	}{
		{
			name:  "BecomeUser sets default method with empty group",
			apply: func(c *sh.Cmd) { c.BecomeUser("alice") },
			want:  sh.BecomeOption{Method: sh.BecomeUseDefault, User: "alice"},
		},
		{
			name:  "BecomeFull sets every field",
			apply: func(c *sh.Cmd) { c.BecomeFull(sh.BecomeUseSudo, "bob", "wheel") },
			want:  sh.BecomeOption{Method: sh.BecomeUseSudo, User: "bob", Group: "wheel"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := sh.New()
			tc.apply(c)
			assert.Equal(t, tc.want, c.Become)
		})
	}
}

func TestShellError_UnwrapPreservesChain(t *testing.T) {
	inner := errors.New("original")
	se := &sh.ShellError{Err: inner}
	assert.Equal(t, inner, se.Unwrap())
	assert.True(t, errors.Is(se, inner))
}
