//go:build !linux && !windows && !darwin

package certmanager

type trustInstaller struct {
	certFile string
}

func (ti *trustInstaller) install() error {
	// Not supported on this platform
	return nil
}
