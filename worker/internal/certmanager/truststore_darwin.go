//go:build darwin

package certmanager

import (
        "fmt"
        "os"
        "os/exec"
)

type trustInstaller struct {
        certFile string
}

func (ti *trustInstaller) install() error {
        // macOS uses security command to add trusted certificates
        // Requires: sudo or admin privileges

        // First check if already installed
        cmd := exec.Command("security", "verify-cert", "-c", ti.certFile)
        if err := cmd.Run(); err == nil {
                // Already trusted
                return nil
        }

        // Add to login keychain (or system keychain with sudo)
        // Try system keychain first, fall back to login keychain
        keychains := []string{
                "/Library/Keychains/System.keychain",    // System-wide (needs sudo)
                os.Getenv("HOME") + "/Library/Keychains/login.keychain-db", // User's login keychain
        }

        for _, keychain := range keychains {
                if _, err := os.Stat(keychain); err != nil {
                        continue
                }

                cmd := exec.Command("security", "add-trusted-cert", "-k", keychain, ti.certFile)
                if output, err := cmd.CombinedOutput(); err != nil {
                        // Try with -r flag for root certificate
                        cmd2 := exec.Command("security", "add-trusted-cert", "-r", "-k", keychain, ti.certFile)
                        if output2, err2 := cmd2.CombinedOutput(); err2 != nil {
                                // Log but don't fail — user can manually trust
                                _ = fmt.Sprintf("trust store error: %s / %s", string(output), string(output2))
                                continue
                        }
                }
                return nil
        }

        // Could not add to any keychain — not fatal
        return nil
}

func (ti *trustInstaller) remove() {
        cmd := exec.Command("security", "remove-trusted-cert", ti.certFile)
        cmd.Run()
}
