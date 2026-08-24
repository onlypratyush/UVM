package installer

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// helper to create a guaranteed impossible directory across all platforms (Windows, Linux, macOS)
func getImpossibleDir(t *testing.T) string {
	tmpDir := t.TempDir()
	blockingFile := filepath.Join(tmpDir, "blocking_file.txt")
	_ = os.WriteFile(blockingFile, []byte("block"), 0644)
	return filepath.Join(blockingFile, "sub_dir", "bin")
}

func TestBrowserOpener(t *testing.T) {
	_ = BrowserOpener("http://localhost:8484", "darwin")
	_ = BrowserOpener("http://localhost:8484", "windows")
	_ = BrowserOpener("http://localhost:8484", "linux")
}

func TestGetDefaultInstallDir(t *testing.T) {
	d1 := GetDefaultInstallDir("/home/user", "linux")
	if d1 != filepath.Join("/home/user", ".uvm", "bin") {
		t.Errorf("unexpected dir: %s", d1)
	}

	d2 := GetDefaultInstallDir("/Users/user", "darwin")
	if d2 != filepath.Join("/Users/user", ".uvm", "bin") {
		t.Errorf("unexpected dir: %s", d2)
	}

	d3 := GetDefaultInstallDir("C:\\Users\\user", "windows")
	if d3 != filepath.Join("C:\\Users\\user", ".uvm", "bin") {
		t.Errorf("unexpected dir: %s", d3)
	}

	d4 := GetDefaultInstallDir("", "linux")
	if d4 != filepath.Join(".", ".uvm", "bin") {
		t.Errorf("unexpected dir for empty home: %s", d4)
	}
}

func TestDetectSystemInfo(t *testing.T) {
	tmpHome := t.TempDir()

	info := DetectSystemInfo(tmpHome, "linux", "amd64")
	if info.OS != "linux" || info.Arch != "amd64" {
		t.Errorf("unexpected info: %+v", info)
	}
	if info.IsInstalled {
		t.Errorf("expected not installed in clean temp home")
	}

	// Create dummy installed binary
	installDir := filepath.Join(tmpHome, ".uvm", "bin")
	_ = os.MkdirAll(installDir, 0755)
	_ = os.WriteFile(filepath.Join(installDir, "uvm"), []byte("bin"), 0755)

	infoInstalled := DetectSystemInfo(tmpHome, "linux", "amd64")
	if !infoInstalled.IsInstalled {
		t.Errorf("expected installed state to be true")
	}

	// Test windows detection
	infoWin := DetectSystemInfo(tmpHome, "windows", "arm64")
	if infoWin.DetectedShell != "PowerShell" {
		t.Errorf("expected PowerShell on windows, got %s", infoWin.DetectedShell)
	}

	// Test shell detections
	oldShell := os.Getenv("SHELL")
	defer os.Setenv("SHELL", oldShell)

	os.Setenv("SHELL", "/bin/zsh")
	infoZsh := DetectSystemInfo(tmpHome, "darwin", "arm64")
	if infoZsh.DetectedShell != "zsh" {
		t.Errorf("expected zsh shell, got %s", infoZsh.DetectedShell)
	}

	os.Setenv("SHELL", "/usr/bin/fish")
	infoFish := DetectSystemInfo(tmpHome, "linux", "amd64")
	if infoFish.DetectedShell != "fish" {
		t.Errorf("expected fish shell, got %s", infoFish.DetectedShell)
	}

	os.Setenv("SHELL", "/bin/bash")
	infoBash := DetectSystemInfo(tmpHome, "linux", "amd64")
	if infoBash.DetectedShell != "bash" {
		t.Errorf("expected bash shell, got %s", infoBash.DetectedShell)
	}

	// Test fallback with empty parameters and USERPROFILE
	oldHome := os.Getenv("HOME")
	oldUserProfile := os.Getenv("USERPROFILE")
	defer func() {
		os.Setenv("HOME", oldHome)
		os.Setenv("USERPROFILE", oldUserProfile)
	}()

	os.Setenv("HOME", "/custom/home")
	infoDefault := DetectSystemInfo("", "", "")
	if infoDefault.HomeDir == "" {
		t.Errorf("expected default homeDir to not be empty")
	}

	os.Unsetenv("HOME")
	os.Setenv("USERPROFILE", "/custom/profile")
	_ = DetectSystemInfo("", "", "")

	os.Unsetenv("USERPROFILE")
	_ = DetectSystemInfo("", "", "")
}

func TestUpdateShellProfile(t *testing.T) {
	tmpHome := t.TempDir()

	// 1. Windows path
	msg, err := UpdateShellProfile("C:\\test", tmpHome, "PowerShell", "windows")
	if err != nil || !strings.Contains(msg, "Windows User PATH") {
		t.Errorf("unexpected windows profile result: %s, %v", msg, err)
	}

	// 2. Zsh
	targetDir := filepath.Join(tmpHome, ".uvm", "bin")
	zshFile, err := UpdateShellProfile(targetDir, tmpHome, "zsh", "darwin")
	if err != nil {
		t.Fatalf("zsh profile failed: %v", err)
	}
	content, _ := os.ReadFile(zshFile)
	if !strings.Contains(string(content), "export PATH=") {
		t.Errorf("expected zsh export in %s", zshFile)
	}

	// Idempotency check
	zshFile2, err := UpdateShellProfile(targetDir, tmpHome, "zsh", "darwin")
	if err != nil || zshFile2 != zshFile {
		t.Errorf("idempotency check failed: %s, %v", zshFile2, err)
	}

	// 3. Fish
	fishFile, err := UpdateShellProfile(targetDir, tmpHome, "fish", "linux")
	if err != nil {
		t.Fatalf("fish profile failed: %v", err)
	}
	fishContent, _ := os.ReadFile(fishFile)
	if !strings.Contains(string(fishContent), "fish_add_path") {
		t.Errorf("expected fish_add_path in %s", fishFile)
	}

	// 4. Bash with existing .bash_profile
	tmpHome2 := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpHome2, ".bash_profile"), []byte("# profile\n"), 0644)
	bashProfileFile, err := UpdateShellProfile(targetDir, tmpHome2, "bash", "darwin")
	if err != nil || !strings.Contains(bashProfileFile, ".bash_profile") {
		t.Errorf("expected .bash_profile to be updated: %s, %v", bashProfileFile, err)
	}

	// 5. Bash with default .bashrc
	tmpHome3 := t.TempDir()
	bashrcFile, err := UpdateShellProfile(targetDir, tmpHome3, "bash", "linux")
	if err != nil || !strings.Contains(bashrcFile, ".bashrc") {
		t.Errorf("expected .bashrc to be updated: %s, %v", bashrcFile, err)
	}

	// 6. Error handling with impossible directory
	impossibleHome := getImpossibleDir(t)
	_, err = UpdateShellProfile(targetDir, impossibleHome, "zsh", "linux")
	if err == nil {
		t.Errorf("expected error for impossible path")
	}
}

func TestInstallAndUninstall(t *testing.T) {
	tmpHome := t.TempDir()
	targetDir := filepath.Join(tmpHome, "custom_bin")

	// 1. Install with specific options and source binary in bin/
	_ = os.MkdirAll("bin", 0755)
	_ = os.WriteFile("bin/uvm", []byte("#!/bin/sh\necho test\n"), 0755)
	defer os.RemoveAll("bin")

	opts := Options{
		InstallDir: targetDir,
		ModifyPath: true,
		ShellType:  "bash",
	}
	res, err := Install(opts, tmpHome, "linux")
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}
	if !res.Success {
		t.Errorf("expected success, got: %+v", res)
	}
	if _, err := os.Stat(res.BinaryPath); err != nil {
		t.Errorf("binary not found at %s: %v", res.BinaryPath, err)
	}

	// 2. Install when uvm binary exists in current working dir
	_ = os.WriteFile("uvm", []byte("#!/bin/sh\necho local\n"), 0755)
	defer os.Remove("uvm")

	optsCwd := Options{
		InstallDir: filepath.Join(tmpHome, "cwd_bin"),
		ModifyPath: false,
	}
	resCwd, err := Install(optsCwd, tmpHome, "linux")
	if err != nil || !resCwd.Success {
		t.Fatalf("Install from cwd failed: %v", err)
	}

	// 3. Install with default path and empty OS
	optsDefault := Options{
		ModifyPath: true,
	}
	resDefault, err := Install(optsDefault, tmpHome, "")
	if err != nil || !resDefault.Success {
		t.Fatalf("Install with default options failed: %v", err)
	}

	// 4. Install on windows
	optsWin := Options{
		InstallDir: filepath.Join(tmpHome, "win_bin"),
		ModifyPath: true,
	}
	resWin, err := Install(optsWin, tmpHome, "windows")
	if err != nil || !resWin.Success {
		t.Fatalf("Install on windows failed: %v", err)
	}
	if !strings.HasSuffix(resWin.BinaryPath, "uvm.exe") {
		t.Errorf("expected uvm.exe binary on windows, got %s", resWin.BinaryPath)
	}

	// 5. Uninstall linux
	unres, err := Uninstall(targetDir, tmpHome, "linux")
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}
	if !unres.Success {
		t.Errorf("expected uninstall success: %+v", unres)
	}

	// 6. Uninstall windows
	unresWin, err := Uninstall(filepath.Join(tmpHome, "win_bin"), tmpHome, "windows")
	if err != nil || !unresWin.Success {
		t.Fatalf("Uninstall windows failed: %v", err)
	}

	// 7. Uninstall default path and empty OS
	_, _ = Uninstall("", tmpHome, "")

	// 8. Install error with impossible directory
	optsErr := Options{
		InstallDir: getImpossibleDir(t),
	}
	resErr, err := Install(optsErr, tmpHome, "linux")
	if err == nil || resErr.Success {
		t.Errorf("expected error for impossible directory")
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "source.txt")
	dst := filepath.Join(tmpDir, "dest.txt")

	_ = os.WriteFile(src, []byte("test content"), 0644)
	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	content, _ := os.ReadFile(dst)
	if string(content) != "test content" {
		t.Errorf("unexpected content: %s", string(content))
	}

	if err := copyFile(filepath.Join(tmpDir, "non_existent"), dst); err == nil {
		t.Errorf("expected error for non-existent source")
	}

	impossibleDst := getImpossibleDir(t)
	if err := copyFile(src, impossibleDst); err == nil {
		t.Errorf("expected error for impossible dest")
	}
}

func TestRunVisualCLI(t *testing.T) {
	tmpHome := t.TempDir()
	outBuf := new(bytes.Buffer)
	inBuf := new(bytes.Buffer)

	// 1. Install via Visual CLI with custom options
	opts := Options{
		InstallDir: filepath.Join(tmpHome, ".uvm", "bin"),
		ModifyPath: true,
		ShellType:  "zsh",
	}
	err := RunVisualCLI(opts, inBuf, outBuf, tmpHome, "darwin")
	if err != nil {
		t.Fatalf("RunVisualCLI failed: %v", err)
	}
	out := outBuf.String()
	if !strings.Contains(out, "Universal Version Manager") || !strings.Contains(out, "Quick Start") {
		t.Errorf("unexpected visual CLI output: %s", out)
	}

	// 2. Install via Visual CLI with empty options / default OS
	outBuf.Reset()
	err = RunVisualCLI(Options{}, inBuf, outBuf, tmpHome, "")
	if err != nil {
		t.Fatalf("RunVisualCLI with default opts failed: %v", err)
	}

	// 3. Uninstall via Visual CLI
	outBuf.Reset()
	optsUninstall := Options{
		InstallDir: filepath.Join(tmpHome, ".uvm", "bin"),
		Uninstall:  true,
	}
	err = RunVisualCLI(optsUninstall, inBuf, outBuf, tmpHome, "darwin")
	if err != nil {
		t.Fatalf("RunVisualCLI uninstall failed: %v", err)
	}
	if !strings.Contains(outBuf.String(), "uninstalled successfully") {
		t.Errorf("unexpected uninstall output: %s", outBuf.String())
	}

	// 4. Error case with impossible directory
	optsBad := Options{
		InstallDir: getImpossibleDir(t),
	}
	err = RunVisualCLI(optsBad, inBuf, outBuf, tmpHome, "darwin")
	if err == nil {
		t.Errorf("expected error for invalid directory in RunVisualCLI")
	}
}

func TestWebServerEndpoints(t *testing.T) {
	tmpHome := t.TempDir()
	srv := NewWebServer(Options{Port: 9898}, tmpHome, "linux")

	// 1. GET /
	reqIndex := httptest.NewRequest(http.MethodGet, "/", nil)
	wIndex := httptest.NewRecorder()
	srv.Handler.ServeHTTP(wIndex, reqIndex)
	if wIndex.Code != http.StatusOK || !strings.Contains(wIndex.Body.String(), "uvm") {
		t.Errorf("GET / returned unexpected response: %d", wIndex.Code)
	}

	// 2. GET /notfound
	req404 := httptest.NewRequest(http.MethodGet, "/random-page", nil)
	w404 := httptest.NewRecorder()
	srv.Handler.ServeHTTP(w404, req404)
	if w404.Code != http.StatusNotFound {
		t.Errorf("expected 404 for invalid page, got %d", w404.Code)
	}

	// 3. GET /api/status
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	wStatus := httptest.NewRecorder()
	srv.Handler.ServeHTTP(wStatus, reqStatus)
	if wStatus.Code != http.StatusOK {
		t.Errorf("GET /api/status returned %d", wStatus.Code)
	}
	var sysInfo SystemInfo
	if err := json.NewDecoder(wStatus.Body).Decode(&sysInfo); err != nil || sysInfo.OS != "linux" {
		t.Errorf("invalid status JSON: %+v, err: %v", sysInfo, err)
	}

	// 4. POST /api/install
	installPayload := Options{
		InstallDir: filepath.Join(tmpHome, "web_bin"),
		ModifyPath: true,
	}
	body, _ := json.Marshal(installPayload)
	reqInstall := httptest.NewRequest(http.MethodPost, "/api/install", bytes.NewReader(body))
	wInstall := httptest.NewRecorder()
	srv.Handler.ServeHTTP(wInstall, reqInstall)
	if wInstall.Code != http.StatusOK {
		t.Errorf("POST /api/install returned %d", wInstall.Code)
	}

	// 5. POST /api/install invalid method
	reqInstallBadMethod := httptest.NewRequest(http.MethodGet, "/api/install", nil)
	wInstallBadMethod := httptest.NewRecorder()
	srv.Handler.ServeHTTP(wInstallBadMethod, reqInstallBadMethod)
	if wInstallBadMethod.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET /api/install, got %d", wInstallBadMethod.Code)
	}

	// 6. POST /api/install with invalid payload fallback
	reqInstallInvalidBody := httptest.NewRequest(http.MethodPost, "/api/install", bytes.NewReader([]byte("bad-json")))
	wInstallInvalidBody := httptest.NewRecorder()
	srv.Handler.ServeHTTP(wInstallInvalidBody, reqInstallInvalidBody)
	if wInstallInvalidBody.Code != http.StatusOK {
		t.Errorf("expected fallback install, got %d", wInstallInvalidBody.Code)
	}

	// 7. POST /api/uninstall
	reqUninstall := httptest.NewRequest(http.MethodPost, "/api/uninstall", bytes.NewReader(body))
	wUninstall := httptest.NewRecorder()
	srv.Handler.ServeHTTP(wUninstall, reqUninstall)
	if wUninstall.Code != http.StatusOK {
		t.Errorf("POST /api/uninstall returned %d", wUninstall.Code)
	}

	// 8. GET /api/uninstall invalid method
	reqUninstallBadMethod := httptest.NewRequest(http.MethodGet, "/api/uninstall", nil)
	wUninstallBadMethod := httptest.NewRecorder()
	srv.Handler.ServeHTTP(wUninstallBadMethod, reqUninstallBadMethod)
	if wUninstallBadMethod.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET /api/uninstall, got %d", wUninstallBadMethod.Code)
	}

	// 9. GET /api/verify
	reqVerify := httptest.NewRequest(http.MethodGet, "/api/verify", nil)
	wVerify := httptest.NewRecorder()
	srv.Handler.ServeHTTP(wVerify, reqVerify)
	if wVerify.Code != http.StatusOK {
		t.Errorf("GET /api/verify returned %d", wVerify.Code)
	}

	// 10. POST /api/install with error
	badInstallPayload := Options{InstallDir: getImpossibleDir(t)}
	badBody, _ := json.Marshal(badInstallPayload)
	reqBadInstall := httptest.NewRequest(http.MethodPost, "/api/install", bytes.NewReader(badBody))
	wBadInstall := httptest.NewRecorder()
	srv.Handler.ServeHTTP(wBadInstall, reqBadInstall)
	if wBadInstall.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for bad install, got %d", wBadInstall.Code)
	}
}

func TestStartWebUIBackground(t *testing.T) {
	oldOpener := BrowserOpener
	defer func() { BrowserOpener = oldOpener }()
	browserOpened := false
	BrowserOpener = func(url string, goos string) error {
		browserOpened = true
		return nil
	}

	tmpHome := t.TempDir()
	opts := Options{Port: 0}

	go func() {
		_ = StartWebUI(opts, tmpHome, "linux")
	}()

	time.Sleep(150 * time.Millisecond)
	if !browserOpened {
		t.Logf("mock browser opener called asynchronously")
	}
}
