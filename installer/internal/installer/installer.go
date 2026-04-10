package installer

import (
        "fmt"
        "io"
        "os"
        "path/filepath"

        "github.com/Zam83-AZE/zaur-test/installer/internal/detect"
        "github.com/Zam83-AZE/zaur-test/installer/internal/downloader"
        "github.com/Zam83-AZE/zaur-test/installer/internal/service"
        "github.com/Zam83-AZE/zaur-test/installer/internal/verifier"
)

// Config holds all installer configuration.
type Config struct {
        RepoOwner  string // GitHub repo owner
        RepoName   string // GitHub repo name
        Version    string // Version to install ("latest" or tag like "v1.0.0")
        Token      string // GitHub token (optional)
        InstallDir string // Where to install binary
        Port       int    // Worker port
        LogLevel   string // Worker log level
        Force      bool   // Skip confirmation
        Uninstall  bool   // Uninstall mode
        Verify     bool   // Verify checksum
}

// Installer handles the installation orchestration.
type Installer struct {
        config   Config
        platform detect.Platform
        log      io.Writer
}

// New creates a new Installer with the given configuration.
func New(cfg Config, log io.Writer) *Installer {
        platform := detect.Detect()

        // Set defaults
        if cfg.InstallDir == "" {
                cfg.InstallDir = platform.DefaultInstallDir()
        }
        if cfg.Port == 0 {
                cfg.Port = 8088
        }
        if cfg.LogLevel == "" {
                cfg.LogLevel = "INFO"
        }

        return &Installer{
                config:   cfg,
                platform: platform,
                log:      log,
        }
}

// Run executes the installer based on configuration.
func (inst *Installer) Run() error {
        inst.Printf("System Worker Installer %s\n", inst.platform)
        inst.Printf("Platform: %s/%s\n\n", inst.platform.GOOS, inst.platform.GOARCH)

        if inst.config.Uninstall {
                return inst.uninstall()
        }
        return inst.install()
}

func (inst *Installer) install() error {
        // Step 1: Detect platform
        inst.Printf("[1/6] Platform detected: %s\n", inst.platform)
        inst.Printf("       Binary name: %s\n", inst.platform.BinaryName())
        inst.Printf("       Install dir: %s\n", inst.config.InstallDir)

        // Step 2: Download
        inst.Printf("\n[2/6] Downloading worker binary...\n")

        dl := downloader.New(inst.config.RepoOwner, inst.config.RepoName, inst.config.Token)

        var release *downloader.ReleaseInfo
        var err error

        if inst.config.Version == "latest" || inst.config.Version == "" {
                inst.Printf("       Fetching latest release...\n")
                release, err = dl.GetLatestRelease()
        } else {
                inst.Printf("       Fetching release %s...\n", inst.config.Version)
                release, err = dl.GetReleaseByTag(inst.config.Version)
        }

        if err != nil {
                return fmt.Errorf("failed to fetch release: %w\n\n"+
                        "Make sure:\n"+
                        "  1. Repository '%s/%s' exists and is accessible\n"+
                        "  2. A release exists with assets for %s\n"+
                        "  3. If repo is private, provide -token flag\n",
                        inst.config.RepoOwner, inst.config.RepoName, inst.platform)
        }

        inst.Printf("       Release: %s\n", release.TagName)
        if release.Name != "" {
                inst.Printf("       Name:    %s\n", release.Name)
        }

        // Find the right asset for this platform
        binaryName := inst.platform.BinaryName()
        asset, err := release.FindAsset(binaryName)
        if err != nil {
                inst.Printf("       Available assets:\n")
                for _, a := range release.Assets {
                        inst.Printf("         - %s (%d bytes)\n", a.Name, a.Size)
                }
                return fmt.Errorf("no asset found for platform: %w", err)
        }

        inst.Printf("       Asset:   %s (%d bytes)\n", asset.Name, asset.Size)

        // Download to temp location
        tmpDir, err := os.MkdirTemp("", "sysworker-install-*")
        if err != nil {
                return fmt.Errorf("failed to create temp directory: %w", err)
        }
        defer os.RemoveAll(tmpDir)

        tmpPath := filepath.Join(tmpDir, binaryName)
        inst.Printf("       Downloading to %s...\n", tmpPath)

        if err := dl.DownloadAsset(asset, tmpPath); err != nil {
                return fmt.Errorf("download failed: %w", err)
        }
        inst.Printf("       Download complete.\n")

        // Step 3: Verify checksum
        if inst.config.Verify {
                inst.Printf("\n[3/6] Verifying checksum...\n")

                checksumContent, err := dl.DownloadChecksum(release)
                if err != nil || checksumContent == "" {
                        inst.Printf("       WARNING: No checksums file found in release. Skipping verification.\n")
                } else {
                        checksums := verifier.ParseChecksumFile(checksumContent)
                        expectedHash, ok := checksums[binaryName]
                        if !ok {
                                inst.Printf("       WARNING: No checksum for %s in checksums file. Skipping verification.\n", binaryName)
                        } else {
                                if err := verifier.VerifyFile(tmpPath, expectedHash); err != nil {
                                        os.Remove(tmpPath)
                                        return fmt.Errorf("checksum verification failed: %w", err)
                                }
                                inst.Printf("       Checksum verified OK.\n")
                        }
                }
        } else {
                inst.Printf("\n[3/6] Skipping checksum verification (use -verify to enable).\n")
        }

        // Step 4: Remove old version
        inst.Printf("\n[4/6] Cleaning up old installation...\n")
        mgr, err := service.NewManager(inst.platform.ServiceName())
        if err != nil {
                return fmt.Errorf("failed to create service manager: %w", err)
        }

        if mgr.IsInstalled() {
                inst.Printf("       Stopping existing service...\n")
                if err := mgr.Stop(); err != nil {
                        inst.Printf("       WARNING: Failed to stop service: %v\n", err)
                }
                inst.Printf("       Uninstalling old service...\n")
                if err := mgr.Uninstall(); err != nil {
                        inst.Printf("       WARNING: Failed to uninstall service: %v\n", err)
                }
        } else {
                inst.Printf("       No existing installation found.\n")
        }

        // Remove old binary
        installPath := inst.platform.InstallPath(inst.config.InstallDir)
        if _, err := os.Stat(installPath); err == nil {
                inst.Printf("       Removing old binary: %s\n", installPath)
                os.Remove(installPath)
        }

        // Step 5: Install new binary
        inst.Printf("\n[5/6] Installing new binary...\n")

        if err := os.MkdirAll(inst.config.InstallDir, 0755); err != nil {
                return fmt.Errorf("failed to create install directory %s: %w", inst.config.InstallDir, err)
        }

        if err := copyFile(tmpPath, installPath); err != nil {
                return fmt.Errorf("failed to install binary: %w", err)
        }
        inst.Printf("       Binary installed to: %s\n", installPath)

        // Step 6: Install and start service
        inst.Printf("\n[6/6] Configuring system service...\n")

        dataDir := inst.platform.DefaultDataDir()
        if err := os.MkdirAll(filepath.Join(dataDir, "cert"), 0755); err != nil {
                inst.Printf("       WARNING: Failed to create cert directory: %v\n", err)
        }
        if err := os.MkdirAll(filepath.Join(dataDir, "logs"), 0755); err != nil {
                inst.Printf("       WARNING: Failed to create log directory: %v\n", err)
        }

        svcConfig := service.Config{
                BinaryPath: installPath,
                DataDir:    dataDir,
                Port:       inst.config.Port,
                LogLevel:   inst.config.LogLevel,
        }

        if err := mgr.Install(svcConfig); err != nil {
                inst.Printf("       WARNING: Failed to install system service: %v\n", err)
                inst.Printf("       The binary is installed but NOT registered as a service.\n")
                inst.Printf("       You can run it manually: %s\n", installPath)
                inst.Printf("\n")
        } else {
                inst.Printf("       Service installed successfully.\n")
                inst.Printf("       Starting service...\n")
                if err := mgr.Start(); err != nil {
                        inst.Printf("       WARNING: Failed to start service: %v\n", err)
                        inst.Printf("       Try starting manually with the platform's service manager.\n")
                } else {
                        inst.Printf("       Service started!\n")
                }
        }

        // Success summary
        inst.Printf("\n========================================\n")
        inst.Printf("  Installation Complete!\n")
        inst.Printf("========================================\n")
        inst.Printf("  Version:     %s\n", release.TagName)
        inst.Printf("  Binary:      %s\n", installPath)
        inst.Printf("  Data Dir:    %s\n", dataDir)
        inst.Printf("  Port:        %d\n", inst.config.Port)
        inst.Printf("  HTTPS URL:   https://localhost:%d\n", inst.config.Port)
        inst.Printf("  Health:      https://localhost:%d/health\n", inst.config.Port)
        inst.Printf("  Data:        https://localhost:%d/data\n", inst.config.Port)
        inst.Printf("  Logs:        https://localhost:%d/logs\n", inst.config.Port)
        inst.Printf("========================================\n")

        return nil
}

func (inst *Installer) uninstall() error {
        inst.Printf("[1/3] Removing system service...\n")

        mgr, err := service.NewManager(inst.platform.ServiceName())
        if err != nil {
                return fmt.Errorf("failed to create service manager: %w", err)
        }

        if !mgr.IsInstalled() {
                inst.Printf("       Service is not installed.\n")
        } else {
                status, _ := mgr.Status()
                inst.Printf("       Service status: %s\n", status)

                if status == "running" {
                        inst.Printf("       Stopping service...\n")
                        if err := mgr.Stop(); err != nil {
                                inst.Printf("       WARNING: Failed to stop: %v\n", err)
                        }
                }

                inst.Printf("       Uninstalling service...\n")
                if err := mgr.Uninstall(); err != nil {
                        inst.Printf("       WARNING: Failed to uninstall service: %v\n", err)
                } else {
                        inst.Printf("       Service removed.\n")
                }
        }

        // Step 2: Remove binary
        inst.Printf("\n[2/3] Removing binary...\n")
        installPath := inst.platform.InstallPath(inst.config.InstallDir)
        if _, err := os.Stat(installPath); err == nil {
                if err := os.Remove(installPath); err != nil {
                        inst.Printf("       WARNING: Failed to remove binary: %v\n", err)
                } else {
                        inst.Printf("       Binary removed: %s\n", installPath)
                }
        } else {
                inst.Printf("       Binary not found at %s\n", installPath)
        }

        // Step 3: Ask about data removal
        inst.Printf("\n[3/3] Data directory preserved: %s\n", inst.platform.DefaultDataDir())
        inst.Printf("       (certificates, logs, and configuration)\n")
        inst.Printf("       Remove manually if desired.\n")

        inst.Printf("\nUninstall complete!\n")
        return nil
}

// Printf writes a formatted message to the log writer.
func (inst *Installer) Printf(format string, args ...interface{}) {
        if inst.log != nil {
                fmt.Fprintf(inst.log, format, args...)
        }
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
        srcFile, err := os.Open(src)
        if err != nil {
                return err
        }
        defer srcFile.Close()

        dstFile, err := os.Create(dst)
        if err != nil {
                return err
        }
        defer dstFile.Close()

        if _, err := io.Copy(dstFile, srcFile); err != nil {
                return err
        }

        // Preserve executable permission
        info, err := os.Stat(src)
        if err != nil {
                return err
        }

        return os.Chmod(dst, info.Mode())
}

// IsPlatformSupported checks if the current platform is supported for installation.
func IsPlatformSupported() bool {
        os := detect.Detect().GOOS
        switch os {
        case "linux", "windows", "darwin":
                return true
        default:
                return false
        }
}

// GetServiceName returns the standard service name.
func GetServiceName() string {
        return detect.Detect().ServiceName()
}

// GetPlistIdentifier returns the macOS launchd plist identifier.
func GetPlistIdentifier() string {
        return "com.sysworker." + detect.Detect().ServiceName()
}

// GetSystemdUnitName returns the Linux systemd unit name.
func GetSystemdUnitName() string {
        return detect.Detect().ServiceName() + ".service"
}


