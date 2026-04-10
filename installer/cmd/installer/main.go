package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Zam83-AZE/zaur-test/installer/internal/installer"
	"github.com/Zam83-AZE/zaur-test/installer/pkg/version"
)

func main() {
	// Configuration flags
	repo := flag.String("repo", "Zam83-AZE/zaur-test", "GitHub repository (owner/repo)")
	ver := flag.String("version", "latest", "Version to install (tag or 'latest')")
	token := flag.String("token", "", "GitHub token (for private repos)")
	installDir := flag.String("install-dir", "", "Installation directory (default: platform-specific)")
	port := flag.Int("port", 8088, "Worker HTTPS port")
	logLevel := flag.String("log-level", "INFO", "Worker log level (DEBUG, INFO, WARN, ERROR)")
	force := flag.Bool("force", false, "Skip confirmation prompts")
	uninstall := flag.Bool("uninstall", false, "Uninstall the worker")
	verify := flag.Bool("verify", false, "Verify binary checksum before installation")
	showVersion := flag.Bool("version", false, "Show installer version")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "System Worker Installer %s\n\n", version.Version)
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Installs System Worker as a system service from GitHub Releases.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s                    # Install latest version\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -version v1.0.0   # Install specific version\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -uninstall        # Remove worker service\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -port 9090        # Install with custom port\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -verify           # Verify checksum before install\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nSupported platforms: Linux (systemd), Windows (SCM), macOS (launchd)\n")
	}

	flag.Parse()

	if *showVersion {
		fmt.Printf("Installer %s (built: %s, %s)\n", version.Version, version.BuildDate, version.GoVersion)
		os.Exit(0)
	}

	// Parse repo owner/name
	owner, name := parseRepo(*repo)
	if owner == "" || name == "" {
		fmt.Fprintf(os.Stderr, "ERROR: Invalid repository format '%s'. Expected 'owner/repo'.\n", *repo)
		os.Exit(1)
	}

	// Check if the environment has SYSDATA_DIR for data directory override
	if os.Getenv("SYSDATA_DIR") != "" {
		fmt.Fprintf(os.Stderr, "INFO: SYSDATA_DIR is set to '%s'\n", os.Getenv("SYSDATA_DIR"))
	}

	cfg := installer.Config{
		RepoOwner:  owner,
		RepoName:   name,
		Version:    *ver,
		Token:      *token,
		InstallDir: *installDir,
		Port:       *port,
		LogLevel:   *logLevel,
		Force:      *force,
		Uninstall:  *uninstall,
		Verify:     *verify,
	}

	inst := installer.New(cfg, os.Stdout)

	if err := inst.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: %v\n", err)
		os.Exit(1)
	}
}

func parseRepo(repo string) (owner, name string) {
	for i, c := range repo {
		if c == '/' {
			return repo[:i], repo[i+1:]
		}
	}
	return repo, ""
}
