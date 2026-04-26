//go:build !unix && !darwin

package sh

func fallbackShell() string {
	return ""
}
