package certmanager

import (
	"fmt"
	"os"
)

// InstallToSystemTrustStore attempts to install the self-signed certificate
// into the OS trust store. This allows browsers to trust the cert without
// manual intervention. Requires elevated privileges on most platforms.
// Returns nil on success or if platform is not supported.
// Returns an error describing what went wrong if installation fails.
func (cm *CertManager) InstallToSystemTrustStore() error {
	installer := &trustInstaller{certFile: cm.CertFile}
	return installer.install()
}
