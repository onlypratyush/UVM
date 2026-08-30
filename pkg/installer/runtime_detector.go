package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// DetectedRuntime represents details of an existing language runtime found on the system.
type DetectedRuntime struct {
	Name           string   `json:"name"`           // e.g. "node"
	Found          bool     `json:"found"`          // true if detected
	Version        string   `json:"version"`        // e.g. "v22.18.0"
	ExecutablePath string   `json:"executablePath"` // e.g. "C:\Program Files\nodejs\node.exe" or "/usr/local/bin/node"
	InstallDir     string   `json:"installDir"`     // e.g. "C:\Program Files\nodejs"
	NPMPath        string   `json:"npmPath"`        // e.g. "%APPDATA%\npm"
	ManagerType    string   `json:"managerType"`    // "System", "NVM for Windows", "NVM", "FNM", "Homebrew", "Volta", "Custom"
	PathEntries    []string `json:"pathEntries"`    // PATH entries belonging to this runtime
	Details        string   `json:"details"`        // Human-readable summary
}

// RuntimeDetector provides detection logic for existing runtimes.
type RuntimeDetector struct {
	GOOS      string
	Env       func(key string) string
	LookPath  func(file string) (string, error)
	RunCmd    func(name string, args ...string) (string, error)
	Stat      func(name string) (os.FileInfo, error)
	Readlink  func(name string) (string, error)
	UserHome  string
}

// NewRuntimeDetector creates a detector with system defaults.
func NewRuntimeDetector(customHome string, goos string) *RuntimeDetector {
	if goos == "" {
		goos = runtime.GOOS
	}
	home := customHome
	if home == "" {
		if h, err := os.UserHomeDir(); err == nil {
			home = h
		} else if h := os.Getenv("HOME"); h != "" {
			home = h
		} else {
			home = os.Getenv("USERPROFILE")
		}
	}

	return &RuntimeDetector{
		GOOS:     goos,
		Env:      os.Getenv,
		LookPath: exec.LookPath,
		RunCmd: func(name string, args ...string) (string, error) {
			cmd := exec.Command(name, args...)
			out, err := cmd.Output()
			return strings.TrimSpace(string(out)), err
		},
		Stat:     os.Stat,
		Readlink: os.Readlink,
		UserHome: home,
	}
}

// DetectNode detects an existing Node.js installation across platforms.
func (d *RuntimeDetector) DetectNode() DetectedRuntime {
	result := DetectedRuntime{
		Name:  "node",
		Found: false,
	}

	var candidatePaths []string
	var pathEntries []string

	// 1. Check PATH lookup via LookPath / where.exe / which
	if p, err := d.LookPath("node"); err == nil && p != "" {
		candidatePaths = append(candidatePaths, p)
	}

	// 2. Check platform-specific well-known directories
	if d.GOOS == "windows" {
		progFiles := d.Env("ProgramFiles")
		if progFiles == "" {
			progFiles = `C:\Program Files`
		}
		progFilesX86 := d.Env("ProgramFiles(x86)")
		if progFilesX86 == "" {
			progFilesX86 = `C:\Program Files (x86)`
		}
		appData := d.Env("APPDATA")
		if appData == "" && d.UserHome != "" {
			appData = filepath.Join(d.UserHome, "AppData", "Roaming")
		}
		localAppData := d.Env("LOCALAPPDATA")
		if localAppData == "" && d.UserHome != "" {
			localAppData = filepath.Join(d.UserHome, "AppData", "Local")
		}

		// Known Node.js locations on Windows
		winCandidates := []string{
			filepath.Join(progFiles, "nodejs", "node.exe"),
			filepath.Join(progFilesX86, "nodejs", "node.exe"),
			filepath.Join(appData, "npm", "node.exe"),
			filepath.Join(localAppData, "Programs", "nodejs", "node.exe"),
		}

		// NVM for Windows locations
		nvmHome := d.Env("NVM_HOME")
		nvmSymlink := d.Env("NVM_SYMLINK")
		if nvmSymlink != "" {
			winCandidates = append([]string{filepath.Join(nvmSymlink, "node.exe")}, winCandidates...)
		}
		if nvmHome != "" {
			winCandidates = append(winCandidates, filepath.Join(nvmHome, "nodejs", "node.exe"))
		}

		candidatePaths = append(candidatePaths, winCandidates...)
	} else {
		// macOS & Linux candidates
		unixCandidates := []string{
			"/opt/homebrew/bin/node",
			"/usr/local/bin/node",
			"/usr/bin/node",
			filepath.Join(d.UserHome, ".nvm", "current", "bin", "node"),
			filepath.Join(d.UserHome, ".local", "share", "fnm", "current", "bin", "node"),
			filepath.Join(d.UserHome, ".n", "bin", "node"),
			filepath.Join(d.UserHome, ".volta", "bin", "node"),
			filepath.Join(d.UserHome, ".asdf", "shims", "node"),
		}
		candidatePaths = append(candidatePaths, unixCandidates...)
	}

	// 3. Evaluate candidate paths to find the first working node executable
	var foundExe string
	var foundVersion string

	for _, cand := range candidatePaths {
		if cand == "" {
			continue
		}

		// Ignore if candidate is inside UVM's own directory
		if strings.Contains(cand, filepath.Join(".uvm", "bin")) ||
			strings.Contains(cand, filepath.Join(".uvm", "versions")) {
			continue
		}

		if _, err := d.Stat(cand); err == nil {
			// Probe version
			ver, err := d.RunCmd(cand, "-v")
			if err == nil && ver != "" && strings.HasPrefix(ver, "v") {
				foundExe = cand
				foundVersion = ver
				break
			}
		}
	}

	if foundExe == "" {
		return result
	}

	result.Found = true
	result.Version = foundVersion
	result.ExecutablePath = foundExe

	// Determine install directory and manager type
	dir := filepath.Dir(foundExe)
	if d.GOOS != "windows" && filepath.Base(dir) == "bin" {
		result.InstallDir = filepath.Dir(dir)
	} else {
		result.InstallDir = dir
	}

	// Identify manager type
	lowerPath := strings.ToLower(foundExe)
	if strings.Contains(lowerPath, "nvm") || d.Env("NVM_HOME") != "" || d.Env("NVM_SYMLINK") != "" {
		result.ManagerType = "NVM for Windows"
		if d.GOOS != "windows" {
			result.ManagerType = "NVM"
		}
	} else if strings.Contains(lowerPath, "fnm") {
		result.ManagerType = "FNM"
	} else if strings.Contains(lowerPath, "homebrew") || strings.Contains(lowerPath, "cellar") {
		result.ManagerType = "Homebrew"
	} else if strings.Contains(lowerPath, "volta") {
		result.ManagerType = "Volta"
	} else if strings.Contains(lowerPath, "program files") {
		result.ManagerType = "Standard Windows Installer"
	} else {
		result.ManagerType = "System / Standalone"
	}

	// Identify associated PATH entries to clean if moved/deleted
	pathEntries = append(pathEntries, dir)
	if d.GOOS == "windows" {
		appData := d.Env("APPDATA")
		if appData == "" && d.UserHome != "" {
			appData = filepath.Join(d.UserHome, "AppData", "Roaming")
		}
		npmDir := filepath.Join(appData, "npm")
		if _, err := d.Stat(npmDir); err == nil {
			result.NPMPath = npmDir
			pathEntries = append(pathEntries, npmDir)
		}
		if nvmSymlink := d.Env("NVM_SYMLINK"); nvmSymlink != "" {
			pathEntries = append(pathEntries, nvmSymlink)
		}
	} else {
		if dir != "/usr/bin" && dir != "/bin" {
			pathEntries = append(pathEntries, dir)
		}
	}

	result.PathEntries = pathEntries
	result.Details = fmt.Sprintf("Node.js %s found at %s (Managed by %s)", result.Version, result.InstallDir, result.ManagerType)

	return result
}

// DetectAllRuntimes scans and detects all supported runtimes.
func (d *RuntimeDetector) DetectAllRuntimes() []DetectedRuntime {
	var runtimes []DetectedRuntime

	nodeRt := d.DetectNode()
	if nodeRt.Found {
		runtimes = append(runtimes, nodeRt)
	}

	return runtimes
}
