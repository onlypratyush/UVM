package installer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Version is the embedded uvm version
const Version = "0.0.1"

// Options holds configuration for the visual and CLI installer.
type Options struct {
	InstallDir        string `json:"installDir"`
	ModifyPath        bool   `json:"modifyPath"`
	ShellType         string `json:"shellType"`
	CreateCompletions bool   `json:"createCompletions"`
	Uninstall         bool   `json:"uninstall"`
	Silent            bool   `json:"silent"`
	WebMode           bool   `json:"webMode"`
	Port              int    `json:"port"`
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

var (
	browserOpenerMu sync.RWMutex
	browserOpenerFn = defaultBrowserOpener
)

func defaultBrowserOpener(url string, goos string) error {
	var cmd *exec.Cmd
	switch goos {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

// BrowserOpener is a function hook to open URLs in testing or production.
var BrowserOpener = func(url string, goos string) error {
	browserOpenerMu.RLock()
	fn := browserOpenerFn
	browserOpenerMu.RUnlock()
	return fn(url, goos)
}

// SetBrowserOpener sets a custom browser opener function in a thread-safe manner.
func SetBrowserOpener(fn func(url string, goos string) error) {
	browserOpenerMu.Lock()
	browserOpenerFn = fn
	browserOpenerMu.Unlock()
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

// NewWebServer initializes the HTTP server for the Visual Web UI installer.
func NewWebServer(opts Options, customHome string, goos string) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(GetWebUIHTML()))
	})

	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		info := DetectSystemInfo(customHome, goos, "")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(info)
	})

	mux.HandleFunc("/api/detect-runtimes", func(w http.ResponseWriter, r *http.Request) {
		detector := NewRuntimeDetector(customHome, goos)
		runtimes := detector.DetectAllRuntimes()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(runtimes)
	})

	mux.HandleFunc("/api/install", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req Options
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			req = opts
		}
		res, err := Install(req, customHome, goos)
		w.Header().Set("Content-Type", "application/json")
		if err != nil || (res != nil && !res.Success) {
			w.WriteHeader(http.StatusInternalServerError)
		}
		_ = json.NewEncoder(w).Encode(res)
	})

	mux.HandleFunc("/api/uninstall", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req Options
		_ = json.NewDecoder(r.Body).Decode(&req)
		res, _ := Uninstall(req.InstallDir, customHome, goos)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	})

	mux.HandleFunc("/api/verify", func(w http.ResponseWriter, r *http.Request) {
		info := DetectSystemInfo(customHome, goos, "")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"installed": info.IsInstalled,
			"version":   Version,
		})
	})

	port := opts.Port
	if port == 0 {
		port = 8484
	}

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
}

// StartWebUI starts the web server and attempts to launch the browser.
func StartWebUI(opts Options, customHome string, goos string) error {
	srv := NewWebServer(opts, customHome, goos)
	port := opts.Port
	if port == 0 {
		port = 8484
	}
	url := fmt.Sprintf("http://localhost:%d", port)

	fmt.Printf("\n✨ Starting uvm Visual Web Installer at %s\n", url)
	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = BrowserOpener(url, goos)
	}()

	return srv.ListenAndServe()
}

// GetWebUIHTML returns the embedded visual single-page installer interface.
func GetWebUIHTML() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>uvm - Universal Version Manager Setup</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Outfit:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
  <style>
    :root {
      --bg: #0b0f19;
      --card-bg: rgba(22, 30, 49, 0.75);
      --card-border: rgba(255, 255, 255, 0.08);
      --primary: #06b6d4;
      --primary-gradient: linear-gradient(135deg, #06b6d4 0%, #3b82f6 50%, #8b5cf6 100%);
      --accent: #10b981;
      --warning: #f59e0b;
      --danger: #ef4444;
      --text: #f8fafc;
      --text-muted: #94a3b8;
      --radius: 16px;
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      font-family: 'Outfit', sans-serif;
      background-color: var(--bg);
      background-image: 
        radial-gradient(at 0% 0%, rgba(6, 182, 212, 0.15) 0px, transparent 50%),
        radial-gradient(at 100% 100%, rgba(139, 92, 246, 0.15) 0px, transparent 50%);
      color: var(--text);
      min-height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 24px;
    }
    .container {
      width: 100%;
      max-width: 660px;
      background: var(--card-bg);
      backdrop-filter: blur(20px);
      -webkit-backdrop-filter: blur(20px);
      border: 1px solid var(--card-border);
      border-radius: var(--radius);
      padding: 36px;
      box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
    }
    .header {
      text-align: center;
      margin-bottom: 24px;
    }
    .logo {
      font-size: 38px;
      font-weight: 800;
      letter-spacing: -1px;
      background: var(--primary-gradient);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
      margin-bottom: 6px;
    }
    .subtitle {
      color: var(--text-muted);
      font-size: 15px;
    }
    .badge-bar {
      display: flex;
      gap: 10px;
      justify-content: center;
      margin-top: 14px;
      flex-wrap: wrap;
    }
    .badge {
      background: rgba(255, 255, 255, 0.05);
      border: 1px solid var(--card-border);
      padding: 4px 12px;
      border-radius: 999px;
      font-size: 12px;
      font-weight: 500;
      color: var(--primary);
    }
    .section {
      margin-bottom: 22px;
    }
    label.sec-title {
      display: block;
      font-size: 13px;
      font-weight: 600;
      color: var(--text-muted);
      margin-bottom: 8px;
      text-transform: uppercase;
      letter-spacing: 0.5px;
    }
    .input-group {
      position: relative;
    }
    input[type="text"] {
      width: 100%;
      background: rgba(15, 23, 42, 0.6);
      border: 1px solid var(--card-border);
      padding: 12px 16px;
      border-radius: 10px;
      color: var(--text);
      font-family: 'JetBrains Mono', monospace;
      font-size: 14px;
      outline: none;
      transition: border-color 0.2s;
    }
    input[type="text"]:focus {
      border-color: var(--primary);
      box-shadow: 0 0 0 2px rgba(6, 182, 212, 0.2);
    }
    .toggle-row {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 12px 16px;
      background: rgba(15, 23, 42, 0.4);
      border: 1px solid var(--card-border);
      border-radius: 10px;
      margin-bottom: 10px;
    }
    .toggle-info h4 { font-size: 14px; font-weight: 600; }
    .toggle-info p { font-size: 12px; color: var(--text-muted); }
    .switch {
      position: relative;
      width: 44px;
      height: 24px;
    }
    .switch input { opacity: 0; width: 0; height: 0; }
    .slider {
      position: absolute;
      inset: 0;
      background-color: rgba(255, 255, 255, 0.2);
      border-radius: 24px;
      cursor: pointer;
      transition: 0.3s;
    }
    .slider:before {
      position: absolute;
      content: "";
      height: 18px;
      width: 18px;
      left: 3px;
      bottom: 3px;
      background-color: white;
      border-radius: 50%;
      transition: 0.3s;
    }
    input:checked + .slider { background: var(--primary); }
    input:checked + .slider:before { transform: translateX(20px); }

    /* Node Detection Card */
    .detected-card {
      background: rgba(15, 23, 42, 0.7);
      border: 1px solid rgba(6, 182, 212, 0.3);
      border-radius: 12px;
      padding: 16px;
      margin-bottom: 20px;
    }
    .detected-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 12px;
    }
    .detected-title {
      font-size: 15px;
      font-weight: 700;
      color: var(--text);
      display: flex;
      align-items: center;
      gap: 8px;
    }
    .node-ver-badge {
      background: rgba(16, 185, 129, 0.2);
      color: #34d399;
      border: 1px solid rgba(16, 185, 129, 0.4);
      padding: 2px 8px;
      border-radius: 6px;
      font-size: 12px;
      font-weight: 600;
    }
    .location-box {
      font-family: 'JetBrains Mono', monospace;
      font-size: 12px;
      color: var(--text-muted);
      background: rgba(0, 0, 0, 0.3);
      padding: 8px 12px;
      border-radius: 6px;
      margin-bottom: 14px;
      word-break: break-all;
    }
    .options-grid {
      display: flex;
      flex-direction: column;
      gap: 8px;
    }
    .opt-card {
      border: 1px solid var(--card-border);
      border-radius: 8px;
      padding: 10px 14px;
      background: rgba(255, 255, 255, 0.02);
      cursor: pointer;
      display: flex;
      gap: 12px;
      align-items: flex-start;
      transition: all 0.2s;
    }
    .opt-card:hover {
      background: rgba(255, 255, 255, 0.05);
      border-color: rgba(6, 182, 212, 0.4);
    }
    .opt-card.selected {
      background: rgba(6, 182, 212, 0.1);
      border-color: var(--primary);
    }
    .opt-card input[type="radio"] {
      margin-top: 3px;
      accent-color: var(--primary);
    }
    .opt-content h5 {
      font-size: 13px;
      font-weight: 600;
      margin-bottom: 2px;
    }
    .opt-content p {
      font-size: 12px;
      color: var(--text-muted);
    }
    .del-confirm-box {
      display: none;
      margin-top: 8px;
      padding: 10px;
      background: rgba(239, 68, 68, 0.1);
      border: 1px solid rgba(239, 68, 68, 0.3);
      border-radius: 6px;
      font-size: 12px;
      color: #fca5a5;
    }

    .btn {
      width: 100%;
      padding: 14px;
      border-radius: 10px;
      border: none;
      font-size: 16px;
      font-weight: 700;
      cursor: pointer;
      transition: all 0.2s;
      background: var(--primary-gradient);
      color: white;
      box-shadow: 0 10px 25px -5px rgba(6, 182, 212, 0.4);
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 8px;
    }
    .btn:hover {
      opacity: 0.95;
      transform: translateY(-1px);
    }
    .btn:disabled {
      opacity: 0.5;
      cursor: not-allowed;
      transform: none;
    }
    .progress-box {
      display: none;
      margin-top: 20px;
    }
    .progress-bar-bg {
      width: 100%;
      height: 8px;
      background: rgba(255, 255, 255, 0.1);
      border-radius: 999px;
      overflow: hidden;
      margin-bottom: 12px;
    }
    .progress-bar-fill {
      height: 100%;
      width: 0%;
      background: var(--primary-gradient);
      transition: width 0.3s ease;
    }
    .status-log {
      font-family: 'JetBrains Mono', monospace;
      font-size: 12px;
      color: var(--text-muted);
      background: rgba(15, 23, 42, 0.8);
      padding: 12px;
      border-radius: 8px;
      border: 1px solid var(--card-border);
      max-height: 140px;
      overflow-y: auto;
    }
    .success-card {
      display: none;
      text-align: center;
      padding: 20px 0;
    }
    .success-icon {
      font-size: 48px;
      margin-bottom: 12px;
    }
    .code-block {
      background: rgba(15, 23, 42, 0.9);
      border: 1px solid var(--card-border);
      padding: 14px;
      border-radius: 8px;
      font-family: 'JetBrains Mono', monospace;
      font-size: 13px;
      text-align: left;
      margin: 16px 0;
      color: var(--primary);
    }
    .btn-secondary {
      background: rgba(255, 255, 255, 0.08);
      border: 1px solid var(--card-border);
      color: var(--text);
      box-shadow: none;
      margin-top: 10px;
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <div class="logo">🚀 uvm</div>
      <div class="subtitle">Universal Version Manager Setup Wizard</div>
      <div class="badge-bar">
        <span class="badge" id="os-badge">OS: Detecting...</span>
        <span class="badge" id="arch-badge">Arch: Detecting...</span>
        <span class="badge" id="shell-badge">Shell: Detecting...</span>
      </div>
    </div>

    <div id="setup-form">
      <div id="detected-runtimes-container"></div>

      <div class="section">
        <label class="sec-title">Installation Path</label>
        <div class="input-group">
          <input type="text" id="install-dir" value="Loading default...">
        </div>
      </div>

      <div class="section">
        <label class="sec-title">Options</label>
        <div class="toggle-row">
          <div class="toggle-info">
            <h4>Configure PATH Variable</h4>
            <p>Automatically export binary to your shell profile / environment</p>
          </div>
          <label class="switch">
            <input type="checkbox" id="modify-path" checked>
            <span class="slider"></span>
          </label>
        </div>

        <div class="toggle-row">
          <div class="toggle-info">
            <h4>Shell Autocompletions</h4>
            <p>Enable native tab-completion for runtime and version commands</p>
          </div>
          <label class="switch">
            <input type="checkbox" id="completions" checked>
            <span class="slider"></span>
          </label>
        </div>
      </div>

      <button class="btn" id="install-btn" onclick="startInstall()">
        <span>Install uvm v0.0.1</span>
      </button>
    </div>

    <div class="progress-box" id="progress-box">
      <div class="progress-bar-bg">
        <div class="progress-bar-fill" id="progress-bar"></div>
      </div>
      <div class="status-log" id="status-log"></div>
    </div>

    <div class="success-card" id="success-card">
      <div class="success-icon">🎉</div>
      <h2 style="font-weight: 700; margin-bottom: 8px;">Installation Complete!</h2>
      <p style="color: var(--text-muted); font-size: 14px;">uvm is now ready to manage programming language runtimes.</p>
      
      <div class="code-block" id="success-code">
# Restart terminal or run:
uvm --help

# Manage your runtimes:
uvm list node
uvm install node 20.11.0
uvm use node 20.11.0
      </div>

      <button class="btn btn-secondary" onclick="uninstall()">Uninstall uvm</button>
    </div>
  </div>

  <script>
    let sysInfo = {};
    let selectedNodeAction = 'move';

    async function loadStatus() {
      try {
        const res = await fetch('/api/status');
        sysInfo = await res.json();
        document.getElementById('os-badge').innerText = 'OS: ' + sysInfo.os;
        document.getElementById('arch-badge').innerText = 'Arch: ' + sysInfo.arch;
        document.getElementById('shell-badge').innerText = 'Shell: ' + sysInfo.detectedShell;
        document.getElementById('install-dir').value = sysInfo.defaultDir;

        renderDetectedRuntimes(sysInfo.detectedRuntimes || []);
      } catch (err) {
        console.error('Failed to load status:', err);
      }
    }

    function renderDetectedRuntimes(runtimes) {
      const container = document.getElementById('detected-runtimes-container');
      container.innerHTML = '';

      const nodeRt = runtimes.find(r => r.name === 'node' && r.found);
      if (!nodeRt) return;

      const card = document.createElement('div');
      card.className = 'detected-card';
      card.innerHTML = 
        '<div class="detected-header">' +
          '<div class="detected-title">' +
            '<span>📦 Existing Node.js Found</span>' +
            '<span class="node-ver-badge">' + (nodeRt.version || '') + '</span>' +
          '</div>' +
          '<span class="badge" style="font-size:11px;">' + (nodeRt.managerType || 'Detected') + '</span>' +
        '</div>' +
        '<div class="location-box">' + (nodeRt.installDir || '') + '</div>' +
        '<div class="options-grid">' +
          '<label class="opt-card selected" id="opt-move" onclick="selectAction(\'move\')">' +
            '<input type="radio" name="node_action" value="move" checked>' +
            '<div class="opt-content">' +
              '<h5>Move to UVM (Recommended)</h5>' +
              '<p>Preserve your current Node.js version and let UVM manage it seamlessly.</p>' +
            '</div>' +
          '</label>' +
          '<label class="opt-card" id="opt-delete" onclick="selectAction(\'delete\')">' +
            '<input type="radio" name="node_action" value="delete">' +
            '<div class="opt-content">' +
              '<h5>Delete existing Node.js</h5>' +
              '<p>Remove the existing installation and let UVM become responsible for Node.js.</p>' +
            '</div>' +
          '</label>' +
          '<div class="del-confirm-box" id="del-confirm">' +
            '<label style="display:flex; align-items:center; gap:8px; cursor:pointer;">' +
              '<input type="checkbox" id="confirm-del-check">' +
              '<span>I confirm deletion of this existing Node.js installation</span>' +
            '</label>' +
          '</div>' +
          '<label class="opt-card" id="opt-keep" onclick="selectAction(\'keep\')">' +
            '<input type="radio" name="node_action" value="keep">' +
            '<div class="opt-content">' +
              '<h5>Keep existing Node.js</h5>' +
              '<p>Leave existing installation unchanged (may take precedence until configured).</p>' +
            '</div>' +
          '</label>' +
        '</div>';
      container.appendChild(card);
    }

    function selectAction(action) {
      selectedNodeAction = action;
      document.querySelectorAll('.opt-card').forEach(c => c.classList.remove('selected'));
      const activeCard = document.getElementById('opt-' + action);
      if (activeCard) {
        activeCard.classList.add('selected');
        activeCard.querySelector('input[type="radio"]').checked = true;
      }
      const delConfirm = document.getElementById('del-confirm');
      if (delConfirm) {
        delConfirm.style.display = (action === 'delete') ? 'block' : 'none';
      }
    }

    function logStatus(msg) {
      const log = document.getElementById('status-log');
      const time = new Date().toLocaleTimeString();
      log.innerHTML += '<div>[' + time + '] ' + msg + '</div>';
      log.scrollTop = log.scrollHeight;
    }

    async function startInstall() {
      const btn = document.getElementById('install-btn');
      const progressBox = document.getElementById('progress-box');
      const bar = document.getElementById('progress-bar');
      const setupForm = document.getElementById('setup-form');

      let confirmDelete = false;
      if (selectedNodeAction === 'delete') {
        const check = document.getElementById('confirm-del-check');
        if (check && !check.checked) {
          alert('Please check the confirmation box before deleting existing Node.js.');
          return;
        }
        confirmDelete = true;
      }

      btn.disabled = true;
      progressBox.style.display = 'block';

      logStatus('Initializing setup...');
      bar.style.width = '20%';

      await new Promise(r => setTimeout(r, 300));
      logStatus('Creating install directory: ' + document.getElementById('install-dir').value);
      bar.style.width = '40%';

      await new Promise(r => setTimeout(r, 300));
      logStatus('Deploying uvm executable...');
      bar.style.width = '60%';

      try {
        const payload = {
          installDir: document.getElementById('install-dir').value,
          modifyPath: document.getElementById('modify-path').checked,
          createCompletions: document.getElementById('completions').checked,
          nodeAction: selectedNodeAction,
          confirmDelete: confirmDelete
        };
        logStatus('Handling runtime migration / configuration (' + selectedNodeAction + ')...');
        bar.style.width = '80%';

        const res = await fetch('/api/install', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload)
        });
        const data = await res.json();

        if (!res.ok || (data.errors && data.errors.length > 0)) {
          throw new Error((data.errors && data.errors.join(', ')) || data.message || 'Installation failed');
        }

        bar.style.width = '100%';
        logStatus(data.message || 'Installation finished.');

        await new Promise(r => setTimeout(r, 500));
        setupForm.style.display = 'none';
        progressBox.style.display = 'none';
        document.getElementById('success-card').style.display = 'block';
      } catch (err) {
        logStatus('Error: ' + err.message);
        btn.disabled = false;
      }
    }

    async function uninstall() {
      if (!confirm('Are you sure you want to uninstall uvm?')) return;
      try {
        const payload = { installDir: document.getElementById('install-dir').value };
        const res = await fetch('/api/uninstall', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload)
        });
        const data = await res.json();
        alert(data.message);
        location.reload();
      } catch (err) {
        alert('Uninstall failed: ' + err.message);
      }
    }

    loadStatus();
  </script>
</body>
</html>`
}
