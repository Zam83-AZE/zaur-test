package detect

import (
        "strings"
        "testing"
)

func TestDetect(t *testing.T) {
        p := Detect()

        if p.GOOS == "" {
                t.Error("GOOS should not be empty")
        }
        if p.GOARCH == "" {
                t.Error("GOARCH should not be empty")
        }
        if p.OS != p.GOOS && p.GOOS != "linux" && p.GOOS != "windows" && p.GOOS != "darwin" {
                t.Errorf("OS should match GOOS for known platforms, got %s", p.OS)
        }
}

func TestBinaryName(t *testing.T) {
        tests := []struct {
                platform Platform
                expected string
        }{
                {Platform{GOOS: "linux", GOARCH: "amd64", OS: "linux", Arch: "amd64"}, "sysworker-linux-amd64"},
                {Platform{GOOS: "windows", GOARCH: "amd64", OS: "windows", Arch: "amd64"}, "sysworker-windows-amd64.exe"},
                {Platform{GOOS: "darwin", GOARCH: "arm64", OS: "darwin", Arch: "arm64"}, "sysworker-darwin-arm64"},
                {Platform{GOOS: "darwin", GOARCH: "amd64", OS: "darwin", Arch: "amd64"}, "sysworker-darwin-amd64"},
        }

        for _, tc := range tests {
                got := tc.platform.BinaryName()
                if got != tc.expected {
                        t.Errorf("BinaryName() for %s/%s = %q, want %q", tc.platform.GOOS, tc.platform.GOARCH, got, tc.expected)
                }
        }
}

func TestServiceName(t *testing.T) {
        p := Detect()
        if p.ServiceName() != "sysworker" {
                t.Errorf("ServiceName() = %q, want %q", p.ServiceName(), "sysworker")
        }
}

func TestDefaultInstallDir(t *testing.T) {
        p := Detect()
        dir := p.DefaultInstallDir()
        if dir == "" {
                t.Error("DefaultInstallDir should not be empty")
        }
}

func TestDefaultDataDir(t *testing.T) {
        p := Detect()
        dir := p.DefaultDataDir()
        if dir == "" {
                t.Error("DefaultDataDir should not be empty")
        }
        // Should contain sysworker
        if !strings.Contains(dir, "sysworker") {
                t.Errorf("DefaultDataDir %q should contain 'sysworker'", dir)
        }
}

func TestNormalizeGOARCH(t *testing.T) {
        tests := []struct {
                input    string
                expected string
        }{
                {"amd64", "amd64"},
                {"x86_64", "amd64"},
                {"arm64", "arm64"},
                {"aarch64", "arm64"},
                {"386", "386"},
                {"i386", "386"},
        }

        for _, tc := range tests {
                got := NormalizeGOARCH(tc.input)
                if got != tc.expected {
                        t.Errorf("NormalizeGOARCH(%q) = %q, want %q", tc.input, got, tc.expected)
                }
        }
}


