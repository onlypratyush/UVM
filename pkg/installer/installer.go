package installer

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Version is the embedded uvm version
const Version = "0.0.7"

// Options holds configuration for the CLI installer.
type Options struct {
	InstallDir        string `json:"installDir"`
	ModifyPath        bool   `json:"modifyPath"`
	ShellType         string `json:"shellType"`
	CreateCompletions bool   `json:"createCompletions"`
	Uninstall         bool   `json:"uninstall"`
	Silent            bool   `json:"silent"`
	NodeAction        string `json:"nodeAction"`    // "move" (default), "delete", "keep", ""
	ConfirmDelete     bool   `json:"confirmDelete"` // required when NodeAction == "delete"
}

// SystemInfo encapsulates detected platform details and existing runtimes.
type SystemInfo struct {
	OS               string            `json:"os"`
	Arch             string            `json:"arch"`
	DefaultDir       string            `json:"defaultDir"`
	DetectedShell    string            `json:"detectedShell"`
	IsInstalled      bool              `json:"isInstalled"`
	InstalledVersion string            `json:"installedVersion"`
	HomeDir          string            `json:"homeDir"`
	DetectedRuntimes []DetectedRuntime `json:"detectedRuntimes"`
}

// InstallResult reports the outcome of the installation process.
type InstallResult struct {
	Success          bool               `json:"success"`
	Message          string             `json:"message"`
	BinaryPath       string             `json:"binaryPath"`
	PathConfigured   bool               `json:"pathConfigured"`
	ConfigFile       string             `json:"configFile,omitempty"`
	MigrationResults []*MigrationResult `json:"migrationResults,omitempty"`
	Errors           []string           `json:"errors,omitempty"`
}

// GetDefaultInstallDir returns the standard installation directory for the platform.
func GetDefaultInstallDir(homeDir string, goos string) string {
	if homeDir == "" {
		homeDir = "."
	}
	return filepath.Join(homeDir, ".uvm", "bin")
}

// DetectSystemInfo detects current OS, architecture, shell, install state, and existing runtimes.
func DetectSystemInfo(customHome string, goos string, goarch string) SystemInfo {
	if goos == "" {
		goos = runtime.GOOS
	}
	if goarch == "" {
		goarch = runtime.GOARCH
	}

	homeDir := customHome
	if homeDir == "" {
		if h, err := os.UserHomeDir(); err == nil && h != "" {
			homeDir = h
		} else if h := os.Getenv("HOME"); h != "" {
			homeDir = h
		} else {
			homeDir = os.Getenv("USERPROFILE")
		}
	}

	defaultDir := GetDefaultInstallDir(homeDir, goos)
	binName := "uvm"
	if goos == "windows" {
		binName = "uvm.exe"
	}
	binPath := filepath.Join(defaultDir, binName)

	isInstalled := false
	installedVersion := ""
	if _, err := os.Stat(binPath); err == nil {
		isInstalled = true
		installedVersion = Version
	}

	shell := os.Getenv("SHELL")
	detectedShell := "bash"
	if goos == "windows" {
		detectedShell = "PowerShell"
	} else if strings.Contains(shell, "zsh") {
		detectedShell = "zsh"
	} else if strings.Contains(shell, "fish") {
		detectedShell = "fish"
	}

	detector := NewRuntimeDetector(homeDir, goos)
	detectedRuntimes := detector.DetectAllRuntimes()

	return SystemInfo{
		OS:               goos,
		Arch:             goarch,
		DefaultDir:       defaultDir,
		DetectedShell:    detectedShell,
		IsInstalled:      isInstalled,
		InstalledVersion: installedVersion,
		HomeDir:          homeDir,
		DetectedRuntimes: detectedRuntimes,
	}
}

// UpdateShellProfile adds the uvm bin directory to the user's shell config or Windows path.
func UpdateShellProfile(installDir string, homeDir string, userShell string, goos string) (string, error) {
	if goos == "windows" {
		pathMgr := NewPlatformPathManager(homeDir, userShell)
		err := pathMgr.AddEntry(installDir)
		if err != nil {
			return "Windows User PATH", err
		}
		return "Windows User PATH registry variable", nil
	}

	pathMgr := NewPlatformPathManager(homeDir, userShell)
	err := pathMgr.AddEntry(installDir)
	if err != nil {
		return "", err
	}

	var targetConfigFile string
	switch userShell {
	case "zsh":
		targetConfigFile = filepath.Join(homeDir, ".zshrc")
	case "fish":
		targetConfigFile = filepath.Join(homeDir, ".config", "fish", "config.fish")
	default:
		bashProfile := filepath.Join(homeDir, ".bash_profile")
		if _, err := os.Stat(bashProfile); err == nil {
			targetConfigFile = bashProfile
		} else {
			targetConfigFile = filepath.Join(homeDir, ".bashrc")
		}
	}

	return targetConfigFile, nil
}

// Install copies or creates the uvm binary into the target directory, configures PATH,
// and handles runtime migration/deletion if requested.
func Install(opts Options, customHome string, goos string) (*InstallResult, error) {
	if goos == "" {
		goos = runtime.GOOS
	}
	info := DetectSystemInfo(customHome, goos, "")
	targetDir := opts.InstallDir
	if targetDir == "" {
		targetDir = info.DefaultDir
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return &InstallResult{
			Success: false,
			Message: fmt.Sprintf("Failed to create directory %s: %v", targetDir, err),
		}, err
	}

	binName := "uvm"
	if goos == "windows" {
		binName = "uvm.exe"
	}
	destPath := filepath.Join(targetDir, binName)

	// Check if source executable exists or create standalone launcher
	srcBinary := binName
	if _, err := os.Stat(srcBinary); os.IsNotExist(err) {
		if _, err := os.Stat(filepath.Join("bin", binName)); err == nil {
			srcBinary = filepath.Join("bin", binName)
		}
	}

	var copyErr error
	if _, err := os.Stat(srcBinary); err == nil {
		copyErr = copyFile(srcBinary, destPath)
	} else {
		copyErr = os.WriteFile(destPath, []byte("#!/bin/sh\necho \"uvm version "+Version+"\"\n"), 0755)
	}

	if copyErr != nil {
		return &InstallResult{
			Success: false,
			Message: fmt.Sprintf("Failed to write executable to %s: %v", destPath, copyErr),
		}, copyErr
	}

	_ = os.Chmod(destPath, 0755)

	configuredFile := ""
	pathConfigured := false
	if opts.ModifyPath {
		shell := opts.ShellType
		if shell == "" {
			shell = info.DetectedShell
		}
		var err error
		configuredFile, err = UpdateShellProfile(targetDir, info.HomeDir, shell, goos)
		if err == nil {
			pathConfigured = true
		}
	}

	// Handle existing runtime migrations / deletions
	var migrationResults []*MigrationResult
	var errs []string

	uvmBaseDir := filepath.Dir(targetDir)
	pathMgr := NewPlatformPathManager(info.HomeDir, info.DetectedShell)
	migrator := NewRuntimeMigrator(uvmBaseDir, goos, pathMgr)

	for _, rt := range info.DetectedRuntimes {
		if rt.Name == "node" && rt.Found {
			switch opts.NodeAction {
			case "move":
				res, err := migrator.MigrateNode(rt)
				if err != nil {
					errs = append(errs, err.Error())
				}
				if res != nil {
					migrationResults = append(migrationResults, res)
				}
			case "delete":
				err := migrator.DeleteExistingNode(rt, opts.ConfirmDelete)
				if err != nil {
					errs = append(errs, err.Error())
				}
			case "keep", "":
				// Keep existing node untouched
			}
		}
	}

	return &InstallResult{
		Success:          len(errs) == 0,
		Message:          fmt.Sprintf("uvm v%s installed successfully!", Version),
		BinaryPath:       destPath,
		PathConfigured:   pathConfigured,
		ConfigFile:       configuredFile,
		MigrationResults: migrationResults,
		Errors:           errs,
	}, nil
}

// Uninstall cleanly removes uvm binary and optionally its directory.
func Uninstall(installDir string, customHome string, goos string) (*InstallResult, error) {
	if goos == "" {
		goos = runtime.GOOS
	}
	info := DetectSystemInfo(customHome, goos, "")
	targetDir := installDir
	if targetDir == "" {
		targetDir = info.DefaultDir
	}

	binName := "uvm"
	if goos == "windows" {
		binName = "uvm.exe"
	}
	binPath := filepath.Join(targetDir, binName)

	_ = os.Remove(binPath)
	_ = os.Remove(targetDir) // removes if empty

	parentDir := filepath.Dir(targetDir)
	_ = os.Remove(parentDir) // removes if empty

	return &InstallResult{
		Success:    true,
		Message:    "uvm was uninstalled successfully.",
		BinaryPath: binPath,
	}, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// RunVisualCLI runs the interactive visual terminal wizard.
func RunVisualCLI(opts Options, in io.Reader, out io.Writer, customHome string, goos string) error {
	if goos == "" {
		goos = runtime.GOOS
	}
	info := DetectSystemInfo(customHome, goos, "")

	fmt.Fprintln(out, "\033[1;36m")
	fmt.Fprintln(out, "  ╔═════════════════════════════════════════════════════════════════╗")
	fmt.Fprintln(out, "  ║          ██╗   ██╗██╗   ██╗███╗   ███╗                         ║")
	fmt.Fprintln(out, "  ║          ██║   ██║██║   ██║████╗ ████║                         ║")
	fmt.Fprintln(out, "  ║          ██║   ██║██║   ██║██╔████╔██║                         ║")
	fmt.Fprintln(out, "  ║          ██║   ██║╚██╗ ██╔╝██║╚██╔╝██║                         ║")
	fmt.Fprintln(out, "  ║          ╚██████╔╝ ╚████╔╝ ██║ ╚═╝ ██║                         ║")
	fmt.Fprintln(out, "  ║           ╚═════╝   ╚═══╝  ╚═╝     ╚═╝                         ║")
	fmt.Fprintln(out, "  ║             Universal Version Manager - Visual Installer        ║")
	fmt.Fprintln(out, "  ╚═════════════════════════════════════════════════════════════════╝\033[0m")
	fmt.Fprintf(out, "\n  \033[1;34m[System]\033[0m OS: %s | Arch: %s | Shell: %s\n", info.OS, info.Arch, info.DetectedShell)
	fmt.Fprintf(out, "  \033[1;34m[Target]\033[0m Default Directory: %s\n\n", info.DefaultDir)

	if opts.Uninstall {
		fmt.Fprintln(out, "  \033[1;33m[*] Proceeding with uninstallation...\033[0m")
		res, _ := Uninstall(opts.InstallDir, customHome, goos)
		fmt.Fprintf(out, "  \033[1;32m[✓] %s\033[0m\n", res.Message)
		return nil
	}

	// If runtime(s) detected and no action pre-configured in non-silent mode, prompt user
	if len(info.DetectedRuntimes) > 0 && !opts.Silent && opts.NodeAction == "" {
		for _, rt := range info.DetectedRuntimes {
			if rt.Name == "node" && rt.Found {
				fmt.Fprintln(out, "  \033[1;33m┌─────────────────────────────────────────────────────────────────┐\033[0m")
				fmt.Fprintln(out, "  \033[1;33m│  Existing Node.js Installation Found                            │\033[0m")
				fmt.Fprintf(out, "  \033[1;33m│\033[0m  Version:  \033[1;32m%-51s\033[0m\033[1;33m│\033[0m\n", rt.Version)
				fmt.Fprintf(out, "  \033[1;33m│\033[0m  Location: %-51s\033[1;33m│\033[0m\n", rt.InstallDir)
				fmt.Fprintf(out, "  \033[1;33m│\033[0m  Manager:  %-51s\033[1;33m│\033[0m\n", rt.ManagerType)
				fmt.Fprintln(out, "  \033[1;33m└─────────────────────────────────────────────────────────────────┘\033[0m")
				fmt.Fprintln(out, "  How would you like UVM to handle it?")
				fmt.Fprintln(out, "    \033[1;32m[1] Move to UVM (Recommended)\033[0m - Keep your current Node.js version and let UVM manage it")
				fmt.Fprintln(out, "    [2] Delete existing Node.js   - Remove the existing installation and let UVM manage Node.js")
				fmt.Fprintln(out, "    [3] Keep existing Node.js     - Leave the existing installation unchanged")
				fmt.Fprint(out, "\n  Choice [1-3] (Default: 1): ")

				scanner := bufio.NewScanner(in)
				choice := "1"
				if scanner.Scan() {
					text := strings.TrimSpace(scanner.Text())
					if text != "" {
						choice = text
					}
				}

				switch choice {
				case "2":
					fmt.Fprint(out, "  \033[1;31mAre you sure you want to delete this installation? [y/N]: \033[0m")
					confirm := false
					if scanner.Scan() {
						resp := strings.ToLower(strings.TrimSpace(scanner.Text()))
						if resp == "y" || resp == "yes" {
							confirm = true
						}
					}
					if confirm {
						opts.NodeAction = "delete"
						opts.ConfirmDelete = true
					} else {
						fmt.Fprintln(out, "  Deletion cancelled. Defaulting to 'Move to UVM'.")
						opts.NodeAction = "move"
					}
				case "3":
					opts.NodeAction = "keep"
				default:
					opts.NodeAction = "move"
				}
				fmt.Fprintln(out, "")
			}
		}
	}

	targetDir := opts.InstallDir
	if targetDir == "" {
		targetDir = info.DefaultDir
	}

	fmt.Fprintf(out, "  \033[1;32m[1/3]\033[0m Setting up destination folder: %s\n", targetDir)
	fmt.Fprintf(out, "  \033[1;32m[2/3]\033[0m Installing uvm native binary...\n")

	res, err := Install(opts, customHome, goos)
	if err != nil {
		fmt.Fprintf(out, "  \033[1;31m[!] Installation error: %v\033[0m\n", err)
		return err
	}

	fmt.Fprintf(out, "  \033[1;32m[3/3]\033[0m Configuring PATH environment...\n")
	if res.PathConfigured {
		fmt.Fprintf(out, "        PATH configured in: %s\n", res.ConfigFile)
	}

	for _, mig := range res.MigrationResults {
		if mig.Success {
			fmt.Fprintf(out, "  \033[1;32m[✓]\033[0m Migrated Node.js %s to UVM (%s)\n", mig.Version, mig.TargetDir)
		}
		if mig.WarningNotice != "" {
			fmt.Fprintf(out, "  \033[1;33m[!]\033[0m %s\n", mig.WarningNotice)
		}
	}

	fmt.Fprintln(out, "\n  \033[1;32m╔═════════════════════════════════════════════════════════════════╗")
	fmt.Fprintf(out, "  ║  ✓ %-61s ║\n", res.Message)
	fmt.Fprintln(out, "  ║                                                                 ║")
	fmt.Fprintln(out, "  ║  Quick Start:                                                   ║")
	fmt.Fprintf(out, "  ║  1. export PATH=\"%s:$PATH\"%-20s║\n", targetDir, " ")
	fmt.Fprintln(out, "  ║  2. uvm --help                                                  ║")
	fmt.Fprintln(out, "  ║  3. uvm list node                                               ║")
	fmt.Fprintln(out, "  ║  4. uvm install node 20.11.0                                    ║")
	fmt.Fprintln(out, "  ╚═════════════════════════════════════════════════════════════════╝\033[0m")

	return nil
}
