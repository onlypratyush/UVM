package php

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createMockPhpTarGz(topDir string) []byte {
	buf := new(bytes.Buffer)
	gw := gzip.NewWriter(buf)
	tw := tar.NewWriter(gw)

	// Add directory
	dirHeader := &tar.Header{
		Name:     topDir + "/bin/",
		Mode:     0755,
		Typeflag: tar.TypeDir,
	}
	_ = tw.WriteHeader(dirHeader)

	// Add php binary
	phpContent := []byte("#!/bin/sh\necho PHP 8.3.17\n")
	phpHeader := &tar.Header{
		Name:     topDir + "/bin/php",
		Mode:     0755,
		Size:     int64(len(phpContent)),
		Typeflag: tar.TypeReg,
	}
	_ = tw.WriteHeader(phpHeader)
	_, _ = tw.Write(phpContent)

	// Add phar binary
	pharContent := []byte("#!/bin/sh\necho phar\n")
	pharHeader := &tar.Header{
		Name:     topDir + "/bin/phar",
		Mode:     0755,
		Size:     int64(len(pharContent)),
		Typeflag: tar.TypeReg,
	}
	_ = tw.WriteHeader(pharHeader)
	_, _ = tw.Write(pharContent)

	// Add symlink
	symlinkHeader := &tar.Header{
		Name:     topDir + "/bin/php-cgi",
		Linkname: "php",
		Typeflag: tar.TypeSymlink,
	}
	_ = tw.WriteHeader(symlinkHeader)

	_ = tw.Close()
	_ = gw.Close()
	return buf.Bytes()
}

func createMockPhpZip(topDir string) []byte {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	// Directory
	_, _ = zw.Create(topDir + "/")

	// Binaries
	fw, _ := zw.Create(topDir + "/php.exe")
	_, _ = fw.Write([]byte("fake php binary"))

	fw2, _ := zw.Create(topDir + "/phar.phar")
	_, _ = fw2.Write([]byte("fake phar binary"))

	_ = zw.Close()
	return buf.Bytes()
}

func TestNewManager(t *testing.T) {
	m1 := NewManager("/tmp/test_php_uvm")
	if m1.BaseDir != "/tmp/test_php_uvm" {
		t.Errorf("expected BaseDir /tmp/test_php_uvm, got %s", m1.BaseDir)
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
		{"8.3.17", "8.3.17"},
		{"v8.3.17", "8.3.17"},
		{"php8.3.17", "8.3.17"},
		{"php-8.4.4", "8.4.4"},
		{"8.3", "8.3"},
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

	if v, _ := m.ResolveRemoteVersion("latest"); v != "8.4.4" {
		t.Errorf("expected latest to resolve to 8.4.4, got %s", v)
	}
	if v, _ := m.ResolveRemoteVersion("lts"); v != "8.3.17" {
		t.Errorf("expected lts to resolve to 8.3.17, got %s", v)
	}
	if v, _ := m.ResolveRemoteVersion("stable"); v != "8.3.17" {
		t.Errorf("expected stable to resolve to 8.3.17, got %s", v)
	}
	if v, _ := m.ResolveRemoteVersion("8.3"); v != "8.3.17" {
		t.Errorf("expected 8.3 to resolve to 8.3.17, got %s", v)
	}
	if v, _ := m.ResolveRemoteVersion("8.2"); v != "8.2.28" {
		t.Errorf("expected 8.2 to resolve to 8.2.28, got %s", v)
	}
	if v, _ := m.ResolveRemoteVersion("8.3.10"); v != "8.3.10" {
		t.Errorf("expected 8.3.10 to stay 8.3.10, got %s", v)
	}
	if _, err := m.ResolveRemoteVersion(""); err == nil {
		t.Errorf("expected error for empty version")
	}

	alias, _ := m.ResolveVersion("8.3")
	if alias != "8.3.17" {
		t.Errorf("expected ResolveVersion alias 8.3.17, got %s", alias)
	}
}

func TestGetArchiveTarget(t *testing.T) {
	m := NewManager("")

	m.GOOS = "darwin"
	m.GOARCH = "arm64"
	url, file, isZip, err := m.GetArchiveTarget("8.3.17")
	if err != nil || isZip || !strings.Contains(file, "macos-aarch64") || !strings.Contains(url, "v8.3.17") {
		t.Errorf("unexpected darwin arm64 target: url=%s, file=%s, isZip=%v, err=%v", url, file, isZip, err)
	}

	m.GOOS = "darwin"
	m.GOARCH = "amd64"
	_, file, _, _ = m.GetArchiveTarget("8.3.17")
	if !strings.Contains(file, "macos-x86_64") {
		t.Errorf("expected x86_64 in darwin amd64 filename, got: %s", file)
	}

	m.GOOS = "linux"
	m.GOARCH = "amd64"
	_, file, _, _ = m.GetArchiveTarget("8.3.17")
	if !strings.Contains(file, "linux-x86_64") {
		t.Errorf("expected linux-x86_64 in linux filename, got: %s", file)
	}

	m.GOOS = "windows"
	m.GOARCH = "amd64"
	url, file, isZip, err = m.GetArchiveTarget("8.3.17")
	if err != nil || !isZip || !strings.Contains(file, "Win32-vs16-x64.zip") {
		t.Errorf("unexpected windows amd64 target: url=%s, file=%s, isZip=%v, err=%v", url, file, isZip, err)
	}

	m.GOOS = "unsupported_os"
	_, _, _, err = m.GetArchiveTarget("8.3.17")
	if err == nil {
		t.Errorf("expected error for unsupported OS")
	}
}

func TestInstallTarGz(t *testing.T) {
	mockArchive := createMockPhpTarGz("php-8.3.17")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(mockArchive)
	}))
	defer server.Close()

	tmpBase := t.TempDir()
	m := NewManager(tmpBase)
	m.PhpUnixDistURL = server.URL
	m.GOOS = "darwin"
	m.GOARCH = "arm64"

	outBuf := new(bytes.Buffer)
	err := m.Install("8.3.17", outBuf)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	outStr := outBuf.String()
	if !strings.Contains(outStr, "installed successfully") {
		t.Errorf("expected success output, got: %s", outStr)
	}

	// Verify installed files
	phpBin := filepath.Join(m.VersionsDir(), "8.3.17", "bin", "php")
	if _, err := os.Stat(phpBin); err != nil {
		t.Fatalf("expected binary at %s: %v", phpBin, err)
	}

	// Verify already installed fast-path
	outBuf2 := new(bytes.Buffer)
	err = m.Install("8.3.17", outBuf2)
	if err != nil || !strings.Contains(outBuf2.String(), "is already installed") {
		t.Errorf("expected already installed message, got: %s (err: %v)", outBuf2.String(), err)
	}
}

func TestInstallZip(t *testing.T) {
	mockZip := createMockPhpZip("php-8.3.17")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(mockZip)
	}))
	defer server.Close()

	tmpBase := t.TempDir()
	m := NewManager(tmpBase)
	m.PhpDistURL = server.URL
	m.GOOS = "windows"
	m.GOARCH = "amd64"

	outBuf := new(bytes.Buffer)
	err := m.Install("8.3.17", outBuf)
	if err != nil {
		t.Fatalf("Install zip failed: %v", err)
	}

	// Verify windows shims created
	shimExe := filepath.Join(m.BinDir(), "php.exe")
	if _, err := os.Stat(shimExe); err != nil {
		t.Errorf("expected %s to exist", shimExe)
	}
	shimCmd := filepath.Join(m.BinDir(), "php.cmd")
	if _, err := os.Stat(shimCmd); err != nil {
		t.Errorf("expected %s to exist", shimCmd)
	}
}

func TestUseAndCurrent(t *testing.T) {
	tmpBase := t.TempDir()
	m := NewManager(tmpBase)
	m.GOOS = "linux"
	m.GOARCH = "amd64"

	// Mock installed versions
	v1Dir := filepath.Join(m.VersionsDir(), "8.3.17", "bin")
	_ = os.MkdirAll(v1Dir, 0755)
	_ = os.WriteFile(filepath.Join(v1Dir, "php"), []byte("bin1"), 0755)

	v2Dir := filepath.Join(m.VersionsDir(), "8.4.4", "bin")
	_ = os.MkdirAll(v2Dir, 0755)
	_ = os.WriteFile(filepath.Join(v2Dir, "php"), []byte("bin2"), 0755)

	// Test Current before Use
	_, err := m.Current()
	if err == nil {
		t.Errorf("expected error when no version is selected")
	}

	// Use v1
	outBuf := new(bytes.Buffer)
	err = m.Use("8.3", outBuf)
	if err != nil {
		t.Fatalf("Use 8.3 failed: %v", err)
	}

	curr, err := m.Current()
	if err != nil || curr != "8.3.17" {
		t.Errorf("expected current version 8.3.17, got %s (err: %v)", curr, err)
	}

	// Verify Unix shim
	shim := filepath.Join(m.BinDir(), "php")
	if _, err := os.Lstat(shim); err != nil {
		t.Errorf("expected shim symlink at %s", shim)
	}

	// Use uninstalled
	err = m.Use("8.1", new(bytes.Buffer))
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
	_ = os.MkdirAll(filepath.Join(m.VersionsDir(), "8.3.17"), 0755)
	_ = os.MkdirAll(filepath.Join(m.VersionsDir(), "8.4.4"), 0755)

	_ = m.Use("8.3.17", new(bytes.Buffer))

	list, err = m.ListInstalled()
	if err != nil || len(list) != 2 {
		t.Fatalf("expected 2 installed versions, got %d (err: %v)", len(list), err)
	}

	foundActive := false
	for _, v := range list {
		if v.Version == "8.3.17" && v.IsActive {
			foundActive = true
		}
	}
	if !foundActive {
		t.Errorf("expected 8.3.17 to be marked as active")
	}
}

func TestRemove(t *testing.T) {
	tmpBase := t.TempDir()
	m := NewManager(tmpBase)
	m.GOOS = "linux"

	_ = os.MkdirAll(filepath.Join(m.VersionsDir(), "8.3.17", "bin"), 0755)
	_ = os.WriteFile(filepath.Join(m.VersionsDir(), "8.3.17", "bin", "php"), []byte("bin"), 0755)
	_ = m.Use("8.3.17", new(bytes.Buffer))

	outBuf := new(bytes.Buffer)
	err := m.Remove("8.3", outBuf)
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
	err = m.Remove("8.3.17", new(bytes.Buffer))
	if err == nil {
		t.Errorf("expected error removing non-installed version")
	}
}

func TestListRemote(t *testing.T) {
	mockIndex := map[string]interface{}{
		"8.4.4": map[string]interface{}{"date": "13 Feb 2025"},
		"8.3.17": map[string]interface{}{"date": "13 Feb 2025"},
		"8.2.28": map[string]interface{}{"date": "13 Feb 2025"},
	}
	data, _ := json.Marshal(mockIndex)

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

	_ = os.MkdirAll(filepath.Join(m.VersionsDir(), "8.3.17"), 0755)
	_ = os.MkdirAll(filepath.Join(m.VersionsDir(), "8.3.10"), 0755)
	_ = os.MkdirAll(filepath.Join(m.VersionsDir(), "8.4.4"), 0755)

	res, err := m.ResolveInstalledVersion("8.3")
	if err != nil || res != "8.3.17" {
		t.Errorf("expected 8.3 -> 8.3.17, got %s (err: %v)", res, err)
	}

	res, err = m.ResolveInstalledVersion("8.4")
	if err != nil || res != "8.4.4" {
		t.Errorf("expected 8.4 -> 8.4.4, got %s (err: %v)", res, err)
	}

	res, err = m.ResolveInstalledVersion("8.3.10")
	if err != nil || res != "8.3.10" {
		t.Errorf("expected exact 8.3.10, got %s (err: %v)", res, err)
	}
}

func TestCompareVersions(t *testing.T) {
	if compareVersions("8.4.4", "8.3.17") <= 0 {
		t.Errorf("expected 8.4.4 > 8.3.17")
	}
	if compareVersions("8.3.17", "8.3.17") != 0 {
		t.Errorf("expected 8.3.17 == 8.3.17")
	}
	if compareVersions("7.4.33", "8.0.0") >= 0 {
		t.Errorf("expected 7.4.33 < 8.0.0")
	}
}
