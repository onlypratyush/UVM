package python

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

func createMockPythonTarGz(topDir string) []byte {
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

	// Add python binary
	pyContent := []byte("#!/bin/sh\necho Python 3.12.2\n")
	pyHeader := &tar.Header{
		Name:     topDir + "/bin/python3",
		Mode:     0755,
		Size:     int64(len(pyContent)),
		Typeflag: tar.TypeReg,
	}
	_ = tw.WriteHeader(pyHeader)
	_, _ = tw.Write(pyContent)

	// Add pip binary
	pipContent := []byte("#!/bin/sh\necho pip 24.0\n")
	pipHeader := &tar.Header{
		Name:     topDir + "/bin/pip3",
		Mode:     0755,
		Size:     int64(len(pipContent)),
		Typeflag: tar.TypeReg,
	}
	_ = tw.WriteHeader(pipHeader)
	_, _ = tw.Write(pipContent)

	// Add symlink
	symlinkHeader := &tar.Header{
		Name:     topDir + "/bin/python",
		Linkname: "python3",
		Typeflag: tar.TypeSymlink,
	}
	_ = tw.WriteHeader(symlinkHeader)

	_ = tw.Close()
	_ = gw.Close()
	return buf.Bytes()
}

func createMockPythonZip(topDir string) []byte {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	// Directory
	_, _ = zw.Create(topDir + "/bin/")
	_, _ = zw.Create(topDir + "/Scripts/")

	// Binaries
	fw, _ := zw.Create(topDir + "/python.exe")
	_, _ = fw.Write([]byte("fake python binary"))

	fw2, _ := zw.Create(topDir + "/Scripts/pip.exe")
	_, _ = fw2.Write([]byte("fake pip binary"))

	_ = zw.Close()
	return buf.Bytes()
}

func TestNewManager(t *testing.T) {
	m1 := NewManager("/tmp/test_py_uvm")
	if m1.BaseDir != "/tmp/test_py_uvm" {
		t.Errorf("expected BaseDir /tmp/test_py_uvm, got %s", m1.BaseDir)
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

	os.Unsetenv("HOME")
	os.Setenv("USERPROFILE", "/custom/profile")
	_ = NewManager("")
}

func TestNormalizeVersion(t *testing.T) {
	m := NewManager("/tmp/test")
	tests := map[string]string{
		"3.12.2":       "3.12.2",
		"v3.12.2":      "3.12.2",
		"python3.12.2": "3.12.2",
		"py3.12.2":     "3.12.2",
		"latest":       "latest",
		"lts":          "lts",
		"current":      "current",
		"":             "",
		"  3.11.0  ":   "3.11.0",
	}

	for in, expected := range tests {
		if got := m.NormalizeVersion(in); got != expected {
			t.Errorf("NormalizeVersion(%q) = %q, expected %q", in, got, expected)
		}
	}
}

func TestResolveVersion(t *testing.T) {
	m := NewManager("/tmp/test")
	v, err := m.ResolveVersion("3.12.2")
	if err != nil || v != "3.12.2" {
		t.Fatalf("unexpected result: %s, %v", v, err)
	}

	_, err = m.ResolveVersion("")
	if err == nil {
		t.Fatalf("expected error for empty version")
	}

	vLatest, err := m.ResolveVersion("latest")
	if err != nil || vLatest != "3.13.2" {
		t.Errorf("expected 3.13.2 for latest, got %s (err: %v)", vLatest, err)
	}

	vCurrent, err := m.ResolveVersion("current")
	if err != nil || vCurrent != "3.13.2" {
		t.Errorf("expected 3.13.2 for current, got %s (err: %v)", vCurrent, err)
	}

	vLts, err := m.ResolveVersion("lts")
	if err != nil || vLts != "3.12.9" {
		t.Errorf("expected 3.12.9 for lts, got %s (err: %v)", vLts, err)
	}
}

func TestFetchLatestReleaseTag(t *testing.T) {
	m := NewManager("/tmp/test")

	// 1. Success mock
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		meta := PythonReleaseMetadata{
			Version: 1,
			Tag:     "20260825",
		}
		_ = json.NewEncoder(w).Encode(meta)
	}))
	defer srv.Close()

	m.MetadataURL = srv.URL
	m.HTTPClient = srv.Client()

	tag := m.FetchLatestReleaseTag()
	if tag != "20260825" {
		t.Errorf("expected tag 20260825, got %s", tag)
	}

	// 2. Empty URL fallback
	m.MetadataURL = ""
	if got := m.FetchLatestReleaseTag(); got != DefaultReleaseTag {
		t.Errorf("expected fallback tag %s, got %s", DefaultReleaseTag, got)
	}

	// 3. Error fallback
	m.MetadataURL = "http://127.0.0.1:1"
	if got := m.FetchLatestReleaseTag(); got != DefaultReleaseTag {
		t.Errorf("expected fallback tag %s on connection error, got %s", DefaultReleaseTag, got)
	}
}

func TestGetArchiveTarget(t *testing.T) {
	m := NewManager("/tmp/test")

	// Darwin
	m.GOOS = "darwin"
	m.GOARCH = "arm64"
	f1, isZip, err := m.GetArchiveTarget("3.12.2", "20241016")
	if err != nil || f1 != "cpython-3.12.2+20241016-aarch64-apple-darwin-install_only.tar.gz" || isZip {
		t.Errorf("unexpected darwin arm64: %s, %v", f1, err)
	}

	m.GOARCH = "amd64"
	f2, isZip, err := m.GetArchiveTarget("3.12.2", "20241016")
	if err != nil || f2 != "cpython-3.12.2+20241016-x86_64-apple-darwin-install_only.tar.gz" || isZip {
		t.Errorf("unexpected darwin amd64: %s, %v", f2, err)
	}

	// Windows
	m.GOOS = "windows"
	m.GOARCH = "amd64"
	f3, isZip, err := m.GetArchiveTarget("3.12.2", "20241016")
	if err != nil || f3 != "cpython-3.12.2+20241016-x86_64-pc-windows-msvc-shared-install_only.tar.gz" || isZip {
		t.Errorf("unexpected windows amd64: %s, %v", f3, err)
	}

	m.GOARCH = "arm64"
	f4, isZip, err := m.GetArchiveTarget("3.12.2", "20241016")
	if err != nil || f4 != "cpython-3.12.2+20241016-aarch64-pc-windows-msvc-shared-install_only.tar.gz" || isZip {
		t.Errorf("unexpected windows arm64: %s, %v", f4, err)
	}

	// Linux
	m.GOOS = "linux"
	m.GOARCH = "amd64"
	f5, _, err := m.GetArchiveTarget("3.12.2", "")
	if err != nil || !strings.Contains(f5, "x86_64-unknown-linux-gnu") {
		t.Errorf("unexpected linux amd64: %s, %v", f5, err)
	}

	m.GOARCH = "arm64"
	f6, _, err := m.GetArchiveTarget("3.12.2", "20241016")
	if err != nil || f6 != "cpython-3.12.2+20241016-aarch64-unknown-linux-gnu-install_only.tar.gz" {
		t.Errorf("unexpected linux arm64: %s, %v", f6, err)
	}

	m.GOARCH = "arm"
	f7, _, err := m.GetArchiveTarget("3.12.2", "20241016")
	if err != nil || f7 != "cpython-3.12.2+20241016-armv7-unknown-linux-gnu-install_only.tar.gz" {
		t.Errorf("unexpected linux arm: %s, %v", f7, err)
	}

	// Unsupported OS
	m.GOOS = "unsupported_os"
	_, _, err = m.GetArchiveTarget("3.12.2", "20241016")
	if err == nil {
		t.Errorf("expected error for unsupported OS")
	}
}

func TestInstallTarGz(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)
	m.GOOS = "darwin"
	m.GOARCH = "arm64"

	tarData := createMockPythonTarGz("python")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "3.12.2") && strings.HasSuffix(r.URL.Path, ".tar.gz") {
			w.Header().Set("Content-Type", "application/gzip")
			_, _ = w.Write(tarData)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	m.PythonDistURL = srv.URL
	m.MetadataURL = ""
	m.HTTPClient = srv.Client()

	outBuf := new(bytes.Buffer)
	err := m.Install("3.12.2", outBuf)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	if !strings.Contains(outBuf.String(), "installed successfully") {
		t.Errorf("unexpected install output: %s", outBuf.String())
	}

	// Re-install should detect already installed
	outBuf.Reset()
	err = m.Install("3.12.2", outBuf)
	if err != nil || !strings.Contains(outBuf.String(), "already installed") {
		t.Errorf("expected already installed output, got: %s, err: %v", outBuf.String(), err)
	}

	// 404 download error
	err = m.Install("3.9.0", outBuf)
	if err == nil {
		t.Errorf("expected error on 404 download")
	}

	// Target error test with unsupported OS
	m.GOOS = "unsupported"
	err = m.Install("3.11.0", outBuf)
	if err == nil {
		t.Errorf("expected error on unsupported OS")
	}
}

func TestInstallZip(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)
	m.GOOS = "windows"
	m.GOARCH = "amd64"

	zipData := createMockPythonZip("python")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".zip") {
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(zipData)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	m.PythonDistURL = srv.URL
	m.MetadataURL = ""
	m.HTTPClient = srv.Client()

	// Mock GetArchiveTarget return zip
	m.GOOS = "darwin" // to pass target logic, but test extractZip directly
	zipPath := filepath.Join(tmpDir, "test.zip")
	_ = os.WriteFile(zipPath, zipData, 0644)
	destDir := filepath.Join(tmpDir, "extracted")
	err := extractZip(zipPath, destDir)
	if err != nil {
		t.Fatalf("extractZip failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(destDir, "python.exe")); err != nil {
		t.Errorf("expected extracted python.exe")
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
	err = m.Use("3.12.2", outBuf)
	if err == nil {
		t.Errorf("expected error when version is not installed")
	}

	// Mock installation
	versionDir := filepath.Join(m.VersionsDir(), "3.12.2", "bin")
	_ = os.MkdirAll(versionDir, 0755)
	_ = os.WriteFile(filepath.Join(versionDir, "python3"), []byte("#!/bin/sh\n"), 0755)
	_ = os.WriteFile(filepath.Join(versionDir, "pip3"), []byte("#!/bin/sh\n"), 0755)

	outBuf.Reset()
	err = m.Use("3.12.2", outBuf)
	if err != nil {
		t.Fatalf("Use failed: %v", err)
	}

	cur, err := m.Current()
	if err != nil || cur != "3.12.2" {
		t.Errorf("expected active version 3.12.2, got %s (err: %v)", cur, err)
	}

	// Test Use on Windows
	m.GOOS = "windows"
	_ = os.WriteFile(filepath.Join(m.VersionsDir(), "3.12.2", "python.exe"), []byte("exe"), 0755)
	scriptsDir := filepath.Join(m.VersionsDir(), "3.12.2", "Scripts")
	_ = os.MkdirAll(scriptsDir, 0755)
	_ = os.WriteFile(filepath.Join(scriptsDir, "pip.exe"), []byte("exe"), 0755)

	outBuf.Reset()
	err = m.Use("3.12.2", outBuf)
	if err != nil {
		t.Fatalf("Use on Windows failed: %v", err)
	}
	shimPath := filepath.Join(m.BinDir(), "python.exe")
	if _, err := os.Stat(shimPath); err != nil {
		t.Errorf("expected windows python.exe shim at %s", shimPath)
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

	// Create mock versions
	v1 := filepath.Join(m.VersionsDir(), "3.11.8")
	v2 := filepath.Join(m.VersionsDir(), "3.12.2")
	_ = os.MkdirAll(v1, 0755)
	_ = os.MkdirAll(v2, 0755)
	_ = os.WriteFile(filepath.Join(m.VersionsDir(), "ignore.txt"), []byte("test"), 0644)

	// Set 3.12.2 as active
	_ = m.Use("3.12.2", new(bytes.Buffer))

	list, err = m.ListInstalled()
	if err != nil || len(list) != 2 {
		t.Fatalf("expected 2 versions, got %d (err: %v)", len(list), err)
	}

	foundActive := false
	for _, v := range list {
		if v.Version == "3.12.2" && v.IsActive {
			foundActive = true
		}
	}
	if !foundActive {
		t.Errorf("expected 3.12.2 to be marked active")
	}
}

func TestRemove(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	// Remove uninstalled version
	outBuf := new(bytes.Buffer)
	err := m.Remove("3.12.2", outBuf)
	if err == nil {
		t.Errorf("expected error removing non-existent version")
	}

	// Create and activate version
	versionDir := filepath.Join(m.VersionsDir(), "3.12.2", "bin")
	_ = os.MkdirAll(versionDir, 0755)
	_ = m.Use("3.12.2", new(bytes.Buffer))

	// Remove active version
	outBuf.Reset()
	err = m.Remove("3.12.2", outBuf)
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

	// Remove on Windows
	m.GOOS = "windows"
	versionDirWin := filepath.Join(m.VersionsDir(), "3.11.0", "bin")
	_ = os.MkdirAll(versionDirWin, 0755)
	_ = m.Use("3.11.0", new(bytes.Buffer))
	err = m.Remove("3.11.0", outBuf)
	if err != nil {
		t.Fatalf("Remove windows version failed: %v", err)
	}
}

func TestExtractionEdgeCases(t *testing.T) {
	tmpDir := t.TempDir()

	badTar := bytes.NewReader([]byte("not a gzip stream"))
	err := extractTarGz(badTar, tmpDir)
	if err == nil {
		t.Errorf("expected error for invalid gzip stream")
	}

	badZipPath := filepath.Join(tmpDir, "corrupt.zip")
	_ = os.WriteFile(badZipPath, []byte("not a zip"), 0644)
	err = extractZip(badZipPath, tmpDir)
	if err == nil {
		t.Errorf("expected error for corrupt zip file")
	}
}

func TestListRemote(t *testing.T) {
	m := NewManager("/tmp/test")
	list, err := m.ListRemote(3)
	if err != nil || len(list) != 3 {
		t.Fatalf("expected 3 remote releases, got %d (err: %v)", len(list), err)
	}
	if list[0].Version != "3.13.2" {
		t.Errorf("expected 3.13.2, got %s", list[0].Version)
	}
}

func TestResolveInstalledVersionPartial(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	// Create installed versions
	_ = os.MkdirAll(filepath.Join(m.VersionsDir(), "3.12.2"), 0755)
	_ = os.MkdirAll(filepath.Join(m.VersionsDir(), "3.12.9"), 0755)
	_ = os.MkdirAll(filepath.Join(m.VersionsDir(), "3.11.0"), 0755)

	res, err := m.ResolveInstalledVersion("3.12")
	if err != nil || res != "3.12.9" {
		t.Errorf("expected 3.12.9 for 3.12 prefix, got %s (err: %v)", res, err)
	}

	res, err = m.ResolveInstalledVersion("py3.12")
	if err != nil || res != "3.12.9" {
		t.Errorf("expected 3.12.9 for py3.12 prefix, got %s (err: %v)", res, err)
	}

	res, err = m.ResolveInstalledVersion("3.11")
	if err != nil || res != "3.11.0" {
		t.Errorf("expected 3.11.0 for 3.11 prefix, got %s (err: %v)", res, err)
	}
}

func TestResolveRemoteVersionPartial(t *testing.T) {
	m := NewManager("/tmp/test")

	v, err := m.ResolveRemoteVersion("3.12")
	if err != nil || v != "3.12.9" {
		t.Errorf("expected 3.12.9 for 3.12 remote resolve, got %s (err: %v)", v, err)
	}

	v, err = m.ResolveRemoteVersion("3.11")
	if err != nil || v != "3.11.11" {
		t.Errorf("expected 3.11.11 for 3.11 remote resolve, got %s (err: %v)", v, err)
	}
}

func TestCompareVersions(t *testing.T) {
	if compareVersions("3.12.9", "3.12.2") <= 0 {
		t.Errorf("expected 3.12.9 > 3.12.2")
	}
	if compareVersions("3.11.0", "3.12.0") >= 0 {
		t.Errorf("expected 3.11.0 < 3.12.0")
	}
	if compareVersions("3.12.0", "3.12") != 0 {
		t.Errorf("expected 3.12.0 == 3.12")
	}
}


