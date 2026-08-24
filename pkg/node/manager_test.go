package node

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

func createMockTarGz(topDir string) []byte {
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

	// Add regular file
	fileContent := []byte("#!/bin/sh\necho v20.11.0\n")
	fileHeader := &tar.Header{
		Name:     topDir + "/bin/node",
		Mode:     0755,
		Size:     int64(len(fileContent)),
		Typeflag: tar.TypeReg,
	}
	_ = tw.WriteHeader(fileHeader)
	_, _ = tw.Write(fileContent)

	// Add symlink
	symlinkHeader := &tar.Header{
		Name:     topDir + "/bin/nodejs",
		Linkname: "node",
		Typeflag: tar.TypeSymlink,
	}
	_ = tw.WriteHeader(symlinkHeader)

	_ = tw.Close()
	_ = gw.Close()
	return buf.Bytes()
}

func createMockZip(topDir string) []byte {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	// Directory
	_, _ = zw.Create(topDir + "/bin/")

	// File
	fw, _ := zw.Create(topDir + "/node.exe")
	_, _ = fw.Write([]byte("fake node binary"))

	_ = zw.Close()
	return buf.Bytes()
}

func TestNewManager(t *testing.T) {
	m1 := NewManager("/tmp/test_uvm")
	if m1.BaseDir != "/tmp/test_uvm" {
		t.Errorf("expected BaseDir /tmp/test_uvm, got %s", m1.BaseDir)
	}

	m2 := NewManager("")
	if m2.BaseDir == "" {
		t.Errorf("expected default baseDir to not be empty")
	}

	// Test fallback when HOME is unset
	oldHome := os.Getenv("HOME")
	oldUserProf := os.Getenv("USERPROFILE")
	defer func() {
		os.Setenv("HOME", oldHome)
		os.Setenv("USERPROFILE", oldUserProf)
	}()

	os.Unsetenv("HOME")
	os.Setenv("USERPROFILE", "/custom/profile")
	_ = NewManager("")
}

func TestNormalizeVersion(t *testing.T) {
	m := NewManager("/tmp/test")
	tests := map[string]string{
		"20.11.0":  "v20.11.0",
		"v20.11.0": "v20.11.0",
		"latest":   "latest",
		"lts":      "lts",
		"current":  "current",
		"":         "",
		"  18.0.0 ": "v18.0.0",
	}

	for in, expected := range tests {
		if got := m.NormalizeVersion(in); got != expected {
			t.Errorf("NormalizeVersion(%q) = %q, expected %q", in, got, expected)
		}
	}
}

func TestResolveVersion(t *testing.T) {
	m := NewManager("/tmp/test")
	v, err := m.ResolveVersion("20.11.0")
	if err != nil || v != "v20.11.0" {
		t.Fatalf("unexpected result: %s, %v", v, err)
	}

	_, err = m.ResolveVersion("")
	if err == nil {
		t.Fatalf("expected error for empty version")
	}

	// Mock server for latest and lts
	mockReleases := []NodeRelease{
		{Version: "v22.2.0", LTS: false},
		{Version: "v20.11.0", LTS: "Iron"},
		{Version: "v18.20.0", LTS: true},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/index.json" {
			_ = json.NewEncoder(w).Encode(mockReleases)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	m.NodeDistURL = srv.URL
	m.HTTPClient = srv.Client()

	vLatest, err := m.ResolveVersion("latest")
	if err != nil || vLatest != "v22.2.0" {
		t.Errorf("expected v22.2.0 for latest, got %s (err: %v)", vLatest, err)
	}

	vCurrent, err := m.ResolveVersion("current")
	if err != nil || vCurrent != "v22.2.0" {
		t.Errorf("expected v22.2.0 for current, got %s (err: %v)", vCurrent, err)
	}

	vLts, err := m.ResolveVersion("lts")
	if err != nil || vLts != "v20.11.0" {
		t.Errorf("expected v20.11.0 for lts, got %s (err: %v)", vLts, err)
	}

	// Empty releases mock
	emptySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]NodeRelease{})
	}))
	defer emptySrv.Close()
	m.NodeDistURL = emptySrv.URL
	_, err = m.ResolveVersion("latest")
	if err == nil {
		t.Errorf("expected error for empty releases")
	}

	// Invalid JSON response
	badJsonSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer badJsonSrv.Close()
	m.NodeDistURL = badJsonSrv.URL
	_, err = m.ResolveVersion("latest")
	if err == nil {
		t.Errorf("expected error for invalid JSON")
	}

	// 500 error
	errSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer errSrv.Close()
	m.NodeDistURL = errSrv.URL
	_, err = m.ResolveVersion("latest")
	if err == nil {
		t.Errorf("expected error for 500 server response")
	}

	m.NodeDistURL = "http://127.0.0.1:1" // connection refused
	_, err = m.ResolveVersion("latest")
	if err == nil {
		t.Errorf("expected error for connection refused")
	}
}

func TestGetArchiveTarget(t *testing.T) {
	m := NewManager("/tmp/test")

	// Darwin
	m.GOOS = "darwin"
	m.GOARCH = "arm64"
	f1, isZip, err := m.GetArchiveTarget("v20.11.0")
	if err != nil || f1 != "node-v20.11.0-darwin-arm64.tar.gz" || isZip {
		t.Errorf("unexpected darwin arm64: %s, %v", f1, err)
	}

	m.GOARCH = "amd64"
	f2, isZip, err := m.GetArchiveTarget("v20.11.0")
	if err != nil || f2 != "node-v20.11.0-darwin-x64.tar.gz" || isZip {
		t.Errorf("unexpected darwin amd64: %s, %v", f2, err)
	}

	// Windows
	m.GOOS = "windows"
	m.GOARCH = "amd64"
	f3, isZip, err := m.GetArchiveTarget("v20.11.0")
	if err != nil || f3 != "node-v20.11.0-win-x64.zip" || !isZip {
		t.Errorf("unexpected windows amd64: %s, %v", f3, err)
	}

	m.GOARCH = "arm64"
	f4, isZip, err := m.GetArchiveTarget("v20.11.0")
	if err != nil || f4 != "node-v20.11.0-win-arm64.zip" || !isZip {
		t.Errorf("unexpected windows arm64: %s, %v", f4, err)
	}

	// Linux
	m.GOOS = "linux"
	m.GOARCH = "amd64"
	f5, _, err := m.GetArchiveTarget("v20.11.0")
	if err != nil || f5 != "node-v20.11.0-linux-x64.tar.gz" {
		t.Errorf("unexpected linux amd64: %s, %v", f5, err)
	}

	m.GOARCH = "arm64"
	f6, _, err := m.GetArchiveTarget("v20.11.0")
	if err != nil || f6 != "node-v20.11.0-linux-arm64.tar.gz" {
		t.Errorf("unexpected linux arm64: %s, %v", f6, err)
	}

	m.GOARCH = "arm"
	f7, _, err := m.GetArchiveTarget("v20.11.0")
	if err != nil || f7 != "node-v20.11.0-linux-armv7l.tar.gz" {
		t.Errorf("unexpected linux arm: %s, %v", f7, err)
	}

	// Unsupported OS
	m.GOOS = "plan9"
	_, _, err = m.GetArchiveTarget("v20.11.0")
	if err == nil {
		t.Errorf("expected error for unsupported OS")
	}
}

func TestInstallTarGz(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)
	m.GOOS = "darwin"
	m.GOARCH = "arm64"

	tarData := createMockTarGz("node-v20.11.0-darwin-arm64")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "v20.11.0") && strings.HasSuffix(r.URL.Path, ".tar.gz") {
			w.Header().Set("Content-Type", "application/gzip")
			_, _ = w.Write(tarData)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	m.NodeDistURL = srv.URL
	m.HTTPClient = srv.Client()

	outBuf := new(bytes.Buffer)
	err := m.Install("20.11.0", outBuf)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	if !strings.Contains(outBuf.String(), "installed successfully") {
		t.Errorf("unexpected install output: %s", outBuf.String())
	}

	// Re-install should detect already installed
	outBuf.Reset()
	err = m.Install("20.11.0", outBuf)
	if err != nil || !strings.Contains(outBuf.String(), "already installed") {
		t.Errorf("expected already installed output, got: %s, err: %v", outBuf.String(), err)
	}

	// Download failure test (404)
	err = m.Install("19.0.0", outBuf)
	if err == nil {
		t.Errorf("expected error on 404 download")
	}

	// Target error test with non-installed version
	m.GOOS = "unsupported_os"
	err = m.Install("21.0.0", outBuf)
	if err == nil {
		t.Errorf("expected error on unsupported OS install")
	}
}

func TestInstallZip(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)
	m.GOOS = "windows"
	m.GOARCH = "amd64"

	zipData := createMockZip("node-v20.11.0-win-x64")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".zip") {
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(zipData)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	m.NodeDistURL = srv.URL
	m.HTTPClient = srv.Client()

	outBuf := new(bytes.Buffer)
	err := m.Install("20.11.0", outBuf)
	if err != nil {
		t.Fatalf("Install zip failed: %v", err)
	}

	installedExe := filepath.Join(m.VersionsDir(), "v20.11.0", "node.exe")
	if _, err := os.Stat(installedExe); err != nil {
		t.Errorf("expected installed node.exe at %s", installedExe)
	}
}

func TestUseAndCurrent(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)
	m.GOOS = "darwin"
	m.GOARCH = "arm64"

	// Calling current with no versions installed
	_, err := m.Current()
	if err == nil {
		t.Errorf("expected error when no version is active")
	}

	// Calling use on uninstalled version
	outBuf := new(bytes.Buffer)
	err = m.Use("20.11.0", outBuf)
	if err == nil {
		t.Errorf("expected error when version is not installed")
	}

	// Mock installation
	versionDir := filepath.Join(m.VersionsDir(), "v20.11.0", "bin")
	_ = os.MkdirAll(versionDir, 0755)
	_ = os.WriteFile(filepath.Join(versionDir, "node"), []byte("#!/bin/sh\n"), 0755)
	_ = os.WriteFile(filepath.Join(versionDir, "npm"), []byte("#!/bin/sh\n"), 0755)

	outBuf.Reset()
	err = m.Use("20.11.0", outBuf)
	if err != nil {
		t.Fatalf("Use failed: %v", err)
	}

	cur, err := m.Current()
	if err != nil || cur != "v20.11.0" {
		t.Errorf("expected active version v20.11.0, got %s (err: %v)", cur, err)
	}

	// Test Use on Windows
	m.GOOS = "windows"
	outBuf.Reset()
	err = m.Use("20.11.0", outBuf)
	if err != nil {
		t.Fatalf("Use on Windows failed: %v", err)
	}
	shimPath := filepath.Join(m.BinDir(), "node.exe")
	if _, err := os.Stat(shimPath); err != nil {
		t.Errorf("expected windows node.exe shim at %s", shimPath)
	}
}

func TestListInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	// List when no directory exists
	list, err := m.ListInstalled()
	if err != nil || len(list) != 0 {
		t.Errorf("expected empty list, got: %+v, err: %v", list, err)
	}

	// Create mock versions and a non-directory file
	v1 := filepath.Join(m.VersionsDir(), "v18.20.0")
	v2 := filepath.Join(m.VersionsDir(), "v20.11.0")
	_ = os.MkdirAll(v1, 0755)
	_ = os.MkdirAll(v2, 0755)
	_ = os.WriteFile(filepath.Join(m.VersionsDir(), "ignore_file.txt"), []byte("data"), 0644)

	// Set v20.11.0 as active
	_ = m.Use("20.11.0", new(bytes.Buffer))

	list, err = m.ListInstalled()
	if err != nil || len(list) != 2 {
		t.Fatalf("expected 2 versions, got %d (err: %v)", len(list), err)
	}

	foundActive := false
	for _, v := range list {
		if v.Version == "v20.11.0" && v.IsActive {
			foundActive = true
		}
	}
	if !foundActive {
		t.Errorf("expected v20.11.0 to be marked active")
	}
}

func TestRemove(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	// Remove uninstalled version
	outBuf := new(bytes.Buffer)
	err := m.Remove("20.11.0", outBuf)
	if err == nil {
		t.Errorf("expected error removing non-existent version")
	}

	// Create and activate version
	versionDir := filepath.Join(m.VersionsDir(), "v20.11.0", "bin")
	_ = os.MkdirAll(versionDir, 0755)
	_ = m.Use("20.11.0", new(bytes.Buffer))

	// Remove active version
	outBuf.Reset()
	err = m.Remove("20.11.0", outBuf)
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	if !strings.Contains(outBuf.String(), "removed successfully") {
		t.Errorf("unexpected output: %s", outBuf.String())
	}

	// Current should now be empty/error
	_, err = m.Current()
	if err == nil {
		t.Errorf("expected error after removing active version")
	}

	// Test Remove on Windows
	m.GOOS = "windows"
	versionDirWin := filepath.Join(m.VersionsDir(), "v18.0.0")
	_ = os.MkdirAll(versionDirWin, 0755)
	_ = m.Use("18.0.0", new(bytes.Buffer))
	err = m.Remove("18.0.0", outBuf)
	if err != nil {
		t.Fatalf("Remove windows version failed: %v", err)
	}
}

func TestExtractionEdgeCases(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. extractTarGz with invalid gzip stream
	badTar := bytes.NewReader([]byte("not a gzip stream"))
	err := extractTarGz(badTar, tmpDir)
	if err == nil {
		t.Errorf("expected error for invalid gzip stream")
	}

	// 2. extractZip with invalid zip file
	badZipPath := filepath.Join(tmpDir, "corrupt.zip")
	_ = os.WriteFile(badZipPath, []byte("not a zip"), 0644)
	err = extractZip(badZipPath, tmpDir)
	if err == nil {
		t.Errorf("expected error for corrupt zip file")
	}
}

func TestStripTopDir(t *testing.T) {
	if res := stripTopDir("single"); res != "" {
		t.Errorf("expected empty for single, got %s", res)
	}
	if res := stripTopDir("node-v20/bin/node"); res != filepath.Join("bin", "node") {
		t.Errorf("expected bin/node, got %s", res)
	}
}
