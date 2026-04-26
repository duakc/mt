//go:build debug

package debug

import (
	"os"
	"strings"
	"sync"
)

var (
	testEnvOnce sync.Once
	isTestEnv   = false
)

// inspired by https://github.com/nekomeowww/xo/blob/main/env.go#L10-L19

func IsTestEnv() bool {
	testEnvOnce.Do(func() {
		for _, arg := range os.Args {
			if strings.HasPrefix(arg, "-test.") {
				isTestEnv = true
				break
			}
		}
	})

	return isTestEnv
}
