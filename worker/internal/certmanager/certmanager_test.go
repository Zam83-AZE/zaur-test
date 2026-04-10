package certmanager

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateCertificates(t *testing.T) {
	tmpDir := t.TempDir()
	cm := New(tmpDir)

	if cm.certsExist() {
		t.Error("should not exist yet")
	}

	if err := cm.EnsureCertificates(); err != nil {
		t.Fatalf("EnsureCertificates: %v", err)
	}

	if !cm.certsExist() {
		t.Error("should exist after generation")
	}

	certData, _ := os.ReadFile(cm.CertFile)
	keyData, _ := os.ReadFile(cm.KeyFile)

	if len(certData) == 0 {
		t.Error("cert.pem is empty")
	}
	if len(keyData) == 0 {
		t.Error("key.pem is empty")
	}
	if !strings.HasPrefix(string(certData), "-----BEGIN CERTIFICATE-----") {
		t.Error("cert.pem wrong PEM header")
	}
	if !strings.HasPrefix(string(keyData), "-----BEGIN RSA PRIVATE KEY-----") {
		t.Error("key.pem wrong PEM header")
	}
}

func TestIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	cm := New(tmpDir)

	cm.EnsureCertificates()
	d1, _ := os.ReadFile(cm.CertFile)
	cm.EnsureCertificates()
	d2, _ := os.ReadFile(cm.CertFile)

	if string(d1) != string(d2) {
		t.Error("cert regenerated on second call")
	}
}

func TestCertDirCreation(t *testing.T) {
	tmpDir := t.TempDir()
	nested := filepath.Join(tmpDir, "a", "b", "c")
	cm := New(nested)

	cm.EnsureCertificates()

	if _, err := os.Stat(nested); os.IsNotExist(err) {
		t.Error("nested dir not created")
	}
}

func TestGetCertPaths(t *testing.T) {
	tmpDir := t.TempDir()
	cm := New(tmpDir)
	c, k := cm.GetCertPaths()
	if c != filepath.Join(tmpDir, "cert.pem") {
		t.Errorf("cert path: got %q", c)
	}
	if k != filepath.Join(tmpDir, "key.pem") {
		t.Errorf("key path: got %q", k)
	}
}
