package sh

import (
	"os"
	"testing"

	"github.com/duakc/mt/gosys"
	"github.com/stretchr/testify/assert"
)

func TestShellCommand(t *testing.T) {
	type testCases struct {
		Input        string
		Output       Shell
		ShellCommand string
		ShellArg     []string
	}
	t.Run("linux", func(t *testing.T) {
		if !gosys.IsLinux || !gosys.IsDarwin {
			return
		}
		var cases = []testCases{
			{Input: "/bin/bash", Output: ShellBash, ShellCommand: "bash", ShellArg: []string{"-c"}},
			{Input: "/bin/zsh", Output: ShellZsh, ShellCommand: "zsh", ShellArg: []string{"-c"}},
			{Input: "/bin/fish", Output: ShellFish, ShellCommand: "fish", ShellArg: []string{"-c"}},
			{Input: "/bin/dash", Output: ShellDash, ShellCommand: "dash", ShellArg: []string{"-c"}},
			{Input: "/bin/sh", Output: ShellSh, ShellCommand: "sh", ShellArg: []string{"-c"}},
			// nix or nix-darwin
			{Input: "/run/current-system/sw/bin/bash", Output: ShellBash, ShellCommand: "bash", ShellArg: []string{"-c"}},
			// bad input
			{Input: "this_not_is_a_shell", Output: ShellSh, ShellCommand: "sh", ShellArg: []string{"-c"}},
		}
		systemShell, _ := os.LookupEnv("SHELL")
		defer os.Setenv("SHELL", systemShell)
		for i, c := range cases {
			err := os.Setenv("SHELL", c.Input)
			assert.NoErrorf(t, err, "SetEnv")
			shell := DefaultShell()
			assert.Equalf(t, c.Output, shell, "testCase.Index=%d", i)
			cmd, args := ShellCommand(shell)
			assert.Equalf(t, c.ShellCommand, cmd, "testCase.Index=%d", i)
			assert.Equalf(t, c.ShellArg, args, "testCase.Index=%d", i)
		}
	})
}
