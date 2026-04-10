package verifier

import (
	"os"
	"path/filepath"
	"testing"
)

func TestComputeFileHash(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	content := "Hello, World! This is a test file for SHA256 hashing."
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	hash, err := ComputeFileHash(testFile)
	if err != nil {
		t.Fatalf("ComputeFileHash error: %v", err)
	}

	if len(hash) != 64 {
		t.Errorf("expected 64-char hex hash, got %d chars", len(hash))
	}

	// Verify same content produces same hash
	hash2, err := ComputeFileHash(testFile)
	if err != nil {
		t.Fatalf("ComputeFileHash second call error: %v", err)
	}

	if hash != hash2 {
		t.Errorf("same file should produce same hash: %s != %s", hash, hash2)
	}
}

func TestVerifyFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "verify.txt")

	content := "test content for verification"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	correctHash, err := ComputeFileHash(testFile)
	if err != nil {
		t.Fatalf("ComputeFileHash error: %v", err)
	}

	// Should pass with correct hash
	if err := VerifyFile(testFile, correctHash); err != nil {
		t.Errorf("VerifyFile with correct hash should not error: %v", err)
	}

	// Should pass with "hash  filename" format
	if err := VerifyFile(testFile, correctHash+"  verify.txt"); err != nil {
		t.Errorf("VerifyFile with 'hash  filename' format should not error: %v", err)
	}

	// Should fail with wrong hash
	if err := VerifyFile(testFile, "0000000000000000000000000000000000000000000000000000000000000000"); err == nil {
		t.Error("VerifyFile with wrong hash should error")
	}

	// Should fail with empty hash
	if err := VerifyFile(testFile, ""); err == nil {
		t.Error("VerifyFile with empty hash should error")
	}
}

func TestParseChecksumFile(t *testing.T) {
	content := `# Checksums
abc123def456  sysworker-linux-amd64
789abc012def  sysworker-windows-amd64.exe
fedcba098765 *sysworker-darwin-arm64
`

	checksums := ParseChecksumFile(content)

	if len(checksums) != 3 {
		t.Errorf("expected 3 checksums, got %d", len(checksums))
	}

	if checksums["sysworker-linux-amd64"] != "abc123def456" {
		t.Errorf("linux checksum mismatch: got %q", checksums["sysworker-linux-amd64"])
	}

	if checksums["sysworker-windows-amd64.exe"] != "789abc012def" {
		t.Errorf("windows checksum mismatch: got %q", checksums["sysworker-windows-amd64.exe"])
	}

	if checksums["sysworker-darwin-arm64"] != "fedcba098765" {
		t.Errorf("darwin checksum mismatch: got %q", checksums["sysworker-darwin-arm64"])
	}
}
