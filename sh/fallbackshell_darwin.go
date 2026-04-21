//go:build darwin

package sh

import (
	"os"
	"os/exec"
	"strings"
)

func fallbackShell() string {
	user := os.Getenv("USER")
	if user == "" {
		return ""
	}
	out, err := exec.Command("dscl", ".", "-read", "/Users/"+user, "UserShell").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "UserShell:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "UserShell:"))
		}
	}
	return ""
}
