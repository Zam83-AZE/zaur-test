package certmanager

import (
        "crypto/rand"
        "crypto/rsa"
        "crypto/x509"
        "crypto/x509/pkix"
        "encoding/pem"
        "fmt"
        "math/big"
        "net"
        "os"
        "path/filepath"
        "time"
)

type CertManager struct {
        CertDir  string
        CertFile string
        KeyFile  string
}

func New(certDir string) *CertManager {
        return &CertManager{
                CertDir:  certDir,
                CertFile: filepath.Join(certDir, "cert.pem"),
                KeyFile:  filepath.Join(certDir, "key.pem"),
        }
}

func (cm *CertManager) EnsureCertificates() error {
        if err := os.MkdirAll(cm.CertDir, 0755); err != nil {
                return fmt.Errorf("failed to create cert directory: %w", err)
        }
        if cm.certsExist() {
                return nil
        }
        return cm.generateCertificates()
}

func (cm *CertManager) certsExist() bool {
        certInfo, certErr := os.Stat(cm.CertFile)
        keyInfo, keyErr := os.Stat(cm.KeyFile)
        if certErr != nil || keyErr != nil {
                return false
        }
        if certInfo.Size() == 0 || keyInfo.Size() == 0 {
                return false
        }
        return true
}

func (cm *CertManager) generateCertificates() error {
        privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
        if err != nil {
                return fmt.Errorf("failed to generate RSA key: %w", err)
        }

        notBefore := time.Now()
        notAfter := notBefore.AddDate(10, 0, 0)

        serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
        serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
        if err != nil {
                return fmt.Errorf("failed to generate serial number: %w", err)
        }

        template := x509.Certificate{
                SerialNumber: serialNumber,
                Subject: pkix.Name{
                        Organization: []string{"SysWorker"},
                        CommonName:   "localhost",
                },
                NotBefore:             notBefore,
                NotAfter:              notAfter,
                KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
                        ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
                        BasicConstraintsValid: true,
                        IsCA:                  true,
                DNSNames:              []string{"localhost"},
                IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
        }

        derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
        if err != nil {
                return fmt.Errorf("failed to create certificate: %w", err)
        }

        certOut, err := os.Create(cm.CertFile)
        if err != nil {
                return fmt.Errorf("failed to create cert file: %w", err)
        }
        defer certOut.Close()
        if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
                return fmt.Errorf("failed to write cert: %w", err)
        }

        keyOut, err := os.OpenFile(cm.KeyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
        if err != nil {
                return fmt.Errorf("failed to create key file: %w", err)
        }
        defer keyOut.Close()
        if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}); err != nil {
                return fmt.Errorf("failed to write key: %w", err)
        }

        return nil
}

func (cm *CertManager) GetCertPaths() (string, string) {
        return cm.CertFile, cm.KeyFile
}
