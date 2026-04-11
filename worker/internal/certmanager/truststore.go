package certmanager

// InstallToSystemTrustStore attempts to install the self-signed certificate
// into the OS trust store. Platform-specific implementations are in
// truststore_linux.go, truststore_darwin.go, truststore_windows.go.
// Falls back to no-op on unsupported platforms.
func (cm *CertManager) InstallToSystemTrustStore() error {
	installer := &trustInstaller{certFile: cm.CertFile}
	return installer.install()
}
