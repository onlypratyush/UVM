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
		if file == "node" {
			return nodeExe, nil
		}
		return "", os.ErrNotExist
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

func TestDetectGo(t *testing.T) {
	tmpDir := t.TempDir()
	goBin := filepath.Join(tmpDir, "usr", "local", "go", "bin")
	_ = os.MkdirAll(goBin, 0755)
	goExe := filepath.Join(goBin, "go")
	_ = os.WriteFile(goExe, []byte("fake"), 0755)

	detector := NewRuntimeDetector(tmpDir, "darwin")
	detector.LookPath = func(file string) (string, error) {
		if file == "go" {
			return goExe, nil
		}
		return "", os.ErrNotExist
	}
	detector.RunCmd = func(name string, args ...string) (string, error) {
		if name == goExe && len(args) > 0 && args[0] == "version" {
			return "go version go1.22.0 darwin/arm64", nil
		}
		return "", fmt.Errorf("command failed")
	}

	res := detector.DetectGo()
	if !res.Found || res.Version != "go1.22.0" {
		t.Fatalf("expected Go detected go1.22.0, got: %+v", res)
	}

	// Test windows Go fallback
	winDetector := NewRuntimeDetector(tmpDir, "windows")
	progFiles := filepath.Join(tmpDir, "Program Files")
	winGoExe := filepath.Join(progFiles, "Go", "bin", "go.exe")
	_ = os.MkdirAll(filepath.Dir(winGoExe), 0755)
	_ = os.WriteFile(winGoExe, []byte("fake"), 0755)
	winDetector.Env = func(key string) string {
		if key == "ProgramFiles" {
			return progFiles
		}
		return ""
	}
	winDetector.LookPath = func(file string) (string, error) {
		return "", os.ErrNotExist
	}
	winDetector.RunCmd = func(name string, args ...string) (string, error) {
		return "go version go1.23.0 windows/amd64", nil
	}

	winRes := winDetector.DetectGo()
	if !winRes.Found || winRes.Version != "go1.23.0" {
		t.Fatalf("expected Windows Go detected, got: %+v", winRes)
	}
}

func TestDetectPython(t *testing.T) {
	tmpDir := t.TempDir()
	pyBin := filepath.Join(tmpDir, "usr", "local", "bin")
	_ = os.MkdirAll(pyBin, 0755)
	pyExe := filepath.Join(pyBin, "python3")
	_ = os.WriteFile(pyExe, []byte("fake"), 0755)

	detector := NewRuntimeDetector(tmpDir, "darwin")
	detector.LookPath = func(file string) (string, error) {
		if file == "python3" {
			return pyExe, nil
		}
		return "", os.ErrNotExist
	}
	detector.RunCmd = func(name string, args ...string) (string, error) {
		if name == pyExe && len(args) > 0 && args[0] == "--version" {
			return "Python 3.12.2", nil
		}
		return "", fmt.Errorf("command failed")
	}

	res := detector.DetectPython()
	if !res.Found || res.Version != "3.12.2" {
		t.Fatalf("expected Python detected 3.12.2, got: %+v", res)
	}

	// Test windows Python
	winDetector := NewRuntimeDetector(tmpDir, "windows")
	localAppData := filepath.Join(tmpDir, "AppData", "Local")
	winPyExe := filepath.Join(localAppData, "Programs", "Python", "Python312", "python.exe")
	_ = os.MkdirAll(filepath.Dir(winPyExe), 0755)
	_ = os.WriteFile(winPyExe, []byte("fake"), 0755)
	winDetector.Env = func(key string) string {
		if key == "LOCALAPPDATA" {
			return localAppData
		}
		return ""
	}
	winDetector.LookPath = func(file string) (string, error) {
		return "", os.ErrNotExist
	}
	winDetector.RunCmd = func(name string, args ...string) (string, error) {
		return "Python 3.12.0", nil
	}

	winRes := winDetector.DetectPython()
	if !winRes.Found || winRes.Version != "3.12.0" {
		t.Fatalf("expected Windows Python detected, got: %+v", winRes)
	}

	all := detector.DetectAllRuntimes()
	if len(all) == 0 {
		t.Errorf("expected detected runtimes in all, got 0")
	}
}

func TestDetectPHP(t *testing.T) {
	tmpDir := t.TempDir()
	phpBin := filepath.Join(tmpDir, "opt", "homebrew", "bin")
	_ = os.MkdirAll(phpBin, 0755)
	phpExe := filepath.Join(phpBin, "php")
	_ = os.WriteFile(phpExe, []byte("fake"), 0755)

	detector := NewRuntimeDetector(tmpDir, "darwin")
	detector.LookPath = func(file string) (string, error) {
		if file == "php" {
			return phpExe, nil
		}
		return "", os.ErrNotExist
	}
	detector.RunCmd = func(name string, args ...string) (string, error) {
		if name == phpExe && len(args) > 0 && args[0] == "-v" {
			return "PHP 8.3.17 (cli) (built: Feb 13 2025)", nil
		}
		return "", fmt.Errorf("command failed")
	}

	res := detector.DetectPHP()
	if !res.Found || res.Version != "8.3.17" {
		t.Fatalf("expected PHP detected 8.3.17, got: %+v", res)
	}

	// Test windows PHP
	winDetector := NewRuntimeDetector(tmpDir, "windows")
	progFiles := filepath.Join(tmpDir, "Program Files")
	winPhpExe := filepath.Join(progFiles, "PHP", "php.exe")
	_ = os.MkdirAll(filepath.Dir(winPhpExe), 0755)
	_ = os.WriteFile(winPhpExe, []byte("fake"), 0755)
	winDetector.Env = func(key string) string {
		if key == "ProgramFiles" {
			return progFiles
		}
		return ""
	}
	winDetector.LookPath = func(file string) (string, error) {
		return "", os.ErrNotExist
	}
	winDetector.RunCmd = func(name string, args ...string) (string, error) {
		return "PHP 8.4.4 (cli) (built: Feb 13 2025)", nil
	}

	winRes := winDetector.DetectPHP()
	if !winRes.Found || winRes.Version != "8.4.4" {
		t.Fatalf("expected Windows PHP detected, got: %+v", winRes)
	}
}

func TestDetectJava(t *testing.T) {
	tmpDir := t.TempDir()
	javaBin := filepath.Join(tmpDir, "usr", "bin")
	_ = os.MkdirAll(javaBin, 0755)
	javaExe := filepath.Join(javaBin, "java")
	_ = os.WriteFile(javaExe, []byte("fake"), 0755)

	detector := NewRuntimeDetector(tmpDir, "linux")
	detector.LookPath = func(file string) (string, error) {
		if file == "java" {
			return javaExe, nil
		}
		return "", os.ErrNotExist
	}
	detector.RunCmd = func(name string, args ...string) (string, error) {
		if name == javaExe && len(args) > 0 && (args[0] == "-version" || args[0] == "--version") {
			return "openjdk version \"21.0.6\" 2025-01-21", nil
		}
		return "", fmt.Errorf("command failed")
	}

	res := detector.DetectJava()
	if !res.Found || res.Version != "21.0.6" {
		t.Fatalf("expected Java detected 21.0.6, got: %+v", res)
	}

	// Test JAVA_HOME detection
	javaHomeDir := filepath.Join(tmpDir, "opt", "jdk-17")
	javaHomeBin := filepath.Join(javaHomeDir, "bin")
	_ = os.MkdirAll(javaHomeBin, 0755)
	javaHomeExe := filepath.Join(javaHomeBin, "java")
	_ = os.WriteFile(javaHomeExe, []byte("fake"), 0755)

	jhDetector := NewRuntimeDetector(tmpDir, "darwin")
	jhDetector.Env = func(key string) string {
		if key == "JAVA_HOME" {
			return javaHomeDir
		}
		return ""
	}
	jhDetector.LookPath = func(file string) (string, error) {
		return "", os.ErrNotExist
	}
	jhDetector.RunCmd = func(name string, args ...string) (string, error) {
		return "openjdk version \"17.0.14\" 2025-01-21", nil
	}

	jhRes := jhDetector.DetectJava()
	if !jhRes.Found || jhRes.Version != "17.0.14" {
		t.Fatalf("expected Java via JAVA_HOME detected, got: %+v", jhRes)
	}
}

func TestDetectBun(t *testing.T) {
	tmpDir := t.TempDir()
	bunBin := filepath.Join(tmpDir, "opt", "homebrew", "bin")
	_ = os.MkdirAll(bunBin, 0755)
	bunExe := filepath.Join(bunBin, "bun")
	_ = os.WriteFile(bunExe, []byte("fake"), 0755)

	detector := NewRuntimeDetector(tmpDir, "darwin")
	detector.LookPath = func(file string) (string, error) {
		if file == "bun" {
			return bunExe, nil
		}
		return "", os.ErrNotExist
	}
	detector.RunCmd = func(name string, args ...string) (string, error) {
		if name == bunExe && len(args) > 0 && args[0] == "-v" {
			return "1.2.4\n", nil
		}
		return "", fmt.Errorf("command failed")
	}

	res := detector.DetectBun()
	if !res.Found || res.Version != "1.2.4" {
		t.Fatalf("expected Bun detected 1.2.4, got: %+v", res)
	}

	// Test windows Bun installer in ~/.bun/bin/bun.exe
	winDetector := NewRuntimeDetector(tmpDir, "windows")
	winBunExe := filepath.Join(tmpDir, ".bun", "bin", "bun.exe")
	_ = os.MkdirAll(filepath.Dir(winBunExe), 0755)
	_ = os.WriteFile(winBunExe, []byte("fake"), 0755)

	winDetector.LookPath = func(file string) (string, error) {
		return "", os.ErrNotExist
	}
	winDetector.RunCmd = func(name string, args ...string) (string, error) {
		return "1.2.0", nil
	}

	winRes := winDetector.DetectBun()
	if !winRes.Found || winRes.Version != "1.2.0" || winRes.ManagerType != "Bun Installer" {
		t.Fatalf("expected Windows Bun detected in ~/.bun, got: %+v", winRes)
	}
}

