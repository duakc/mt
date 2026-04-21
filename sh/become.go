package sh

import (
	"sync"

	"github.com/duakc/mt/gosys"
)

type BecomeMethod uint8

const (
	BecomeNone BecomeMethod = iota
	BecomeUseDefault
	BecomeUseSudo
	BecomeUseSu

	// BecomeUseDoas and BecomeUsePkexec can not set group and preserve the environments
	// to the program.
	// Even if the `doas` can not preserve the environment to the running program.
	// we still prefer use `doas` instead of `su`.
	BecomeUseDoas
	BecomeUsePkexec
)

var (
	onceBecomeMethod    sync.Once
	defaultBecomeMethod = BecomeNone
)

func DefaultBecomeMethod() BecomeMethod {
	onceBecomeMethod.Do(func() {
		defaultBecomeMethod = BecomeNone
		if gosys.IsWindows && hasProgramInPath("sudo.exe") {
			// https://github.com/microsoft/sudo
			// https://news.ycombinator.com/item?id=47828853
			defaultBecomeMethod = BecomeUseSudo
		} else if gosys.IsLinux || gosys.IsFreebsd || gosys.IsOpenbsd || gosys.IsNetbsd ||
			gosys.IsDragonfly {
			switch {
			case hasProgramInPath("sudo"):
				defaultBecomeMethod = BecomeUseSudo
			case hasProgramInPath("doas"):
				defaultBecomeMethod = BecomeUseDoas
			case gosys.IsLinux && hasProgramInPath("pkexec"):
				defaultBecomeMethod = BecomeUsePkexec
			case hasProgramInPath("su"):
				defaultBecomeMethod = BecomeUseSu
			}
		} else if gosys.IsDarwin {
			switch {
			case hasProgramInPath("sudo"):
				defaultBecomeMethod = BecomeUseSudo
			case hasProgramInPath("su"):
				defaultBecomeMethod = BecomeUseSu
			}
		} else if gosys.IsAndroid {
			if hasProgramInPath("su") {
				defaultBecomeMethod = BecomeUseSu
			}
		}
	})

	return defaultBecomeMethod
}

type BecomeOption struct {
	Method BecomeMethod
	User   string
	Group  string
}

func BecomeCommand(option BecomeOption) (string, []string) {
	if option.Method == BecomeUseDefault {
		option.Method = DefaultBecomeMethod()
	}

	if option.Method == BecomeNone {
		return "", nil
	}

	switch option.Method {
	case BecomeUseSudo:
		arg := []string{"-E"}
		if option.Group != "" {
			arg = append(arg, "-g", option.Group)
		}
		if option.User != "" {
			arg = append(arg, "-u", option.User)
		}

		return "sudo", arg
	case BecomeUseSu:
		if option.User == "" {
			option.User = "root"
		}
		arg := []string{"--preserve-environment"}
		if option.Group != "" {
			arg = append(arg, "--group", option.Group)
		}
		return "su", append(arg, option.User, "-c")
	case BecomeUseDoas:
		var arg []string
		if option.User != "" {
			arg = append(arg, "-u", option.User)
		}
		return "doas", arg
	case BecomeUsePkexec:
		arg := []string{"--keep-cwd"}
		if option.User != "" {
			arg = append(arg, "--user", option.User)
		}
		return "pkexec", arg
	default:
		return "", nil
	}
}
