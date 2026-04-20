package sh

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/duakc/mt/gosys"
)

type Shell uint8

const (
	ShellUseDefault Shell = iota
	ShellSh

	// unix-like system
	ShellBash
	ShellZsh
	ShellFish

	ShellDash // Debian Almquist shell
	ShellAsh  // Almquist shell

	// windows
	ShellCmd
	ShellPowerShell

	// android
	ShellMksh // MirBSD Korn shell
)

var (
	defaultShellOnce sync.Once
	defaultShell     Shell
)

func DefaultShell() Shell {
	defaultShellOnce.Do(func() {
		var sh Shell
		if gosys.IsWindows {
			sh = ShellPowerShell
			if !hasProgramInPath("powershell.exe") {
				sh = ShellCmd
			}
		} else if gosys.IsDarwin || gosys.IsLinux {
			env, _ := os.LookupEnv("SHELL")
			x := filepath.Base(env)
			switch x {
			case "sh":
				sh = ShellSh
			case "bash":
				sh = ShellBash
			case "zsh":
				sh = ShellZsh
			case "fish":
				sh = ShellFish
			case "ash":
				sh = ShellAsh
			case "dash":
				sh = ShellDash
			default:
				sh = ShellSh
			}
		} else if gosys.IsAndroid {
			sh = ShellMksh
		} else {
			sh = ShellSh
		}
		defaultShell = sh
	})
	return defaultShell
}

func ShellCommand(s Shell) (cmd string, args []string) {
	if s == ShellUseDefault {
		s = DefaultShell()
	}
	switch s {
	case ShellSh:
		return "sh", []string{"-c"}
	case ShellBash:
		return "bash", []string{"-c"}
	case ShellZsh:
		return "zsh", []string{"-c"}
	case ShellFish:
		return "fish", []string{"-c"}
	case ShellDash:
		return "dash", []string{"-c"}
	case ShellAsh:
		return "ash", []string{"-c"}
	case ShellCmd:
		return "cmd.exe", []string{"/c"}
	case ShellPowerShell:
		return "powershell.exe", []string{"-Command"}
	case ShellMksh:
		// we still use sh here
		return "sh", []string{"-c"}
	default:
		return "sh", []string{"-c"}
	}
}
