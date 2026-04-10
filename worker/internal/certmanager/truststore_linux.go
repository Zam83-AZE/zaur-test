//go:build linux

package certmanager

import (
        "fmt"
        "os"
        "os/exec"
        "path/filepath"
)

type trustInstaller struct {
        certFile string
}

func (ti *trustInstaller) install() error {
        // Target directory for local certificates
        // Debian/Ubuntu: /usr/local/share/ca-certificates/
        // RHEL/Fedora: /etc/pki/ca-trust/source/anchors/
        certDirs := []struct {
                dir        string
                filename   string
                updateCmd  string
        }{
                {"/usr/local/share/ca-certificates", "sysworker.crt", "update-ca-certificates"},
                {"/etc/pki/ca-trust/source/anchors", "sysworker.crt", "update-ca-trust"},
        }

        certData, err := os.ReadFile(ti.certFile)
        if err != nil {
                return fmt.Errorf("cannot read cert file: %w", err)
        }

        for _, target := range certDirs {
                // Check if directory exists (implies we're on this distro)
                if _, err := os.Stat(target.dir); err != nil {
                        continue
                }

                destPath := filepath.Join(target.dir, target.filename)

                // Check if already installed (compare content)
                if existing, err := os.ReadFile(destPath); err == nil {
                        if string(existing) == string(certData) {
                                // Already installed and up to date
                                return nil
                        }
                }

                // Copy cert to system trust store
                if err := os.WriteFile(destPath, certData, 0644); err != nil {
                        return fmt.Errorf("failed to copy cert to %s: %w", destPath, err)
                }

                // Run update command
                cmd := exec.Command(target.updateCmd)
                if output, err := cmd.CombinedOutput(); err != nil {
                        return fmt.Errorf("failed to run %s: %w\nOutput: %s", target.updateCmd, err, string(output))
                }

                return nil
        }

        // No known trust store found — not a fatal error
        return nil
}

func (ti *trustInstaller) remove() {
        certDirs := []struct {
                dir      string
                filename string
        }{
                {"/usr/local/share/ca-certificates", "sysworker.crt"},
                {"/etc/pki/ca-trust/source/anchors", "sysworker.crt"},
        }

        for _, target := range certDirs {
                destPath := filepath.Join(target.dir, target.filename)
                os.Remove(destPath)
        }

        // Try update commands
        for _, cmd := range []string{"update-ca-certificates --fresh", "update-ca-trust extract"} {
                exec.Command("sh", "-c", cmd).Run()
        }
}
