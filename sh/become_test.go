package sh_test

import (
	"testing"

	"github.com/duakc/mt/sh"

	"github.com/stretchr/testify/assert"
)

func TestBecomeCommand_None(t *testing.T) {
	cmd, args := sh.BecomeCommand(sh.BecomeOption{Method: sh.BecomeNone})
	assert.Empty(t, cmd)
	assert.Nil(t, args)
}

func TestBecomeCommand_Sudo(t *testing.T) {
	cases := []struct {
		name     string
		opt      sh.BecomeOption
		wantArgs []string
	}{
		{
			name:     "no user no group",
			opt:      sh.BecomeOption{Method: sh.BecomeUseSudo},
			wantArgs: []string{"-E"},
		},
		{
			name:     "user only",
			opt:      sh.BecomeOption{Method: sh.BecomeUseSudo, User: "alice"},
			wantArgs: []string{"-E", "-u", "alice"},
		},
		{
			name:     "group only",
			opt:      sh.BecomeOption{Method: sh.BecomeUseSudo, Group: "wheel"},
			wantArgs: []string{"-E", "-g", "wheel"},
		},
		{
			name:     "user and group",
			opt:      sh.BecomeOption{Method: sh.BecomeUseSudo, User: "alice", Group: "admins"},
			wantArgs: []string{"-E", "-g", "admins", "-u", "alice"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cmd, args := sh.BecomeCommand(tc.opt)
			assert.Equal(t, "sudo", cmd)
			assert.Equal(t, tc.wantArgs, args)
		})
	}
}

func TestBecomeCommand_Su(t *testing.T) {
	cases := []struct {
		name            string
		opt             sh.BecomeOption
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:         "defaults to root",
			opt:          sh.BecomeOption{Method: sh.BecomeUseSu},
			wantContains: []string{"root", "-c"},
		},
		{
			name:            "explicit user replaces root",
			opt:             sh.BecomeOption{Method: sh.BecomeUseSu, User: "bob"},
			wantContains:    []string{"bob", "-c"},
			wantNotContains: []string{"root"},
		},
		{
			name:         "group passes --group",
			opt:          sh.BecomeOption{Method: sh.BecomeUseSu, User: "alice", Group: "wheel"},
			wantContains: []string{"--group", "wheel", "alice"},
		},
		{
			name:         "always preserves environment",
			opt:          sh.BecomeOption{Method: sh.BecomeUseSu, User: "alice"},
			wantContains: []string{"--preserve-environment"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cmd, args := sh.BecomeCommand(tc.opt)
			assert.Equal(t, "su", cmd)
			for _, want := range tc.wantContains {
				assert.Contains(t, args, want)
			}
			for _, notWant := range tc.wantNotContains {
				assert.NotContains(t, args, notWant)
			}
		})
	}
}

func TestBecomeCommand_Doas(t *testing.T) {
	cases := []struct {
		name     string
		opt      sh.BecomeOption
		wantArgs []string
	}{
		{
			name:     "no user has no args",
			opt:      sh.BecomeOption{Method: sh.BecomeUseDoas},
			wantArgs: nil,
		},
		{
			name:     "user is forwarded",
			opt:      sh.BecomeOption{Method: sh.BecomeUseDoas, User: "carol"},
			wantArgs: []string{"-u", "carol"},
		},
		{
			name:     "group is ignored",
			opt:      sh.BecomeOption{Method: sh.BecomeUseDoas, User: "carol", Group: "wheel"},
			wantArgs: []string{"-u", "carol"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cmd, args := sh.BecomeCommand(tc.opt)
			assert.Equal(t, "doas", cmd)
			if tc.wantArgs == nil {
				assert.Empty(t, args)
			} else {
				assert.Equal(t, tc.wantArgs, args)
			}
		})
	}
}

func TestBecomeCommand_Pkexec(t *testing.T) {
	cases := []struct {
		name     string
		opt      sh.BecomeOption
		wantArgs []string
	}{
		{
			name:     "no user keeps cwd only",
			opt:      sh.BecomeOption{Method: sh.BecomeUsePkexec},
			wantArgs: []string{"--keep-cwd"},
		},
		{
			name:     "user appended after --keep-cwd",
			opt:      sh.BecomeOption{Method: sh.BecomeUsePkexec, User: "dave"},
			wantArgs: []string{"--keep-cwd", "--user", "dave"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cmd, args := sh.BecomeCommand(tc.opt)
			assert.Equal(t, "pkexec", cmd)
			assert.Equal(t, tc.wantArgs, args)
			assert.Equal(t, "--keep-cwd", args[0], "pkexec must always lead with --keep-cwd")
		})
	}
}

func TestDefaultBecomeMethod_IsKnownAndStable(t *testing.T) {
	m := sh.DefaultBecomeMethod()
	valid := []sh.BecomeMethod{
		sh.BecomeNone,
		sh.BecomeUseSudo,
		sh.BecomeUseSu,
		sh.BecomeUseDoas,
		sh.BecomeUsePkexec,
	}
	assert.Contains(t, valid, m, "DefaultBecomeMethod should return a known BecomeMethod (and never BecomeUseDefault, which would loop)")
	assert.Equal(t, m, sh.DefaultBecomeMethod(), "DefaultBecomeMethod should be idempotent")
}

func TestBecomeCommand_DefaultResolvesToConcreteMethod(t *testing.T) {
	// BecomeUseDefault must resolve through DefaultBecomeMethod() to a real
	// method; if the platform has any escalator at all the resulting cmd
	// must be non-empty.
	cmd, _ := sh.BecomeCommand(sh.BecomeOption{Method: sh.BecomeUseDefault, User: "root"})
	if sh.DefaultBecomeMethod() != sh.BecomeNone {
		assert.NotEmpty(t, cmd)
	}
}
