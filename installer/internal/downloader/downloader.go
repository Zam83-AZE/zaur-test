package downloader

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ReleaseInfo holds GitHub release metadata.
type ReleaseInfo struct {
	TagName string  `json:"tag_name"`
	Name    string  `json:"name"`
	Body    string  `json:"body"`
	Assets  []Asset `json:"assets"`
}

// Asset represents a single release asset.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
	ContentType        string `json:"content_type"`
}

// Downloader handles downloading worker binaries from GitHub Releases.
type Downloader struct {
	RepoOwner string
	RepoName  string
	Token     string // Optional GitHub token for private repos or rate limit
	Client    *http.Client
}

// New creates a new Downloader for the given GitHub repository.
func New(owner, repo, token string) *Downloader {
	return &Downloader{
		RepoOwner: owner,
		RepoName:  repo,
		Token:     token,
		Client: &http.Client{
			Timeout: 300 * time.Second, // 5 minutes for large binaries
		},
	}
}

// GetLatestRelease fetches the latest release information from GitHub.
func (d *Downloader) GetLatestRelease() (*ReleaseInfo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", d.RepoOwner, d.RepoName)

	body, err := d.doRequest(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest release: %w", err)
	}

	var release ReleaseInfo
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %w", err)
	}

	if len(release.Assets) == 0 {
		return nil, fmt.Errorf("release %s has no assets", release.TagName)
	}

	return &release, nil
}

// GetReleaseByTag fetches a specific release by tag name.
func (d *Downloader) GetReleaseByTag(tag string) (*ReleaseInfo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", d.RepoOwner, d.RepoName, tag)

	body, err := d.doRequest(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release for tag %s: %w", tag, err)
	}

	var release ReleaseInfo
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %w", err)
	}

	return &release, nil
}

// FindAsset finds a specific asset by name (case-insensitive partial match).
func (r *ReleaseInfo) FindAsset(pattern string) (*Asset, error) {
	pattern = strings.ToLower(pattern)
	for i := range r.Assets {
		if strings.ToLower(r.Assets[i].Name) == pattern {
			return &r.Assets[i], nil
		}
	}
	// Try partial match
	for i := range r.Assets {
		if strings.Contains(strings.ToLower(r.Assets[i].Name), pattern) {
			return &r.Assets[i], nil
		}
	}
	return nil, fmt.Errorf("asset matching '%s' not found in release %s", pattern, r.TagName)
}

// DownloadAsset downloads an asset to the specified destination path.
func (d *Downloader) DownloadAsset(asset *Asset, destPath string) error {
	// Ensure destination directory exists
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", destDir, err)
	}

	req, err := http.NewRequest("GET", asset.BrowserDownloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}

	if d.Token != "" {
		req.Header.Set("Authorization", "token "+d.Token)
	}

	resp, err := d.Client.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned HTTP %d for %s", resp.StatusCode, asset.BrowserDownloadURL)
	}

	// Write to temp file first, then rename
	tmpPath := destPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	size, err := io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write file: %w", err)
	}

	if asset.Size > 0 && size != asset.Size {
		os.Remove(tmpPath)
		return fmt.Errorf("download size mismatch: expected %d, got %d", asset.Size, size)
	}

	// Make executable (Unix)
	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to set executable permission: %w", err)
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// DownloadChecksum downloads the checksums file for a release and returns its content.
func (d *Downloader) DownloadChecksum(release *ReleaseInfo) (string, error) {
	asset, err := release.FindAsset("checksums")
	if err != nil {
		// Checksums file may not exist
		return "", nil
	}

	req, err := http.NewRequest("GET", asset.BrowserDownloadURL, nil)
	if err != nil {
		return "", err
	}

	if d.Token != "" {
		req.Header.Set("Authorization", "token "+d.Token)
	}

	resp, err := d.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksum download returned HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (d *Downloader) doRequest(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "SysWorker-Installer/1.0")

	if d.Token != "" {
		req.Header.Set("Authorization", "token "+d.Token)
	}

	resp, err := d.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("resource not found (404)")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
