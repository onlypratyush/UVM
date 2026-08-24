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
			name:        "install go 1.22.0",
			args:        []string{"install", "go", "1.22.0"},
			expectedOut: "Installing go version 1.22.0...\n",
			expectError: false,
		},
		{
			name:        "install python 3.12.2",
			args:        []string{"install", "python", "3.12.2"},
			expectedOut: "Installing python version 3.12.2...\n",
			expectError: false,
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
			name:        "use go 1.22.0",
			args:        []string{"use", "go", "1.22.0"},
			expectedOut: "Using go version 1.22.0\n",
			expectError: false,
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
				if outBuf.String() != tt.expectedOut {
					t.Errorf("expected output %q, got %q", tt.expectedOut, outBuf.String())
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

	// List when no node versions are installed
	_ = Execute([]string{"list"}, outBuf, errBuf)
	if !strings.Contains(outBuf.String(), "No Node.js versions installed") {
		t.Errorf("unexpected output when empty: %s", outBuf.String())
	}

	outBuf.Reset()
	_ = Execute([]string{"list", "node"}, outBuf, errBuf)
	if !strings.Contains(outBuf.String(), "No installed Node.js versions found") {
		t.Errorf("unexpected output for empty list node: %s", outBuf.String())
	}

	// Create multiple installed versions (one active, one inactive)
	vDir1 := filepath.Join(tmpHome, ".uvm", "versions", "node", "v20.11.0")
	vDir2 := filepath.Join(tmpHome, ".uvm", "versions", "node", "v18.20.0")
	_ = os.MkdirAll(vDir1, 0755)
	_ = os.MkdirAll(vDir2, 0755)

	// Set v20.11.0 active
	currentParent := filepath.Join(tmpHome, ".uvm", "current")
	_ = os.MkdirAll(currentParent, 0755)
	_ = os.WriteFile(filepath.Join(currentParent, "node.version"), []byte("v20.11.0"), 0644)

	outBuf.Reset()
	_ = Execute([]string{"ls", "nodejs"}, outBuf, errBuf)
	out := outBuf.String()
	if !strings.Contains(out, "* v20.11.0 (active)") || !strings.Contains(out, "v18.20.0") {
		t.Errorf("expected active and inactive node versions list, got: %s", out)
	}

	// List all when installed
	outBuf.Reset()
	_ = Execute([]string{"list"}, outBuf, errBuf)
	if !strings.Contains(outBuf.String(), "Installed Node.js versions") {
		t.Errorf("expected node list in list all, got: %s", outBuf.String())
	}

	// Other runtimes
	outBuf.Reset()
	_ = Execute([]string{"list", "go"}, outBuf, errBuf)
	if !strings.Contains(outBuf.String(), "Listing installed versions for go") {
		t.Errorf("unexpected go list output: %s", outBuf.String())
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

	vDir := filepath.Join(tmpHome, ".uvm", "versions", "node", "v20.11.0")
	_ = os.MkdirAll(vDir, 0755)

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	// Remove node version
	err := Execute([]string{"remove", "node", "20.11.0"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "removed successfully") {
		t.Fatalf("remove failed: %v, out: %s", err, outBuf.String())
	}

	// Remove non-node runtime
	outBuf.Reset()
	err = Execute([]string{"rm", "go", "1.22.0"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "Removing go version 1.22.0") {
		t.Errorf("unexpected rm go output: %s", outBuf.String())
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
	if err != nil || !strings.Contains(outBuf.String(), "No active Node.js version") {
		t.Errorf("expected no active version message, got: %s", outBuf.String())
	}

	// Set active version
	currentParent := filepath.Join(tmpHome, ".uvm", "current")
	_ = os.MkdirAll(currentParent, 0755)
	_ = os.WriteFile(filepath.Join(currentParent, "node.version"), []byte("v20.11.0"), 0644)

	outBuf.Reset()
	err = Execute([]string{"current", "node"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "v20.11.0") {
		t.Errorf("expected current version v20.11.0, got: %s", outBuf.String())
	}

	outBuf.Reset()
	err = Execute([]string{"current", "go"}, outBuf, errBuf)
	if err != nil || !strings.Contains(outBuf.String(), "none") {
		t.Errorf("expected none for go, got: %s", outBuf.String())
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

	os.Args = []string{"uvm", "list", "go"}
	if err := run(); err != nil {
		t.Fatalf("run() returned error: %v", err)
	}
}

func TestMainExecutionSuccess(t *testing.T) {
	if os.Getenv("BE_UVM_MAIN_SUCCESS") == "1" {
		os.Args = []string{"uvm", "list", "go"}
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestMainExecutionSuccess")
	cmd.Env = append(os.Environ(), "BE_UVM_MAIN_SUCCESS=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("process exited with err: %v, output: %s", err, string(output))
	}
	if !strings.Contains(string(output), "Listing installed versions for go") {
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
