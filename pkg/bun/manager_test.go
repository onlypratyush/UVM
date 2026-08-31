package bun

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createMockBunZip(topDir string, isWindows bool) []byte {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	// Directory
	_, _ = zw.Create(topDir + "/")

	// Binaries
	if isWindows {
		fw, _ := zw.Create(topDir + "/bun.exe")
		_, _ = fw.Write([]byte("fake bun windows binary"))
	} else {
		fw, _ := zw.Create(topDir + "/bun")
		_, _ = fw.Write([]byte("#!/bin/sh\necho 1.2.4\n"))
	}

	_ = zw.Close()
	return buf.Bytes()
}

func TestNewManager(t *testing.T) {
	m1 := NewManager("/tmp/test_bun_uvm")
	if m1.BaseDir != "/tmp/test_bun_uvm" {
		t.Errorf("expected BaseDir /tmp/test_bun_uvm, got %s", m1.BaseDir)
	}

	m2 := NewManager("")
	if m2.BaseDir == "" {
		t.Errorf("expected default baseDir to not be empty")
	}

	oldHome := os.Getenv("HOME")
	oldUserProf := os.Getenv("USERPROFILE")
	defer func() {
		os.Setenv("HOME", oldHome)
		os.Setenv("USERPROFILE", oldUserProf)
	}()

	os.Setenv("HOME", "")
	os.Setenv("USERPROFILE", "/custom/userprof")
	m3 := NewManager("")
	if !strings.HasPrefix(m3.BaseDir, "/custom/userprof") {
		t.Errorf("expected baseDir with USERPROFILE, got %s", m3.BaseDir)
	}
}

func TestNormalizeVersion(t *testing.T) {
	m := NewManager("")

	cases := []struct {
		input    string
		expected string
	}{
		{"1.2.4", "1.2.4"},
		{"v1.2.4", "1.2.4"},
		{"bun-v1.2.4", "1.2.4"},
		{"bun-1.2.4", "1.2.4"},
		{"bun1.2", "1.2"},
		{"1.2", "1.2"},
		{"latest", "latest"},
		{"lts", "lts"},
		{"", ""},
	}

	for _, c := range cases {
		got := m.NormalizeVersion(c.input)
		if got != c.expected {
			t.Errorf("NormalizeVersion(%q) = %q, expected %q", c.input, got, c.expected)
		}
	}
}

func TestResolveVersion(t *testing.T) {
	m := NewManager("")

	if v, _ := m.ResolveRemoteVersion("latest"); v != "1.2.4" {
		t.Errorf("expected latest to resolve to 1.2.4, got %s", v)
	}
	if v, _ := m.ResolveRemoteVersion("lts"); v != "1.2.4" {
		t.Errorf("expected lts to resolve to 1.2.4, got %s", v)
	}
	if v, _ := m.ResolveRemoteVersion("1.2"); v != "1.2.4" {
		t.Errorf("expected 1.2 to resolve to 1.2.4, got %s", v)
	}
	if v, _ := m.ResolveRemoteVersion("1.1"); v != "1.1.45" {
		t.Errorf("expected 1.1 to resolve to 1.1.45, got %s", v)
	}
	if v, _ := m.ResolveRemoteVersion("1.0"); v != "1.0.36" {
		t.Errorf("expected 1.0 to resolve to 1.0.36, got %s", v)
	}
	if v, _ := m.ResolveRemoteVersion("1.2.0"); v != "1.2.0" {
		t.Errorf("expected 1.2.0 to stay 1.2.0, got %s", v)
	}
	if _, err := m.ResolveRemoteVersion(""); err == nil {
		t.Errorf("expected error for empty version")
	}

	alias, _ := m.ResolveVersion("1.2")
	if alias != "1.2.4" {
		t.Errorf("expected ResolveVersion alias 1.2.4, got %s", alias)
	}
}

func TestGetArchiveTarget(t *testing.T) {
	m := NewManager("")

	m.GOOS = "darwin"
	m.GOARCH = "arm64"
	url, file, err := m.GetArchiveTarget("1.2.4")
	if err != nil || file != "bun-darwin-aarch64.zip" || !strings.Contains(url, "bun-v1.2.4/bun-darwin-aarch64.zip") {
		t.Errorf("unexpected darwin arm64 target: url=%s, file=%s, err=%v", url, file, err)
	}

	m.GOOS = "darwin"
	m.GOARCH = "amd64"
	_, file, _ = m.GetArchiveTarget("1.2.4")
	if file != "bun-darwin-x64.zip" {
		t.Errorf("expected bun-darwin-x64.zip, got %s", file)
	}

	m.GOOS = "linux"
	m.GOARCH = "amd64"
	_, file, _ = m.GetArchiveTarget("1.2.4")
	if file != "bun-linux-x64.zip" {
		t.Errorf("expected bun-linux-x64.zip, got %s", file)
	}

	m.GOOS = "linux"
	m.GOARCH = "arm64"
	_, file, _ = m.GetArchiveTarget("1.2.4")
	if file != "bun-linux-aarch64.zip" {
		t.Errorf("expected bun-linux-aarch64.zip, got %s", file)
	}

	m.GOOS = "windows"
	m.GOARCH = "amd64"
	_, file, _ = m.GetArchiveTarget("1.2.4")
	if file != "bun-windows-x64.zip" {
		t.Errorf("expected bun-windows-x64.zip, got %s", file)
	}

	m.GOOS = "unsupported_os"
	_, _, err = m.GetArchiveTarget("1.2.4")
	if err == nil {
		t.Errorf("expected error for unsupported OS")
	}
}

func TestInstallZipUnixAndWindows(t *testing.T) {
	// 1. Unix install
	mockZip := createMockBunZip("bun-darwin-aarch64", false)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(mockZip)
	}))
	defer server.Close()

	tmpBase := t.TempDir()
	m := NewManager(tmpBase)
	m.BunDistURL = server.URL
	m.GOOS = "darwin"
	m.GOARCH = "arm64"

	outBuf := new(bytes.Buffer)
	err := m.Install("1.2.4", outBuf)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	outStr := outBuf.String()
	if !strings.Contains(outStr, "installed successfully") {
		t.Errorf("expected success output, got: %s", outStr)
	}

	// Verify shims
	bunShim := filepath.Join(m.BinDir(), "bun")
	bunxShim := filepath.Join(m.BinDir(), "bunx")
	if _, err := os.Lstat(bunShim); err != nil {
		t.Errorf("expected bun shim symlink")
	}
	if _, err := os.Lstat(bunxShim); err != nil {
		t.Errorf("expected bunx shim symlink")
	}

	// 2. Windows install
	mockWinZip := createMockBunZip("bun-windows-x64", true)
	serverWin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(mockWinZip)
	}))
	defer serverWin.Close()

	tmpBaseWin := t.TempDir()
	mWin := NewManager(tmpBaseWin)
	mWin.BunDistURL = serverWin.URL
	mWin.GOOS = "windows"
	mWin.GOARCH = "amd64"

	err = mWin.Install("1.2.4", new(bytes.Buffer))
	if err != nil {
		t.Fatalf("Windows Install failed: %v", err)
	}

	shimExe := filepath.Join(mWin.BinDir(), "bun.exe")
	shimCmd := filepath.Join(mWin.BinDir(), "bun.cmd")
	shimBunxExe := filepath.Join(mWin.BinDir(), "bunx.exe")
	shimBunxCmd := filepath.Join(mWin.BinDir(), "bunx.cmd")

	if _, err := os.Stat(shimExe); err != nil {
		t.Errorf("expected %s to exist", shimExe)
	}
	if _, err := os.Stat(shimCmd); err != nil {
		t.Errorf("expected %s to exist", shimCmd)
	}
	if _, err := os.Stat(shimBunxExe); err != nil {
		t.Errorf("expected %s to exist", shimBunxExe)
	}
	if _, err := os.Stat(shimBunxCmd); err != nil {
		t.Errorf("expected %s to exist", shimBunxCmd)
	}
}

func TestUseAndCurrent(t *testing.T) {
	tmpBase := t.TempDir()
	m := NewManager(tmpBase)
	m.GOOS = "linux"
	m.GOARCH = "amd64"

	// Mock installed versions
	v1Dir := filepath.Join(m.VersionsDir(), "1.2.4")
	_ = os.MkdirAll(v1Dir, 0755)
	_ = os.WriteFile(filepath.Join(v1Dir, "bun"), []byte("bin1"), 0755)

	v2Dir := filepath.Join(m.VersionsDir(), "1.1.45")
	_ = os.MkdirAll(v2Dir, 0755)
	_ = os.WriteFile(filepath.Join(v2Dir, "bun"), []byte("bin2"), 0755)

	// Test Current before Use
	_, err := m.Current()
	if err == nil {
		t.Errorf("expected error when no version is selected")
	}

	// Use v1
	outBuf := new(bytes.Buffer)
	err = m.Use("1.2", outBuf)
	if err != nil {
		t.Fatalf("Use 1.2 failed: %v", err)
	}

	curr, err := m.Current()
	if err != nil || curr != "1.2.4" {
		t.Errorf("expected current version 1.2.4, got %s (err: %v)", curr, err)
	}

	// Use uninstalled
	err = m.Use("1.0", new(bytes.Buffer))
	if err == nil {
		t.Errorf("expected error using uninstalled version")
	}
}

func TestListInstalled(t *testing.T) {
	tmpBase := t.TempDir()
	m := NewManager(tmpBase)

	// Empty list
	list, err := m.ListInstalled()
	if err != nil || len(list) != 0 {
		t.Errorf("expected empty list, got %+v (err: %v)", list, err)
	}

	// Create versions
	_ = os.MkdirAll(filepath.Join(m.VersionsDir(), "1.2.4"), 0755)
	_ = os.MkdirAll(filepath.Join(m.VersionsDir(), "1.1.45"), 0755)

	_ = m.Use("1.2.4", new(bytes.Buffer))

	list, err = m.ListInstalled()
	if err != nil || len(list) != 2 {
		t.Fatalf("expected 2 installed versions, got %d (err: %v)", len(list), err)
	}

	foundActive := false
	for _, v := range list {
		if v.Version == "1.2.4" && v.IsActive {
			foundActive = true
		}
	}
	if !foundActive {
		t.Errorf("expected 1.2.4 to be active")
	}
}

func TestRemove(t *testing.T) {
	tmpBase := t.TempDir()
	m := NewManager(tmpBase)
	m.GOOS = "linux"

	_ = os.MkdirAll(filepath.Join(m.VersionsDir(), "1.2.4"), 0755)
	_ = os.WriteFile(filepath.Join(m.VersionsDir(), "1.2.4", "bun"), []byte("bin"), 0755)
	_ = m.Use("1.2.4", new(bytes.Buffer))

	outBuf := new(bytes.Buffer)
	err := m.Remove("1.2", outBuf)
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	if !strings.Contains(outBuf.String(), "removed successfully") {
		t.Errorf("expected success message, got: %s", outBuf.String())
	}

	// Check active state cleaned
	_, err = m.Current()
	if err == nil {
		t.Errorf("expected Current to fail after removing active version")
	}

	// Remove non-installed
	err = m.Remove("1.2.4", new(bytes.Buffer))
	if err == nil {
		t.Errorf("expected error removing non-installed version")
	}
}

func TestListRemote(t *testing.T) {
	ghMock := []GitHubRelease{
		{TagName: "bun-v1.2.4", Name: "Bun v1.2.4"},
		{TagName: "bun-v1.2.0", Name: "Bun v1.2.0"},
		{TagName: "bun-v1.1.45", Name: "Bun v1.1.45"},
	}
	data, _ := json.Marshal(ghMock)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}))
	defer server.Close()

	m := NewManager(t.TempDir())
	m.ReleasesAPIURL = server.URL

	releases, err := m.ListRemote(2)
	if err != nil || len(releases) != 2 {
		t.Fatalf("expected 2 releases from server, got %d (err: %v)", len(releases), err)
	}

	// Test fallback with invalid server
	m.ReleasesAPIURL = "http://invalid-url-that-fails-12345.com"
	fallbackList, err := m.ListRemote(5)
	if err != nil || len(fallbackList) == 0 {
		t.Errorf("expected fallback releases, got: %+v (err: %v)", fallbackList, err)
	}
}

func TestResolveInstalledVersionPartial(t *testing.T) {
	tmpBase := t.TempDir()
	m := NewManager(tmpBase)

	_ = os.MkdirAll(filepath.Join(m.VersionsDir(), "1.2.4"), 0755)
	_ = os.MkdirAll(filepath.Join(m.VersionsDir(), "1.2.0"), 0755)
	_ = os.MkdirAll(filepath.Join(m.VersionsDir(), "1.1.45"), 0755)

	res, err := m.ResolveInstalledVersion("1.2")
	if err != nil || res != "1.2.4" {
		t.Errorf("expected 1.2 -> 1.2.4, got %s (err: %v)", res, err)
	}

	res, err = m.ResolveInstalledVersion("1.1")
	if err != nil || res != "1.1.45" {
		t.Errorf("expected 1.1 -> 1.1.45, got %s (err: %v)", res, err)
	}

	res, err = m.ResolveInstalledVersion("1.2.0")
	if err != nil || res != "1.2.0" {
		t.Errorf("expected exact 1.2.0, got %s (err: %v)", res, err)
	}
}

func TestCompareVersions(t *testing.T) {
	if compareVersions("1.2.4", "1.2.0") <= 0 {
		t.Errorf("expected 1.2.4 > 1.2.0")
	}
	if compareVersions("1.2.4", "1.2.4") != 0 {
		t.Errorf("expected 1.2.4 == 1.2.4")
	}
	if compareVersions("1.0.36", "1.1.0") >= 0 {
		t.Errorf("expected 1.0.36 < 1.1.0")
	}
}
