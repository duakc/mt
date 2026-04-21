package sh

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/duakc/mt/gosys"
)

type Shell uint8

const (
	ShellUseDefault Shell = iota
	ShellSh

	ShellCmd
	ShellPowerShell

	ShellBash
	ShellZsh
	ShellFish

	ShellDash // Debian Almquist shell
	ShellAsh  // Almquist shell

	ShellMksh // MirBSD Korn shell

	ShellCsh // C shell

	ShellTcsh // TENEX C shell

	ShellRksh // Restricted Korn shell
	ShellKsh  // same as ShellRksh
)

var unixShellList = []struct {
	shells       []Shell
	defaultShell Shell
	platform     bool
}{
	{
		shells:       []Shell{ShellBash, ShellZsh, ShellFish, ShellDash, ShellSh, ShellAsh, ShellMksh, ShellCsh, ShellTcsh, ShellRksh, ShellKsh},
		defaultShell: ShellSh,
		platform:     gosys.IsLinux,
	},
	{
		shells:       []Shell{ShellZsh, ShellBash, ShellSh, ShellKsh, ShellCsh, ShellTcsh, ShellFish, ShellDash, ShellAsh, ShellMksh},
		defaultShell: ShellZsh,
		platform:     gosys.IsDarwin,
	},
	{
		shells:       []Shell{ShellSh, ShellCsh, ShellTcsh, ShellBash, ShellZsh, ShellFish, ShellMksh, ShellDash, ShellAsh, ShellRksh, ShellKsh},
		defaultShell: ShellSh,
		platform:     gosys.IsFreebsd,
	},
	{
		shells:       []Shell{ShellKsh, ShellSh, ShellCsh, ShellTcsh, ShellBash, ShellZsh, ShellFish, ShellMksh, ShellDash, ShellAsh, ShellRksh},
		defaultShell: ShellSh,
		platform:     gosys.IsOpenbsd,
	},
	{
		shells:       []Shell{ShellSh, ShellCsh, ShellTcsh, ShellKsh, ShellBash, ShellZsh, ShellFish, ShellMksh, ShellDash, ShellAsh, ShellRksh},
		defaultShell: ShellSh,
		platform:     gosys.IsNetbsd,
	},
	{
		shells:       []Shell{ShellSh, ShellCsh, ShellTcsh, ShellBash, ShellZsh, ShellFish, ShellMksh, ShellDash, ShellAsh, ShellRksh, ShellKsh},
		defaultShell: ShellSh,
		platform:     gosys.IsDragonfly,
	},
	{
		shells:       []Shell{ShellAsh, ShellMksh, ShellSh, ShellBash, ShellZsh, ShellFish, ShellDash, ShellCsh, ShellTcsh, ShellRksh, ShellKsh},
		defaultShell: ShellSh,
		platform:     gosys.IsAndroid,
	},
	{
		shells:       []Shell{ShellBash, ShellSh, ShellZsh, ShellFish, ShellDash, ShellAsh, ShellMksh, ShellCsh, ShellTcsh, ShellRksh, ShellKsh},
		defaultShell: ShellSh,
		platform:     gosys.IsHurd,
	},
}

func unixShellFromString(path string) Shell {
	if path == "" {
		return ShellSh
	}
	base := filepath.Base(path)
	for _, v := range unixShellList {
		if !v.platform {
			continue
		}
		for _, s := range v.shells {
			if base == s.String() {
				return s
			}
		}
		return v.defaultShell
	}
	return ShellSh
}

func (s Shell) String() string {
	switch s {
	case ShellUseDefault:
		return DefaultShell().String()
	case ShellSh:
		return "sh"
	case ShellCmd:
		return "cmd"
	case ShellPowerShell:
		return "powershell"
	case ShellBash:
		return "bash"
	case ShellZsh:
		return "zsh"
	case ShellFish:
		return "fish"
	case ShellDash:
		return "dash"
	case ShellAsh:
		return "ash"
	case ShellMksh:
		return "mksh"
	case ShellCsh:
		return "csh"
	case ShellTcsh:
		return "tcsh"
	case ShellRksh:
		return "rksh"
	case ShellKsh:
		return "ksh"
	default:
		return "sh"
	}
}

var (
	defaultShellOnce sync.Once
	defaultShell     Shell
)

func DefaultShell() Shell {
	defaultShellOnce.Do(func() {
		// windows
		// if `powershell.exe` existed, use PowerShell first
		// else choose `cmd.exe`.
		if gosys.IsWindows {
			if _, err := exec.LookPath("powershell.exe"); err == nil {
				defaultShell = ShellPowerShell
			} else {
				defaultShell = ShellCmd
			}
			return
		}

		// unix:
		// lookup $SHELL environment first
		// if didn't get ,read /etc/passwd for this user manually
		shellEnv := os.Getenv("SHELL")
		if shellEnv != "" {
			defaultShell = unixShellFromString(shellEnv)
		} else {
			shellPath := fallbackShell()
			if shellPath != "" {
				defaultShell = unixShellFromString(shellPath)
			} else {
				defaultShell = ShellSh
			}
		}
	})
	return defaultShell
}

func ShellCommand(s Shell) (cmd string, args []string) {
	if s == ShellUseDefault {
		s = DefaultShell()
	}
	switch s {
	case ShellCmd:
		return "cmd.exe", []string{"/c"}
	case ShellPowerShell:
		return "powershell.exe", []string{"-Command"}
	default:
		return s.String(), []string{"-c"}
	}
}
