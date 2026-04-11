package downloader

import (
        "encoding/json"
        "fmt"
        "net/http"
        "net/http/httptest"
        "os"
        "path/filepath"
        "testing"
)

func TestFindAsset(t *testing.T) {
        release := &ReleaseInfo{
                TagName: "v1.0.0",
                Assets: []Asset{
                        {Name: "sysworker-linux-amd64", BrowserDownloadURL: "https://example.com/sysworker-linux-amd64"},
                        {Name: "sysworker-windows-amd64.exe", BrowserDownloadURL: "https://example.com/sysworker-windows-amd64.exe"},
                        {Name: "checksums.txt", BrowserDownloadURL: "https://example.com/checksums.txt"},
                },
        }

        tests := []struct {
                pattern string
                want    string
                wantErr bool
        }{
                {"sysworker-linux-amd64", "https://example.com/sysworker-linux-amd64", false},
                {"checksums.txt", "https://example.com/checksums.txt", false},
                {"nonexistent", "", true},
        }

        for _, tc := range tests {
                asset, err := release.FindAsset(tc.pattern)
                if tc.wantErr {
                        if err == nil {
                                t.Errorf("FindAsset(%q) expected error, got nil", tc.pattern)
                        }
                        continue
                }
                if err != nil {
                        t.Errorf("FindAsset(%q) error: %v", tc.pattern, err)
                        continue
                }
                if asset.BrowserDownloadURL != tc.want {
                        t.Errorf("FindAsset(%q) = %q, want %q", tc.pattern, asset.BrowserDownloadURL, tc.want)
                }
        }
}

func TestFindAssetPartialMatch(t *testing.T) {
        release := &ReleaseInfo{
                TagName: "v1.0.0",
                Assets: []Asset{
                        {Name: "sysworker-linux-amd64", BrowserDownloadURL: "https://example.com/linux"},
                },
        }

        asset, err := release.FindAsset("linux")
        if err != nil {
                t.Fatalf("FindAsset partial match error: %v", err)
        }
        if !containsStr(asset.Name, "linux") {
                t.Errorf("partial match should find linux asset, got %s", asset.Name)
        }
}

func TestDownloadAsset(t *testing.T) {
        content := []byte("mock binary content")
        ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
                w.Write(content)
        }))
        defer ts.Close()

        d := New("owner", "repo", "")
        asset := &Asset{
                Name:               "test-binary",
                BrowserDownloadURL: ts.URL + "/download",
                Size:               int64(len(content)),
        }

        tmpDir := t.TempDir()
        destPath := filepath.Join(tmpDir, "test-binary")

        err := d.DownloadAsset(asset, destPath)
        if err != nil {
                t.Fatalf("DownloadAsset error: %v", err)
        }

        data, err := os.ReadFile(destPath)
        if err != nil {
                t.Fatalf("failed to read downloaded file: %v", err)
        }

        if string(data) != string(content) {
                t.Errorf("downloaded content mismatch: got %q, want %q", string(data), string(content))
        }

        // Check file is executable
        info, _ := os.Stat(destPath)
        if info.Mode()&0111 == 0 {
                t.Error("downloaded file should be executable")
        }
}

func TestGetLatestRelease(t *testing.T) {
        release := ReleaseInfo{
                TagName: "v1.0.0",
                Name:    "v1.0.0 Release",
                Assets: []Asset{
                        {Name: "sysworker-linux-amd64", BrowserDownloadURL: "https://example.com/linux", Size: 100},
                },
        }

        ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.Header().Set("Content-Type", "application/json")
                json.NewEncoder(w).Encode(release)
        }))
        defer ts.Close()

        d := New("owner", "repo", "")

        // Override the URL to use test server
        result, err := d.getRelease(ts.URL + "/repos/owner/repo/releases/latest")
        if err != nil {
                t.Fatalf("GetLatestRelease error: %v", err)
        }

        if result.TagName != "v1.0.0" {
                t.Errorf("TagName = %q, want %q", result.TagName, "v1.0.0")
        }
}

// Helper to call the internal doRequest with a custom URL
func (d *Downloader) getRelease(url string) (*ReleaseInfo, error) {
        body, err := d.doRequest(url)
        if err != nil {
                return nil, err
        }

        var release ReleaseInfo
        if err := json.Unmarshal(body, &release); err != nil {
                return nil, err
        }

        return &release, nil
}

func containsStr(s, substr string) bool {
        for i := 0; i <= len(s)-len(substr); i++ {
                if s[i:i+len(substr)] == substr {
                        return true
                }
        }
        return false
}
