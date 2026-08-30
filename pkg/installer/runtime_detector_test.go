package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectNodeNone(t *testing.T) {
	detector := NewRuntimeDetector(t.TempDir(), "linux")
	detector.LookPath = func(file string) (string, error) {
		return "", os.ErrNotExist
	}
	detector.Stat = func(name string) (os.FileInfo, error) {
		return nil, os.ErrNotExist
	}

	result := detector.DetectNode()
	if result.Found {
		t.Errorf("expected no node detected, got %+v", result)
	}

	all := detector.DetectAllRuntimes()
	if len(all) != 0 {
		t.Errorf("expected empty all runtimes, got %d", len(all))
	}
}

func TestDetectNodeWindowsProgramFiles(t *testing.T) {
	tmpDir := t.TempDir()
	progFiles := filepath.Join(tmpDir, "Program Files")
	nodeDir := filepath.Join(progFiles, "nodejs")
	_ = os.MkdirAll(nodeDir, 0755)
	nodeExe := filepath.Join(nodeDir, "node.exe")
	_ = os.WriteFile(nodeExe, []byte("fake"), 0755)

	detector := NewRuntimeDetector(tmpDir, "windows")
	detector.Env = func(key string) string {
		switch key {
		case "ProgramFiles":
			return progFiles
		case "APPDATA":
			return filepath.Join(tmpDir, "AppData", "Roaming")
		default:
			return ""
		}
	}
	detector.LookPath = func(file string) (string, error) {
		return "", os.ErrNotExist
	}
	detector.RunCmd = func(name string, args ...string) (string, error) {
		if name == nodeExe && len(args) > 0 && args[0] == "-v" {
			return "v22.18.0", nil
		}
		return "", fmt.Errorf("command failed")
	}

	result := detector.DetectNode()
	if !result.Found {
		t.Fatalf("expected node to be detected")
	}
	if result.Version != "v22.18.0" {
		t.Errorf("expected version v22.18.0, got %s", result.Version)
	}
	if result.ExecutablePath != nodeExe {
		t.Errorf("expected exe path %s, got %s", nodeExe, result.ExecutablePath)
	}
	if result.ManagerType != "Standard Windows Installer" {
		t.Errorf("expected Standard Windows Installer, got %s", result.ManagerType)
	}
}

func TestDetectNodeWindowsNVM(t *testing.T) {
	tmpDir := t.TempDir()
	nvmSymlink := filepath.Join(tmpDir, "nvm-nodejs")
	_ = os.MkdirAll(nvmSymlink, 0755)
	nodeExe := filepath.Join(nvmSymlink, "node.exe")
	_ = os.WriteFile(nodeExe, []byte("fake"), 0755)

	detector := NewRuntimeDetector(tmpDir, "windows")
	detector.Env = func(key string) string {
		switch key {
		case "NVM_SYMLINK":
			return nvmSymlink
		case "NVM_HOME":
			return filepath.Join(tmpDir, "nvm-home")
		default:
			return ""
		}
	}
	detector.RunCmd = func(name string, args ...string) (string, error) {
		return "v20.11.0", nil
	}

	result := detector.DetectNode()
	if !result.Found || result.ManagerType != "NVM for Windows" {
		t.Errorf("expected NVM for Windows detection, got: %+v", result)
	}
}

func TestDetectNodeMacOSHomebrew(t *testing.T) {
	tmpDir := t.TempDir()
	brewBin := filepath.Join(tmpDir, "opt", "homebrew", "bin")
	_ = os.MkdirAll(brewBin, 0755)
	nodeExe := filepath.Join(brewBin, "node")
	_ = os.WriteFile(nodeExe, []byte("fake"), 0755)

	detector := NewRuntimeDetector(tmpDir, "darwin")
	detector.LookPath = func(file string) (string, error) {
		return nodeExe, nil
	}
	detector.RunCmd = func(name string, args ...string) (string, error) {
		return "v20.10.0", nil
	}

	result := detector.DetectNode()
	if !result.Found || result.Version != "v20.10.0" {
		t.Fatalf("expected node detected, got %+v", result)
	}
	if result.ManagerType != "Homebrew" {
		t.Errorf("expected Homebrew manager type, got %s", result.ManagerType)
	}

	all := detector.DetectAllRuntimes()
	if len(all) != 1 {
		t.Errorf("expected 1 runtime in all, got %d", len(all))
	}
}

func TestDetectNodeIgnoresUVM(t *testing.T) {
	tmpDir := t.TempDir()
	uvmBin := filepath.Join(tmpDir, ".uvm", "bin")
	_ = os.MkdirAll(uvmBin, 0755)
	nodeExe := filepath.Join(uvmBin, "node")
	_ = os.WriteFile(nodeExe, []byte("fake"), 0755)

	detector := NewRuntimeDetector(tmpDir, "linux")
	detector.LookPath = func(file string) (string, error) {
		return nodeExe, nil
	}

	result := detector.DetectNode()
	if result.Found {
		t.Errorf("expected detector to ignore node inside .uvm, but found %+v", result)
	}
}
