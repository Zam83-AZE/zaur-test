package collector

import (
	"os"
	"runtime"
	"testing"
)

func SkipOnCI(t *testing.T) {
	t.Helper()
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
		t.Skip("skipping in CI environment")
	}
}

func SkipIfNotRoot(t *testing.T) {
	t.Helper()
	if os.Getuid() != 0 {
		t.Skip("skipping - requires root")
	}
}

func IsCI() bool {
	return os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != ""
}

func SkipOnOS(t *testing.T, goos ...string) {
	t.Helper()
	for _, o := range goos {
		if runtime.GOOS == o {
			return
		}
	}
	t.Skipf("skipping - requires OS: %v, running on: %s", goos, runtime.GOOS)
}
