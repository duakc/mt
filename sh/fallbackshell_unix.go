//go:build unix && !darwin

package sh

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

func fallbackShell() string {
	uidStr := strconv.Itoa(os.Getuid())
	f, err := os.Open("/etc/passwd")
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, ":")
		if len(fields) >= 7 && fields[2] == uidStr {
			return fields[6]
		}
	}
	return ""
}
