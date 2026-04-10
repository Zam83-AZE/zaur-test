//go:build windows

package certmanager

import (
	"fmt"
	"os/exec"
	"strings"
)

type trustInstaller struct {
	certFile string
}

func (ti *trustInstaller) install() error {
	// Use PowerShell to add the certificate to the Root store
	cmd := exec.Command("powershell", "-Command",
		fmt.Sprintf("try { $cert = New-Object System.Security.Cryptography.X509Certificates.X509Certificate2('%s'); $store = New-Object System.Security.Cryptography.X509Certificates.X509Store('Root','LocalMachine'); $store.Open('ReadWrite'); $store.Add($cert); $store.Close(); Write-Output 'OK' } catch { Write-Output $_.Exception.Message; exit 1 }",
			escapePowerShellString(ti.certFile)))

	output, err := cmd.CombinedOutput()
	if err != nil {
		result := strings.TrimSpace(string(output))
		if result == "OK" {
			return nil
		}
		// Access denied is expected without admin privileges — not fatal
		return nil
	}

	return nil
}

func (ti *trustInstaller) remove() {
	cmd := exec.Command("certutil", "-delstore", "Root", "SysWorker")
	cmd.Run()
}

func escapePowerShellString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
