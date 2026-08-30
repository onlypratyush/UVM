package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestRunAppHelp(t *testing.T) {
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	inBuf := new(bytes.Buffer)

	err := RunApp([]string{"--help"}, inBuf, outBuf, errBuf)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(outBuf.String(), "uvm Visual Installer") {
		t.Errorf("expected help output, got: %s", outBuf.String())
	}
}

func TestRunAppInvalidFlag(t *testing.T) {
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	inBuf := new(bytes.Buffer)

	err := RunApp([]string{"--invalid-flag-xyz"}, inBuf, outBuf, errBuf)
	if err == nil {
		t.Fatalf("expected error on invalid flag")
	}
}

func TestRunAppSilentInstall(t *testing.T) {
	tmpDir := t.TempDir()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	inBuf := new(bytes.Buffer)

	err := RunApp([]string{"--silent", "--dir", tmpDir, "--no-path", "--node-action", "keep", "--confirm-delete"}, inBuf, outBuf, errBuf)
	if err != nil {
		t.Fatalf("silent install failed: %v", err)
	}
	if !strings.Contains(outBuf.String(), "Universal Version Manager") {
		t.Errorf("unexpected output: %s", outBuf.String())
	}
}

func TestRunAppWebMode(t *testing.T) {
	inBuf := new(bytes.Buffer)
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	go func() {
		_ = RunApp([]string{"--web", "--port", "9977"}, inBuf, outBuf, errBuf)
	}()
}

func TestRunFunction(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tmpDir := t.TempDir()
	os.Args = []string{"uvm-installer", "--silent", "--dir", tmpDir, "--no-path"}
	if err := run(); err != nil {
		t.Fatalf("run() returned error: %v", err)
	}
}

func TestMainExecutionSuccess(t *testing.T) {
	if os.Getenv("BE_INSTALLER_MAIN_SUCCESS") == "1" {
		os.Args = []string{"uvm-installer", "--help"}
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestMainExecutionSuccess")
	cmd.Env = append(os.Environ(), "BE_INSTALLER_MAIN_SUCCESS=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("main() exited with error: %v, output: %s", err, string(output))
	}
	if !strings.Contains(string(output), "uvm Visual Installer") {
		t.Errorf("unexpected main output: %s", string(output))
	}
}

func TestMainExecutionFailure(t *testing.T) {
	if os.Getenv("BE_INSTALLER_MAIN_FAILURE") == "1" {
		os.Args = []string{"uvm-installer", "--bad-flag"}
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestMainExecutionFailure")
	cmd.Env = append(os.Environ(), "BE_INSTALLER_MAIN_FAILURE=1")
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected main() to exit with error code for invalid flag")
	}
}
