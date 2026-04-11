# System Worker

Cross-platform system monitoring service that collects hardware and OS information, exposes it via a local HTTPS API, and runs as a background system service.

## Architecture

| Component | Description |
|-----------|-------------|
| **Worker** | Background HTTPS service on `localhost:8088` collecting system specs |
| **Installer** | Downloads worker from GitHub Releases, installs as system service |

```
+-------------------+     GitHub Releases      +-------------------+
|    Installer      | ======================> |    Worker         |
| (one-time setup)  |   downloads binary      | (system service)  |
+-------------------+                         +-------------------+
                                                      |
                                               HTTPS :8088
                                                      |
                                              +-------+-------+
                                              |  /health      |
                                              |  /data        |
                                              |  /logs        |
                                              +---------------+
```

## How the Installer Works

The installer is a **lightweight bootstrap binary** (~5MB) that you download from GitHub Releases. When you run it, it handles everything automatically:

```
Step 1: Detect           Step 2: Fetch           Step 3: Download
+-----------------+     +-------------------+    +------------------+
| Detect OS &     | --> | Query GitHub      | -> | Download worker  |
| Architecture    |     | Releases API for  |    | binary for your  |
| (linux/amd64,   |     | latest/specified  |    | platform from    |
| windows/amd64,  |     | release version   |    | GitHub Releases  |
| darwin/arm64)   |     |                   |    |                  |
+-----------------+     +-------------------+    +------------------+
                                                          |
Step 4: Verify           Step 5: Cleanup         Step 6: Install
+-----------------+     +-------------------+    +------------------+
| Verify SHA256   | <-- | Stop old service  | <- | Remove old       |
| checksum against|     | if upgrading      |    | worker binary    |
| checksums.txt   |     +-------------------+    +------------------+
+-----------------+                                      |
                                                          v
                                                   +------------------+
                                                   | Copy new worker  |
                                                   | binary to system |
                                                   | path, set perms  |
                                                   +------------------+
                                                          |
                                                          v
                                                   +------------------+
                                                   | Install as       |
                                                   | system service   |
                                                   | (auto-start on   |
                                                   | boot)            |
                                                   +------------------+
```

### What actually happens on your machine:

1. **You download the installer** (e.g. `installer-linux-amd64`) from the [Releases](https://github.com/Zam83-AZE/zaur-test/releases) page
2. **You run it** with `sudo ./installer-linux-amd64`
3. **The installer detects your platform** (OS + CPU architecture)
4. **It calls GitHub Releases API** to find the matching worker binary (`sysworker-linux-amd64`)
5. **It downloads the worker binary** to a temporary directory
6. **It verifies the SHA256 checksum** against `checksums.txt` from the same release
7. **It stops any previous version** of the service if one exists
8. **It copies the new binary** to the system path (`/usr/local/bin/sysworker` on Linux)
9. **It registers the worker as a system service** that auto-starts on boot:
   - **Linux** → creates a systemd unit file, runs `systemctl enable --now`
   - **Windows** → registers with Service Control Manager
   - **macOS** → creates a launchd plist in `~/Library/LaunchAgents/`
10. **The worker starts immediately** and begins serving on `https://localhost:8088`

> **TL;DR**: You only need to download and run the installer once. It downloads the actual worker binary for you, verifies it, and sets it up as a permanent background service. No manual configuration required.

## Features

### Worker
- Self-signed TLS certificates (auto-generated, auto-trusted)
- System information collection: OS, BIOS, Baseboard, CPU, Memory, Disk, Network, GPU, User
- Rotating file logger (10MB/file, 30 days retention, 5 rotated files)
- Access logging for every API request
- Graceful degradation for permissions and trust store

### Installer
- Downloads latest (or specific version) worker binary from GitHub Releases
- SHA256 checksum verification
- Cross-platform service installation:
  - **Linux**: systemd unit with security hardening
  - **Windows**: Service Control Manager (SCM)
  - **macOS**: launchd plist (user-level agent)
- Uninstall support with data preservation
- Graceful fallback if service setup fails

## Platforms

| Platform | Architecture | Service Manager | Status |
|----------|-------------|-----------------|--------|
| Linux | amd64 | systemd | Tested |
| Windows | amd64 | SCM | CI passes |
| macOS | amd64 (Intel) | launchd | CI passes |
| macOS | arm64 (Apple Silicon) | launchd | CI passes |

## Quick Start

### 1. Download Installer

Download the installer for your platform from [Releases](https://github.com/Zam83-AZE/zaur-test/releases/latest).

### 2. Run Installer

**Linux:**
```bash
chmod +x installer-linux-amd64
sudo ./installer-linux-amd64
```

**Windows (Admin PowerShell):**
```powershell
.\installer-windows-amd64.exe
```

**macOS:**
```bash
chmod +x installer-darwin-arm64
./installer-darwin-arm64
```

### 3. Verify

```bash
curl -sk https://localhost:8088/health
# {"status":"ok"}

curl -sk https://localhost:8088/data | python3 -m json.tool
```

## CLI Flags

### Worker

| Flag | Default | Description |
|------|---------|-------------|
| `-port` | `8088` | HTTPS server port |
| `-cert-dir` | `~/.sysworker/cert` | TLS certificate directory |
| `-log-dir` | `~/.sysworker/logs` | Log output directory |
| `-log-level` | `INFO` | Log level: DEBUG, INFO, WARN, ERROR |
| `-version` | | Print version and exit |

### Installer

| Flag | Default | Description |
|------|---------|-------------|
| `-repo` | `Zam83-AZE/zaur-test` | GitHub repository (owner/repo) |
| `-version` | `latest` | Version to install (tag or "latest") |
| `-token` | | GitHub token (for private repos) |
| `-install-dir` | platform-specific | Installation directory |
| `-port` | `8088` | Worker HTTPS port |
| `-log-level` | `INFO` | Worker log level |
| `-verify` | `false` | Verify SHA256 checksum |
| `-uninstall` | `false` | Remove worker service and binary |
| `-force` | `false` | Skip confirmation |
| `-installer-version` | | Print installer version |

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check - returns `{"status":"ok"}` |
| `/data` | GET | Full system information JSON |
| `/logs` | GET | Recent log entries (`?limit=20&type=access`) |
| `/logs/download` | GET | Download full log file |

### Example: /data response

```json
{
  "version": "v1.0.0",
  "hostname": "my-pc",
  "os": {
    "name": "Ubuntu 24.04 LTS",
    "kernel": "6.8.0-48-generic",
    "arch": "x86_64"
  },
  "bios": {
    "vendor": "American Megatrends Inc.",
    "version": "1.2.3",
    "release_date": "2024-01-15"
  },
  "cpu": {
    "model": "Intel Core i7-9750H",
    "physical_cores": 6,
    "logical_cores": 12,
    "frequency_mhz": 2600
  },
  "memory": {
    "total_gb": 15.5,
    "available_gb": 8.2
  },
  "disks": [...],
  "network": [...],
  "gpu": [...],
  "current_user": {...}
}
```

## File Locations

| File | Linux | macOS | Windows |
|------|-------|-------|---------|
| Binary | `/usr/local/bin/sysworker` | `~/.sysworker/bin/sysworker` | `C:\Program Files\SysWorker\sysworker.exe` |
| Certificates | `~/.sysworker/cert/` | `~/.sysworker/cert/` | `C:\ProgramData\SysWorker\cert\` |
| Logs | `~/.sysworker/logs/` | `~/.sysworker/logs/` | `C:\ProgramData\SysWorker\logs\` |
| systemd unit | `/etc/systemd/system/sysworker.service` | - | - |
| launchd plist | - | `~/Library/LaunchAgents/sysworker.plist` | - |

Environment variable `SYSDATA_DIR` overrides the default data directory.

## Service Management

### Linux (systemd)
```bash
sudo systemctl status sysworker
sudo systemctl stop sysworker
sudo systemctl start sysworker
journalctl -u sysworker -f
```

### macOS (launchd)
```bash
launchctl list | grep sysworker
launchctl unload ~/Library/LaunchAgents/sysworker.plist
launchctl load ~/Library/LaunchAgents/sysworker.plist
```

### Windows (SCM)
```powershell
sc.exe query sysworker
sc.exe stop sysworker
sc.exe start sysworker
```

## Uninstall

```bash
./installer-linux-amd64 -uninstall
```

Removes the service and binary. Data directory (`~/.sysworker/`) is preserved for manual cleanup.

## Development

### Project Structure

```
zaur-test/
  worker/                    # System Worker service
    cmd/worker/main.go       # Entry point
    internal/
      certmanager/            # TLS cert generation + trust store
      collector/              # System info collectors (per-OS)
      httpserver/             # HTTPS server + routes
      logger/                 # Rotating file logger
      models/                 # Data structs
    pkg/version/              # Build-injected version
    go.mod

  installer/                  # Installer application
    cmd/installer/main.go     # Entry point with CLI flags
    internal/
      detect/                  # OS/arch detection
      downloader/              # GitHub Releases API client
      installer/               # Install orchestration
      service/                 # Service managers (per-OS)
      verifier/                # SHA256 checksum verification
    pkg/version/               # Build-injected version
    go.mod

  .github/workflows/
    ci.yml                     # CI: lint, test, build, smoke test
    release.yml                # Release: tag → build → GitHub Release
```

### Build Locally

```bash
# Worker
cd worker
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o sysworker ./cmd/worker/

# Installer
cd installer
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o installer ./cmd/installer/
```

### Create a Release

```bash
git tag v1.0.0
git push origin v1.0.0
```

The release workflow will build all 4 platform binaries, generate SHA256 checksums, and create a GitHub Release with all artifacts.

## Tech Stack

- **Go 1.22** with build tags for cross-platform support
- **crypto/x509** for self-signed TLS certificates
- **systemd/SCM/launchd** for service management
- **GitHub Actions** for CI/CD (lint, test, cross-compile, smoke test, release)
- **SHA-256** for binary integrity verification
