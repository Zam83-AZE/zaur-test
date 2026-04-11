//go:build !windows

package main

func tryRunAsService(port int, certDir, logDir, logLevel string) bool {
	return false
}
