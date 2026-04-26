//go:build !debug

package debug

func IsTestEnv() bool {
	return false
}
