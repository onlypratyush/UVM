package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewRootCmd(t *testing.T) {
	cmd := NewRootCmd()
	if cmd.Use != "uvm" {
		t.Errorf("expected Use 'uvm', got '%s'", cmd.Use)
	}
	if cmd.Version != version {
		t.Errorf("expected Version '%s', got '%s'", version, cmd.Version)
	}
}

func TestExecuteRootHelp(t *testing.T) {
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := Execute([]string{"--help"}, outBuf, errBuf)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	out := outBuf.String()
	if !strings.Contains(out, "Universal Version Manager") {
		t.Errorf("expected help output to contain description, got: %s", out)
	}
}

func TestExecuteVersion(t *testing.T) {
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := Execute([]string{"--version"}, outBuf, errBuf)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	out := outBuf.String()
	if !strings.Contains(out, version) {
		t.Errorf("expected output to contain version '%s', got: %s", version, out)
	}
}

func TestInstallCmd(t *testing.T) {
	tmpHome := t.TempDir()
	oldHome := os.Getenv("HOME")
	oldUserProfile := os.Getenv("USERPROFILE")
	defer func() {
		os.Setenv("HOME", oldHome)
		os.Setenv("USERPROFILE", oldUserProfile)
	}()
	os.Setenv("HOME", tmpHome)
	os.Setenv("USERPROFILE", tmpHome)

	// Mock already installed node version
	nodeDir := filepath.Join(tmpHome, ".uvm", "versions", "node", "v20.11.0")
	_ = os.MkdirAll(nodeDir, 0755)

	// Mock already installed go version
	goDir := filepath.Join(tmpHome, ".uvm", "versions", "go", "go1.22.0")
	_ = os.MkdirAll(goDir, 0755)

	// Mock already installed python version
	pyDir := filepath.Join(tmpHome, ".uvm", "versions", "python", "3.12.2")
	_ = os.MkdirAll(pyDir, 0755)

	// Mock already installed php version
	phpDir := filepath.Join(tmpHome, ".uvm", "versions", "php", "8.3.17")
	_ = os.MkdirAll(phpDir, 0755)

	// Mock already installed java version
	javaDir := filepath.Join(tmpHome, ".uvm", "versions", "java", "21.0.6")
	_ = os.MkdirAll(javaDir, 0755)

	// Mock already installed bun version
	bunDir := filepath.Join(tmpHome, ".uvm", "versions", "bun", "1.2.4")
	_ = os.MkdirAll(bunDir, 0755)

	tests := []struct {
		name        string
		args        []string
		expectedOut string
		expectError bool
	}{
		{
			name:        "install already installed node",
			args:        []string{"install", "node", "20.11.0"},
			expectedOut: "Node.js v20.11.0 is already installed",
			expectError: false,
		},
		{
			name:        "install nodejs alias",
			args:        []string{"install", "nodejs", "20.11.0"},
			expectedOut: "Node.js v20.11.0 is already installed",
			expectError: false,
		},
		{
			name:        "install already installed go",
			args:        []string{"install", "go", "1.22.0"},
			expectedOut: "Go go1.22.0 is already installed",
			expectError: false,
		},
		{
			name:        "install golang alias",
			args:        []string{"install", "golang", "1.22.0"},
			expectedOut: "Go go1.22.0 is already installed",
			expectError: false,
		},
		{
			name:        "install already installed python",
			args:        []string{"install", "python", "3.12.2"},
			expectedOut: "Python 3.12.2 is already installed",
			expectError: false,
		},
		{
			name:        "install py alias",
			args:        []string{"install", "py", "3.12.2"},
			expectedOut: "Python 3.12.2 is already installed",
			expectError: false,
		},
		{
			name:        "install python3 alias",
			args:        []string{"install", "python3", "3.12.2"},
			expectedOut: "Python 3.12.2 is already installed",
			expectError: false,
		},
		{
			name:        "install already installed php",
			args:        []string{"install", "php", "8.3.17"},
			expectedOut: "PHP 8.3.17 is already installed",
			expectError: false,
		},
		{
			name:        "install already installed java",
			args:        []string{"install", "java", "21.0.6"},
			expectedOut: "Java 21.0.6 is already installed",
			expectError: false,
		},
		{
			name:        "install jdk alias",
			args:        []string{"install", "jdk", "21.0.6"},
			expectedOut: "Java 21.0.6 is already installed",
			expectError: false,
		},
		{
			name:        "install openjdk alias",
			args:        []string{"install", "openjdk", "21.0.6"},
			expectedOut: "Java 21.0.6 is already installed",
			expectError: false,
		},
		{
			name:        "install already installed bun",
			args:        []string{"install", "bun", "1.2.4"},
			expectedOut: "Bun 1.2.4 is already installed",
			expectError: false,
		},
		{
			name:        "install unsupported runtime",
			args:        []string{"install", "rust", "1.75.0"},
			expectError: true,
		},
		{
			name:        "install missing args",
			args:        []string{"install", "node"},
			expectError: true,
		},
		{
			name:        "install too many args",
			args:        []string{"install", "node", "20.11.0", "extra"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outBuf := new(bytes.Buffer)
			errBuf := new(bytes.Buffer)

			err := Execute(tt.args, outBuf, errBuf)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for args %v, got none", tt.args)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !strings.Contains(outBuf.String(), tt.expectedOut) {
					t.Errorf("expected output to contain %q, got %q", tt.expectedOut, outBuf.String())
				}
			}
		})
	}
}

func TestUseCmd(t *testing.T) {
	tmpHome := t.TempDir()
	oldHome := os.Getenv("HOME")
	oldUserProfile := os.Getenv("USERPROFILE")
	defer func() {
		os.Setenv("HOME", oldHome)
		os.Setenv("USERPROFILE", oldUserProfile)
	}()
	os.Setenv("HOME", tmpHome)
	os.Setenv("USERPROFILE", tmpHome)

	// Create mock node version
	nodeDir := filepath.Join(tmpHome, ".uvm", "versions", "node", "v20.11.0", "bin")
	_ = os.MkdirAll(nodeDir, 0755)

	// Create mock go version
	goDir := filepath.Join(tmpHome, ".uvm", "versions", "go", "go1.22.0", "bin")
	_ = os.MkdirAll(goDir, 0755)

	// Create mock python version
	pyDir := filepath.Join(tmpHome, ".uvm", "versions", "python", "3.12.2", "bin")
	_ = os.MkdirAll(pyDir, 0755)

	// Create mock php version
	phpDir := filepath.Join(tmpHome, ".uvm", "versions", "php", "8.3.17", "bin")
	_ = os.MkdirAll(phpDir, 0755)

	// Create mock java version
	javaDir := filepath.Join(tmpHome, ".uvm", "versions", "java", "21.0.6", "bin")
	_ = os.MkdirAll(javaDir, 0755)

	// Create mock bun version
	bunDir := filepath.Join(tmpHome, ".uvm", "versions", "bun", "1.2.4")
	_ = os.MkdirAll(bunDir, 0755)
	_ = os.WriteFile(filepath.Join(bunDir, "bun"), []byte("bin"), 0755)

	tests := []struct {
		name        string
		args        []string
		expectedOut string
		expectError bool
	}{
		{
			name:        "use node installed",
			args:        []string{"use", "node", "20.11.0"},
			expectedOut: "Now using Node.js v20.11.0\n",
			expectError: false,
		},
		{
			name:        "use nodejs alias",
			args:        []string{"use", "nodejs", "20.11.0"},
			expectedOut: "Now using Node.js v20.11.0\n",
			expectError: false,
		},
		{
			name:        "use node uninstalled",
			args:        []string{"use", "node", "18.0.0"},
			expectError: true,
		},
		{
			name:        "use go installed",
			args:        []string{"use", "go", "1.22.0"},
			expectedOut: "Now using Go go1.22.0\n",
			expectError: false,
		},
		{
			name:        "use golang alias",
			args:        []string{"use", "golang", "1.22.0"},
			expectedOut: "Now using Go go1.22.0\n",
			expectError: false,
		},
		{
			name:        "use go uninstalled",
			args:        []string{"use", "go", "1.18.0"},
			expectError: true,
		},
		{
			name:        "use python installed",
			args:        []string{"use", "python", "3.12.2"},
			expectedOut: "Now using Python 3.12.2\n",
			expectError: false,
		},
		{
			name:        "use py alias",
			args:        []string{"use", "py", "3.12.2"},
			expectedOut: "Now using Python 3.12.2\n",
			expectError: false,
		},
		{
			name:        "use python3 alias",
			args:        []string{"use", "python3", "3.12.2"},
			expectedOut: "Now using Python 3.12.2\n",
			expectError: false,
		},
		{
			name:        "use python uninstalled",
			args:        []string{"use", "python", "3.9.0"},
			expectError: true,
		},
		{
			name:        "use php installed",
			args:        []string{"use", "php", "8.3.17"},
			expectedOut: "Now using PHP 8.3.17\n",
			expectError: false,
		},
		{
			name:        "use php uninstalled",
			args:        []string{"use", "php", "7.4.0"},
			expectError: true,
		},
		{
			name:        "use java installed",
			args:        []string{"use", "java", "21.0.6"},
			expectedOut: "Now using Java 21.0.6\n",
			expectError: false,
		},
		{
			name:        "use jdk alias",
			args:        []string{"use", "jdk", "21.0.6"},
			expectedOut: "Now using Java 21.0.6\n",
			expectError: false,
		},
		{
			name:        "use openjdk alias",
			args:        []string{"use", "openjdk", "21.0.6"},
			expectedOut: "Now using Java 21.0.6\n",
			expectError: false,
		},
		{
			name:        "use java uninstalled",
			args:        []string{"use", "java", "11.0.0"},
			expectError: true,
		},
		{
			name:        "use bun installed",
			args:        []string{"use", "bun", "1.2.4"},
			expectedOut: "Now using Bun 1.2.4\n",
			expectError: false,
		},
		{
			name:        "use bun uninstalled",
			args:        []string{"use", "bun", "0.8.0"},
			expectError: true,
		},
		{
			name:        "use unsupported runtime",
			args:        []string{"use", "ruby", "3.2.0"},
			expectError: true,
		},
		{
			name:        "use missing args",
			args:        []string{"use", "node"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outBuf := new(bytes.Buffer)
			errBuf := new(bytes.Buffer)

			err := Execute(tt.args, outBuf, errBuf)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for args %v, got none", tt.args)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !strings.Contains(outBuf.String(), tt.expectedOut) {
					t.Errorf("expected output to contain %q, got %q", tt.expectedOut, outBuf.String())
				}
			}
		})
	}
}

func TestListCmd(t *testing.T) {
	tmpHome := t.TempDir()
	oldHome := os.Getenv("HOME")
	oldUserProfile := os.Getenv("USERPROFILE")
	defer func() {
		os.Setenv("HOME", oldHome)
		os.Setenv("USERPROFILE", oldUserProfile)
	}()
	os.Setenv("HOME", tmpHome)
	os.Setenv("USERPROFILE", tmpHome)

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	// List when no runtimes are installed
	_ = Execute([]string{"list"}, outBuf, errBuf)
	if !strings.Contains(outBuf.String(), "No runtime versions installed") {
		t.Errorf("unexpected output when empty: %s", outBuf.String())
	}

	// Empty list for individual runtimes
	outBuf.Reset()
	_ = Execute([]string{"list", "node"}, outBuf, errBuf)
	if !strings.Contains(outBuf.String(), "No installed Node.js versions found") {
		t.Errorf("unexpected output for empty list node: %s", outBuf.String())
	}

	outBuf.Reset()
	_ = Execute([]string{"list", "go"}, outBuf, errBuf)
	if !strings.Contains(outBuf.String(), "No installed Go versions found") {
		t.Errorf("unexpected output for empty list go: %s", outBuf.String())
	}

	outBuf.Reset()
	_ = Execute([]string{"list", "python"}, outBuf, errBuf)
	if !strings.Contains(outBuf.String(), "No installed Python versions found") {
		t.Errorf("unexpected output for empty list python: %s", outBuf.String())
	}

	outBuf.Reset()
	_ = Execute([]string{"list", "php"}, outBuf, errBuf)
	if !strings.Contains(outBuf.String(), "No installed PHP versions found") {
		t.Errorf("unexpected output for empty list php: %s", outBuf.String())
	}

	outBuf.Reset()
	_ = Execute([]string{"list", "java"}, outBuf, errBuf)
	if !strings.Contains(outBuf.String(), "No installed Java versions found") {
		t.Errorf("unexpected output for empty list java: %s", outBuf.String())
	}

	outBuf.Reset()
	_ = Execute([]string{"list", "bun"}, outBuf, errBuf)
	if !strings.Contains(outBuf.String(), "No installed Bun versions found") {
		t.Errorf("unexpected output for empty list bun: %s", outBuf.String())
	}

	// Create installed versions for all runtimes
	_ = os.MkdirAll(filepath.Join(tmpHome, ".uvm", "versions", "node", "v20.11.0"), 0755)
	_ = os.MkdirAll(filepath.Join(tmpHome, ".uvm", "versions", "go", "go1.22.0"), 0755)
	_ = os.MkdirAll(filepath.Join(tmpHome, ".uvm", "versions", "python", "3.12.2"), 0755)
	_ = os.MkdirAll(filepath.Join(tmpHome, ".uvm", "versions", "php", "8.3.17"), 0755)
	_ = os.MkdirAll(filepath.Join(tmpHome, ".uvm", "versions", "java", "21.0.6"), 0755)
	_ = os.MkdirAll(filepath.Join(tmpHome, ".uvm", "versions", "bun", "1.2.4"), 0755)

	// Activate versions
	currentParent := filepath.Join(tmpHome, ".uvm", "current")
	_ = os.MkdirAll(currentParent, 0755)
	_ = os.WriteFile(filepath.Join(currentParent, "node.version"), []byte("v20.11.0"), 0644)
	_ = os.WriteFile(filepath.Join(currentParent, "go.version"), []byte("go1.22.0"), 0644)
	_ = os.WriteFile(filepath.Join(currentParent, "python.version"), []byte("3.12.2"), 0644)
	_ = os.WriteFile(filepath.Join(currentParent, "php.version"), []byte("8.3.17"), 0644)
	_ = os.WriteFile(filepath.Join(currentParent, "java.version"), []byte("21.0.6"), 0644)
	_ = os.WriteFile(filepath.Join(currentParent, "bun.version"), []byte("1.2.4"), 0644)

	// List all
	outBuf.Reset()
	_ = Execute([]string{"list"}, outBuf, errBuf)
	out := outBuf.String()
	if !strings.Contains(out, "Installed Node.js versions:") ||
		!strings.Contains(out, "Installed Go versions:") ||
		!strings.Contains(out, "Installed Python versions:") ||
		!strings.Contains(out, "Installed PHP versions:") ||
		!strings.Contains(out, "Installed Java versions:") ||
		!strings.Contains(out, "Installed Bun versions:") {
		t.Errorf("expected all 6 runtimes in list all output, got: %s", out)
	}

	// List specific runtimes
	outBuf.Reset()
	_ = Execute([]string{"ls", "golang"}, outBuf, errBuf)
	if !strings.Contains(outBuf.String(), "* go1.22.0 (active)") {
		t.Errorf("unexpected go list output: %s", outBuf.String())
	}

	outBuf.Reset()
	_ = Execute([]string{"ls", "py"}, outBuf, errBuf)
	if !strings.Contains(outBuf.String(), "* 3.12.2 (active)") {
		t.Errorf("unexpected py list output: %s", outBuf.String())
	}

	outBuf.Reset()
	_ = Execute([]string{"ls", "php"}, outBuf, errBuf)
	if !strings.Contains(outBuf.String(), "* 8.3.17 (active)") {
		t.Errorf("unexpected php list output: %s", outBuf.String())
	}

	outBuf.Reset()
	_ = Execute([]string{"ls", "jdk"}, outBuf, errBuf)
	if !strings.Contains(outBuf.String(), "* 21.0.6 (active)") {
		t.Errorf("unexpected java list output: %s", outBuf.String())
	}

	outBuf.Reset()
	_ = Execute([]string{"ls", "bun"}, outBuf, errBuf)
	if !strings.Contains(outBuf.String(), "* 1.2.4 (active)") {
		t.Errorf("unexpected bun list output: %s", outBuf.String())
	}

	// Unsupported runtime
	outBuf.Reset()
	err := Execute([]string{"list", "invalid_rt"}, outBuf, errBuf)
	if err == nil {
		t.Errorf("expected error for unsupported runtime list")
	}
}

func TestRemoveCmd(t *testing.T) {
	tmpHome := t.TempDir()
	oldHome := os.Getenv("HOME")
	oldUserProfile := os.Getenv("USERPROFILE")
	defer func() {
		os.Setenv("HOME", oldHome)
		os.Setenv("USERPROFILE", oldUserProfile)
	}()
	os.Setenv("HOME", tmpHome)
	os.Setenv("USERPROFILE", tmpHome)

	_ = os.MkdirAll(filepath.Join(tmpHome, ".uvm", "versions", "node", "v20.11.0"), 0755)
	_ = os.MkdirAll(filepath.Join(tmpHome, ".uvm", "versions", "go", "go1.22.0"), 0755)
	_ = os.MkdirAll(filepath.Join(tmpHome, ".uvm", "versions", "python", "3.12.2"), 0755)
	_ = os.MkdirAll(filepath.Join(tmpHome, ".uvm", "versions", "php", "8.3.17"), 0755)
	_ = os.MkdirAll(filepath.Join(tmpHome, ".uvm", "versions", "java", "21.0.6"), 0755)
	_ = os.MkdirAll(filepath.Join(tmpHome, ".uvm", "versions", "bun", "1.2.4"), 0755)

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	// Remove node version
	err := Execute([]string{"remove", "node", "20.11.0"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "removed successfully") {
		t.Fatalf("remove node failed: %v, out: %s", err, outBuf.String())
	}

	// Remove go version
	outBuf.Reset()
	err = Execute([]string{"rm", "go", "1.22.0"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "removed successfully") {
		t.Fatalf("remove go failed: %v, out: %s", err, outBuf.String())
	}

	// Remove python version
	outBuf.Reset()
	err = Execute([]string{"uninstall", "python", "3.12.2"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "removed successfully") {
		t.Fatalf("remove python failed: %v, out: %s", err, outBuf.String())
	}

	// Remove php version
	outBuf.Reset()
	err = Execute([]string{"remove", "php", "8.3.17"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "removed successfully") {
		t.Fatalf("remove php failed: %v, out: %s", err, outBuf.String())
	}

	// Remove java version
	outBuf.Reset()
	err = Execute([]string{"rm", "java", "21.0.6"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "removed successfully") {
		t.Fatalf("remove java failed: %v, out: %s", err, outBuf.String())
	}

	// Remove bun version
	outBuf.Reset()
	err = Execute([]string{"rm", "bun", "1.2.4"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "removed successfully") {
		t.Fatalf("remove bun failed: %v, out: %s", err, outBuf.String())
	}

	// Remove unsupported runtime
	outBuf.Reset()
	err = Execute([]string{"rm", "invalid_rt", "1.0.0"}, outBuf, errBuf)
	if err == nil {
		t.Errorf("expected error removing unsupported runtime")
	}
}

func TestCurrentCmd(t *testing.T) {
	tmpHome := t.TempDir()
	oldHome := os.Getenv("HOME")
	oldUserProfile := os.Getenv("USERPROFILE")
	defer func() {
		os.Setenv("HOME", oldHome)
		os.Setenv("USERPROFILE", oldUserProfile)
	}()
	os.Setenv("HOME", tmpHome)
	os.Setenv("USERPROFILE", tmpHome)

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	// Current when none is active
	err := Execute([]string{"current"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "No active runtime versions set") {
		t.Errorf("expected no active runtime versions message, got: %s", outBuf.String())
	}

	// Set active versions
	currentParent := filepath.Join(tmpHome, ".uvm", "current")
	_ = os.MkdirAll(currentParent, 0755)
	_ = os.WriteFile(filepath.Join(currentParent, "node.version"), []byte("v20.11.0"), 0644)
	_ = os.WriteFile(filepath.Join(currentParent, "go.version"), []byte("go1.22.0"), 0644)
	_ = os.WriteFile(filepath.Join(currentParent, "python.version"), []byte("3.12.2"), 0644)
	_ = os.WriteFile(filepath.Join(currentParent, "php.version"), []byte("8.3.17"), 0644)
	_ = os.WriteFile(filepath.Join(currentParent, "java.version"), []byte("21.0.6"), 0644)
	_ = os.WriteFile(filepath.Join(currentParent, "bun.version"), []byte("1.2.4"), 0644)

	outBuf.Reset()
	err = Execute([]string{"current"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "Node.js: v20.11.0") ||
		!strings.Contains(outBuf.String(), "Go:      go1.22.0") ||
		!strings.Contains(outBuf.String(), "Python:  3.12.2") ||
		!strings.Contains(outBuf.String(), "PHP:     8.3.17") ||
		!strings.Contains(outBuf.String(), "Java:    21.0.6") ||
		!strings.Contains(outBuf.String(), "Bun:     1.2.4") {
		t.Errorf("expected all 6 runtimes in current output, got: %s", outBuf.String())
	}

	outBuf.Reset()
	err = Execute([]string{"current", "node"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "v20.11.0") {
		t.Errorf("expected current node version, got: %s", outBuf.String())
	}

	outBuf.Reset()
	err = Execute([]string{"current", "go"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "go1.22.0") {
		t.Errorf("expected current go version, got: %s", outBuf.String())
	}

	outBuf.Reset()
	err = Execute([]string{"current", "python"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "3.12.2") {
		t.Errorf("expected current python version, got: %s", outBuf.String())
	}

	outBuf.Reset()
	err = Execute([]string{"current", "php"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "8.3.17") {
		t.Errorf("expected current php version, got: %s", outBuf.String())
	}

	outBuf.Reset()
	err = Execute([]string{"current", "java"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "21.0.6") {
		t.Errorf("expected current java version, got: %s", outBuf.String())
	}

	outBuf.Reset()
	err = Execute([]string{"current", "bun"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "1.2.4") {
		t.Errorf("expected current bun version, got: %s", outBuf.String())
	}

	// Inactive individual runtime
	_ = os.Remove(filepath.Join(currentParent, "bun.version"))
	outBuf.Reset()
	err = Execute([]string{"current", "bun"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "No active Bun version set") {
		t.Errorf("expected no active bun version, got: %s", outBuf.String())
	}

	// Unsupported runtime
	err = Execute([]string{"current", "unsupported_rt"}, outBuf, errBuf)
	if err == nil {
		t.Errorf("expected error for unsupported runtime current")
	}
}

func TestExecuteWithNilStreams(t *testing.T) {
	err := Execute([]string{"--version"}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error with nil streams: %v", err)
	}
}

func TestRunFunction(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tmpHome := t.TempDir()
	oldHome := os.Getenv("HOME")
	oldUserProfile := os.Getenv("USERPROFILE")
	defer func() {
		os.Setenv("HOME", oldHome)
		os.Setenv("USERPROFILE", oldUserProfile)
	}()
	os.Setenv("HOME", tmpHome)
	os.Setenv("USERPROFILE", tmpHome)

	os.Args = []string{"uvm", "--version"}
	if err := run(); err != nil {
		t.Fatalf("run() returned error: %v", err)
	}
}

func TestMainExecutionSuccess(t *testing.T) {
	if os.Getenv("BE_UVM_MAIN_SUCCESS") == "1" {
		os.Args = []string{"uvm", "--version"}
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestMainExecutionSuccess")
	cmd.Env = append(os.Environ(), "BE_UVM_MAIN_SUCCESS=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("process exited with err: %v, output: %s", err, string(output))
	}
	if !strings.Contains(string(output), version) {
		t.Errorf("unexpected output: %s", string(output))
	}
}

func TestMainExecutionFailure(t *testing.T) {
	if os.Getenv("BE_UVM_MAIN_FAILURE") == "1" {
		os.Args = []string{"uvm", "invalid_command_xyz"}
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestMainExecutionFailure")
	cmd.Env = append(os.Environ(), "BE_UVM_MAIN_FAILURE=1")
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected main() to exit with error code for invalid command")
	}
}

func TestListRemoteCmd(t *testing.T) {
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	// List remote all
	err := Execute([]string{"list-remote"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "Available remote versions") {
		t.Errorf("list-remote all failed: %v, out: %s", err, outBuf.String())
	}

	// List remote with aliases
	outBuf.Reset()
	err = Execute([]string{"ls-remote"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "Available remote versions") {
		t.Errorf("ls-remote failed: %v, out: %s", err, outBuf.String())
	}

	outBuf.Reset()
	err = Execute([]string{"list-all"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "Available remote versions") {
		t.Errorf("list-all failed: %v, out: %s", err, outBuf.String())
	}

	outBuf.Reset()
	err = Execute([]string{"ls-r"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "Available remote versions") {
		t.Errorf("ls-r failed: %v, out: %s", err, outBuf.String())
	}

	// List remote with flag on list
	outBuf.Reset()
	err = Execute([]string{"list", "--remote"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "Available remote versions") {
		t.Errorf("list --remote failed: %v, out: %s", err, outBuf.String())
	}

	// List remote node
	outBuf.Reset()
	err = Execute([]string{"list-remote", "node"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "Available Node.js versions") {
		t.Errorf("list-remote node failed: %v, out: %s", err, outBuf.String())
	}

	// List remote go
	outBuf.Reset()
	err = Execute([]string{"list-remote", "go"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "Available Go versions") {
		t.Errorf("list-remote go failed: %v, out: %s", err, outBuf.String())
	}

	// List remote python
	outBuf.Reset()
	err = Execute([]string{"list-remote", "python"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "Available Python versions") {
		t.Errorf("list-remote python failed: %v, out: %s", err, outBuf.String())
	}

	// List remote php
	outBuf.Reset()
	err = Execute([]string{"list-remote", "php"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "Available PHP versions") {
		t.Errorf("list-remote php failed: %v, out: %s", err, outBuf.String())
	}

	// List remote java
	outBuf.Reset()
	err = Execute([]string{"list-remote", "java"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "Available Java versions") {
		t.Errorf("list-remote java failed: %v, out: %s", err, outBuf.String())
	}

	// List remote bun
	outBuf.Reset()
	err = Execute([]string{"list-remote", "bun"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "Available Bun versions") {
		t.Errorf("list-remote bun failed: %v, out: %s", err, outBuf.String())
	}

	// List remote aliases (nodejs, golang, py, jdk, openjdk)
	outBuf.Reset()
	err = Execute([]string{"list-remote", "nodejs"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "Available Node.js versions") {
		t.Errorf("list-remote nodejs failed: %v, out: %s", err, outBuf.String())
	}

	outBuf.Reset()
	err = Execute([]string{"list-remote", "jdk"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "Available Java versions") {
		t.Errorf("list-remote jdk failed: %v, out: %s", err, outBuf.String())
	}

	// List --remote <runtime>
	outBuf.Reset()
	err = Execute([]string{"list", "-r", "bun"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "Available Bun versions") {
		t.Errorf("list -r bun failed: %v, out: %s", err, outBuf.String())
	}

	// List remote invalid runtime
	outBuf.Reset()
	err = Execute([]string{"list-remote", "invalid_rt"}, outBuf, errBuf)
	if err == nil {
		t.Errorf("expected error for list-remote invalid_rt")
	}
}

func TestPartialVersionResolution(t *testing.T) {
	tmpHome := t.TempDir()
	oldHome := os.Getenv("HOME")
	oldUserProfile := os.Getenv("USERPROFILE")
	defer func() {
		os.Setenv("HOME", oldHome)
		os.Setenv("USERPROFILE", oldUserProfile)
	}()
	os.Setenv("HOME", tmpHome)
	os.Setenv("USERPROFILE", tmpHome)

	// Mock installations with full patch versions
	_ = os.MkdirAll(filepath.Join(tmpHome, ".uvm", "versions", "node", "v24.20.0", "bin"), 0755)
	_ = os.MkdirAll(filepath.Join(tmpHome, ".uvm", "versions", "go", "go1.22.6", "bin"), 0755)
	_ = os.MkdirAll(filepath.Join(tmpHome, ".uvm", "versions", "python", "3.12.9", "bin"), 0755)
	_ = os.MkdirAll(filepath.Join(tmpHome, ".uvm", "versions", "php", "8.3.17", "bin"), 0755)
	_ = os.MkdirAll(filepath.Join(tmpHome, ".uvm", "versions", "java", "21.0.6", "bin"), 0755)

	bunDir := filepath.Join(tmpHome, ".uvm", "versions", "bun", "1.2.4")
	_ = os.MkdirAll(bunDir, 0755)
	_ = os.WriteFile(filepath.Join(bunDir, "bun"), []byte("bin"), 0755)

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	// Use node with major version "24"
	err := Execute([]string{"use", "node", "24"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "Now using Node.js v24.20.0") {
		t.Errorf("use node 24 failed: %v, out: %s", err, outBuf.String())
	}

	// Use go with major.minor "1.22"
	outBuf.Reset()
	err = Execute([]string{"use", "go", "1.22"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "Now using Go go1.22.6") {
		t.Errorf("use go 1.22 failed: %v, out: %s", err, outBuf.String())
	}

	// Use python with major.minor "3.12"
	outBuf.Reset()
	err = Execute([]string{"use", "python", "3.12"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "Now using Python 3.12.9") {
		t.Errorf("use python 3.12 failed: %v, out: %s", err, outBuf.String())
	}

	// Use php with major.minor "8.3"
	outBuf.Reset()
	err = Execute([]string{"use", "php", "8.3"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "Now using PHP 8.3.17") {
		t.Errorf("use php 8.3 failed: %v, out: %s", err, outBuf.String())
	}

	// Use java with feature version "21"
	outBuf.Reset()
	err = Execute([]string{"use", "java", "21"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "Now using Java 21.0.6") {
		t.Errorf("use java 21 failed: %v, out: %s", err, outBuf.String())
	}

	// Use bun with prefix "1.2"
	outBuf.Reset()
	err = Execute([]string{"use", "bun", "1.2"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "Now using Bun 1.2.4") {
		t.Errorf("use bun 1.2 failed: %v, out: %s", err, outBuf.String())
	}

	// Remove node with major version "24"
	outBuf.Reset()
	err = Execute([]string{"remove", "node", "24"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "v24.20.0 removed successfully") {
		t.Errorf("remove node 24 failed: %v, out: %s", err, outBuf.String())
	}

	// Remove go with major.minor "1.22"
	outBuf.Reset()
	err = Execute([]string{"rm", "go", "1.22"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "go1.22.6 removed successfully") {
		t.Errorf("remove go 1.22 failed: %v, out: %s", err, outBuf.String())
	}

	// Remove python with major.minor "3.12"
	outBuf.Reset()
	err = Execute([]string{"uninstall", "python", "3.12"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "3.12.9 removed successfully") {
		t.Errorf("remove python 3.12 failed: %v, out: %s", err, outBuf.String())
	}

	// Remove php with major.minor "8.3"
	outBuf.Reset()
	err = Execute([]string{"remove", "php", "8.3"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "8.3.17 removed successfully") {
		t.Errorf("remove php 8.3 failed: %v, out: %s", err, outBuf.String())
	}

	// Remove java with feature "21"
	outBuf.Reset()
	err = Execute([]string{"rm", "java", "21"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "21.0.6 removed successfully") {
		t.Errorf("remove java 21 failed: %v, out: %s", err, outBuf.String())
	}

	// Remove bun with prefix "1.2"
	outBuf.Reset()
	err = Execute([]string{"rm", "bun", "1.2"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "1.2.4 removed successfully") {
		t.Errorf("remove bun 1.2 failed: %v, out: %s", err, outBuf.String())
	}
}

func TestCompletions(t *testing.T) {
	cmd := NewRootCmd()

	// Test runtime completion
	rts, _ := runtimeCompletion(cmd, []string{}, "")
	if len(rts) != 6 {
		t.Errorf("expected 6 runtimes in completion, got %d", len(rts))
	}

	rtsNone, _ := runtimeCompletion(cmd, []string{"node"}, "")
	if rtsNone != nil {
		t.Errorf("expected nil for second arg in runtimeCompletion")
	}

	// Test version completion for install
	installFn := versionCompletion(true)
	versInstall, _ := installFn(cmd, []string{"node"}, "")
	if len(versInstall) == 0 {
		t.Errorf("expected version completions for install")
	}

	_, _ = installFn(cmd, []string{"go"}, "")
	_, _ = installFn(cmd, []string{"python"}, "")
	_, _ = installFn(cmd, []string{"php"}, "")
	_, _ = installFn(cmd, []string{"java"}, "")
	_, _ = installFn(cmd, []string{"bun"}, "")

	// Test version completion for use
	useFn := versionCompletion(false)
	versUse, _ := useFn(cmd, []string{"node"}, "")
	if versUse != nil && len(versUse) > 0 {
		// ok
	}
	_, _ = useFn(cmd, []string{"go"}, "")
	_, _ = useFn(cmd, []string{"python"}, "")
	_, _ = useFn(cmd, []string{"php"}, "")
	_, _ = useFn(cmd, []string{"java"}, "")
	_, _ = useFn(cmd, []string{"bun"}, "")
	_, _ = useFn(cmd, []string{}, "")
	_, _ = useFn(cmd, []string{"node", "v20", "extra"}, "")
}

func TestUseAutoSwitchUvmrc(t *testing.T) {
	tmpHome := t.TempDir()
	oldHome := os.Getenv("HOME")
	oldUserProfile := os.Getenv("USERPROFILE")
	defer func() {
		os.Setenv("HOME", oldHome)
		os.Setenv("USERPROFILE", oldUserProfile)
	}()
	os.Setenv("HOME", tmpHome)
	os.Setenv("USERPROFILE", tmpHome)

	// Create installed versions in home
	_ = os.MkdirAll(filepath.Join(tmpHome, ".uvm", "versions", "node", "v20.11.0", "bin"), 0755)
	_ = os.MkdirAll(filepath.Join(tmpHome, ".uvm", "versions", "go", "go1.22.6", "bin"), 0755)
	_ = os.MkdirAll(filepath.Join(tmpHome, ".uvm", "versions", "php", "8.3.17", "bin"), 0755)
	_ = os.MkdirAll(filepath.Join(tmpHome, ".uvm", "versions", "java", "21.0.6", "bin"), 0755)

	bunDir := filepath.Join(tmpHome, ".uvm", "versions", "bun", "1.2.4")
	_ = os.MkdirAll(bunDir, 0755)
	_ = os.WriteFile(filepath.Join(bunDir, "bun"), []byte("bin"), 0755)

	// Create project directory with .uvmrc
	projectDir := filepath.Join(tmpHome, "my-project")
	_ = os.MkdirAll(projectDir, 0755)
	_ = os.WriteFile(filepath.Join(projectDir, ".uvmrc"), []byte("node 20\ngo 1.22\nphp 8.3\njava 21\nbun 1.2\n"), 0644)

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	_ = os.Chdir(projectDir)

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	// Run `uvm use` (0 args) inside project directory
	err := Execute([]string{"use"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "Found") ||
		!strings.Contains(outBuf.String(), "Now using Node.js v20.11.0") ||
		!strings.Contains(outBuf.String(), "Now using PHP 8.3.17") ||
		!strings.Contains(outBuf.String(), "Now using Java 21.0.6") ||
		!strings.Contains(outBuf.String(), "Now using Bun 1.2.4") {
		t.Fatalf("use auto-switch failed: %v, out: %s", err, outBuf.String())
	}

	// Test in empty dir without .uvmrc
	emptyDir := filepath.Join(tmpHome, "empty-dir")
	_ = os.MkdirAll(emptyDir, 0755)
	_ = os.Chdir(emptyDir)

	outBuf.Reset()
	err = Execute([]string{"use"}, outBuf, errBuf)
	if err == nil {
		t.Errorf("expected error when running 'uvm use' with no .uvmrc")
	}

	// Test single arg error
	err = Execute([]string{"use", "node"}, outBuf, errBuf)
	if err == nil {
		t.Errorf("expected error when running 'uvm use node' with 1 argument")
	}
}

func TestPinCmd(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	_ = os.Chdir(tmpDir)

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	// Pin node 20
	err := Execute([]string{"pin", "node", "20"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "Pinned node 20") {
		t.Fatalf("pin node failed: %v, out: %s", err, outBuf.String())
	}

	// Pin go 1.22
	outBuf.Reset()
	err = Execute([]string{"pin", "go", "1.22"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "Pinned go 1.22") {
		t.Fatalf("pin go failed: %v, out: %s", err, outBuf.String())
	}

	// Pin php 8.3
	outBuf.Reset()
	err = Execute([]string{"pin", "php", "8.3"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "Pinned php 8.3") {
		t.Fatalf("pin php failed: %v, out: %s", err, outBuf.String())
	}

	// Pin java 21
	outBuf.Reset()
	err = Execute([]string{"pin", "java", "21"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "Pinned java 21") {
		t.Fatalf("pin java failed: %v, out: %s", err, outBuf.String())
	}

	// Pin bun 1.2
	outBuf.Reset()
	err = Execute([]string{"pin", "bun", "1.2"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "Pinned bun 1.2") {
		t.Fatalf("pin bun failed: %v, out: %s", err, outBuf.String())
	}

	// Pin invalid runtime
	outBuf.Reset()
	err = Execute([]string{"pin", "invalid_rt", "1.0"}, outBuf, errBuf)
	if err == nil {
		t.Errorf("expected error pinning invalid runtime")
	}

	// Verify .uvmrc exists and contains all pinned runtimes
	data, err := os.ReadFile(filepath.Join(tmpDir, ".uvmrc"))
	if err != nil || !strings.Contains(string(data), "node 20") || !strings.Contains(string(data), "go 1.22") || !strings.Contains(string(data), "php 8.3") || !strings.Contains(string(data), "java 21") || !strings.Contains(string(data), "bun 1.2") {
		t.Errorf("invalid .uvmrc content: %s", string(data))
	}
}

func TestInitCmd(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	_ = os.Chdir(tmpDir)

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	// 1. Node Express TS
	err := Execute([]string{"init", "my-node-api", "--lang", "node", "--framework", "express", "--ts", "--crud"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "created successfully") {
		t.Fatalf("init node express ts failed: %v, out: %s", err, outBuf.String())
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "my-node-api", ".uvmrc")); err != nil {
		t.Errorf("expected .uvmrc in scaffolded project")
	}

	// 2. Go Gin
	outBuf.Reset()
	err = Execute([]string{"init", "my-go-api", "--lang", "go", "--framework", "gin", "--crud"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "created successfully") {
		t.Fatalf("init go gin failed: %v, out: %s", err, outBuf.String())
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "my-go-api", "go.mod")); err != nil {
		t.Errorf("expected go.mod in scaffolded project")
	}

	// 3. Go Chi & Fiber
	outBuf.Reset()
	_ = Execute([]string{"create", "my-chi-api", "--lang", "go", "--framework", "chi"}, outBuf, errBuf)
	if _, err := os.Stat(filepath.Join(tmpDir, "my-chi-api", "go.mod")); err != nil {
		t.Errorf("expected my-chi-api/go.mod")
	}

	outBuf.Reset()
	_ = Execute([]string{"scaffold", "my-fiber-api", "--lang", "go", "--framework", "fiber"}, outBuf, errBuf)
	if _, err := os.Stat(filepath.Join(tmpDir, "my-fiber-api", "go.mod")); err != nil {
		t.Errorf("expected my-fiber-api/go.mod")
	}

	// 4. Fastify
	outBuf.Reset()
	_ = Execute([]string{"new", "my-fastify-api", "--lang", "node", "--framework", "fastify"}, outBuf, errBuf)
	if _, err := os.Stat(filepath.Join(tmpDir, "my-fastify-api", "package.json")); err != nil {
		t.Errorf("expected my-fastify-api/package.json")
	}
}
