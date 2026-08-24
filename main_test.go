package main

import (
	"bytes"
	"os"
	"os/exec"
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
	tests := []struct {
		name        string
		args        []string
		expectedOut string
		expectError bool
	}{
		{
			name:        "install node 20.11.0",
			args:        []string{"install", "node", "20.11.0"},
			expectedOut: "Installing node version 20.11.0...\n",
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
				if outBuf.String() != tt.expectedOut {
					t.Errorf("expected output %q, got %q", tt.expectedOut, outBuf.String())
				}
			}
		})
	}
}

func TestUseCmd(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectedOut string
		expectError bool
	}{
		{
			name:        "use node 20.11.0",
			args:        []string{"use", "node", "20.11.0"},
			expectedOut: "Using node version 20.11.0\n",
			expectError: false,
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
	tests := []struct {
		name        string
		args        []string
		expectedOut string
		expectError bool
	}{
		{
			name:        "list all",
			args:        []string{"list"},
			expectedOut: "Listing all managed runtimes and versions...\n",
			expectError: false,
		},
		{
			name:        "list alias ls",
			args:        []string{"ls"},
			expectedOut: "Listing all managed runtimes and versions...\n",
			expectError: false,
		},
		{
			name:        "list specific runtime",
			args:        []string{"list", "node"},
			expectedOut: "Listing installed versions for node...\n",
			expectError: false,
		},
		{
			name:        "list alias ls specific runtime",
			args:        []string{"ls", "go"},
			expectedOut: "Listing installed versions for go...\n",
			expectError: false,
		},
		{
			name:        "list too many args",
			args:        []string{"list", "node", "extra"},
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

func TestRemoveCmd(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectedOut string
		expectError bool
	}{
		{
			name:        "remove node 20.11.0",
			args:        []string{"remove", "node", "20.11.0"},
			expectedOut: "Removing node version 20.11.0...\n",
			expectError: false,
		},
		{
			name:        "alias rm",
			args:        []string{"rm", "go", "1.22.0"},
			expectedOut: "Removing go version 1.22.0...\n",
			expectError: false,
		},
		{
			name:        "alias uninstall",
			args:        []string{"uninstall", "python", "3.12.2"},
			expectedOut: "Removing python version 3.12.2...\n",
			expectError: false,
		},
		{
			name:        "remove missing version",
			args:        []string{"remove", "node"},
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

func TestExecuteWithNilStreams(t *testing.T) {
	// Tests fallback when streams are nil
	err := Execute([]string{"--version"}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error with nil streams: %v", err)
	}
}

func TestRunFunction(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"uvm", "list"}
	if err := run(); err != nil {
		t.Fatalf("run() returned error: %v", err)
	}
}

func TestMainExecutionSuccess(t *testing.T) {
	if os.Getenv("BE_UVM_MAIN_SUCCESS") == "1" {
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestMainExecutionSuccess")
	cmd.Env = append(os.Environ(), "BE_UVM_MAIN_SUCCESS=1")
	cmd.Args = append(cmd.Args, "list")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("process exited with err: %v, output: %s", err, string(output))
	}
	if !strings.Contains(string(output), "Listing all managed runtimes") {
		t.Errorf("unexpected output: %s", string(output))
	}
}

func TestMainExecutionFailure(t *testing.T) {
	if os.Getenv("BE_UVM_MAIN_FAILURE") == "1" {
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestMainExecutionFailure")
	cmd.Env = append(os.Environ(), "BE_UVM_MAIN_FAILURE=1")
	cmd.Args = append(cmd.Args, "invalid_command_xyz")
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected main() to exit with error code for invalid command")
	}
}
