package verifier

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

// VerifyFile checks a file's SHA256 checksum against an expected value.
// The expected hash can be just the hex string (64 chars) or in "hash  filename" format.
func VerifyFile(filePath, expectedHash string) error {
	if expectedHash == "" {
		return fmt.Errorf("empty expected hash")
	}

	// Strip any leading "hash  filename" format - take first field
	if parts := strings.Fields(expectedHash); len(parts) > 0 {
		expectedHash = strings.ToLower(parts[0])
	} else {
		expectedHash = strings.ToLower(strings.TrimSpace(expectedHash))
	}

	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file for verification: %w", err)
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	computedHash := hex.EncodeToString(hasher.Sum(nil))
	computedHash = strings.ToLower(computedHash)

	if computedHash != expectedHash {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, computedHash)
	}

	return nil
}

// ComputeFileHash computes the SHA256 hash of a file and returns it as a hex string.
func ComputeFileHash(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// ParseChecksumFile parses a checksums.txt content and returns a map of filename -> hash.
// Supports both "hash  filename" and "hash *filename" formats.
func ParseChecksumFile(content string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		hash := strings.ToLower(parts[0])
		filename := parts[len(parts)-1] // Last field is filename

		// Strip leading * or space from filename
		filename = strings.TrimPrefix(filename, "*")
		filename = strings.TrimPrefix(filename, " ")

		result[filename] = hash
	}

	return result
}
