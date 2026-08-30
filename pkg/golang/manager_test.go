package golang

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

func createMockGoTarGz(topDir string) []byte {
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

	// Add go binary
	goContent := []byte("#!/bin/sh\necho go version go1.22.0 darwin/arm64\n")
	goHeader := &tar.Header{
		Name:     topDir + "/bin/go",
		Mode:     0755,
		Size:     int64(len(goContent)),
		Typeflag: tar.TypeReg,
	}
	_ = tw.WriteHeader(goHeader)
	_, _ = tw.Write(goContent)

	// Add gofmt binary
	gofmtContent := []byte("#!/bin/sh\necho gofmt\n")
	gofmtHeader := &tar.Header{
		Name:     topDir + "/bin/gofmt",
		Mode:     0755,
		Size:     int64(len(gofmtContent)),
		Typeflag: tar.TypeReg,
	}
	_ = tw.WriteHeader(gofmtHeader)
	_, _ = tw.Write(gofmtContent)

	// Add symlink
	symlinkHeader := &tar.Header{
		Name:     topDir + "/bin/go-symlink",
		Linkname: "go",
		Typeflag: tar.TypeSymlink,
	}
	_ = tw.WriteHeader(symlinkHeader)

	_ = tw.Close()
	_ = gw.Close()
	return buf.Bytes()
}

func createMockGoZip(topDir string) []byte {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	// Directory
	_, _ = zw.Create(topDir + "/bin/")

	// Binaries
	fw, _ := zw.Create(topDir + "/bin/go.exe")
	_, _ = fw.Write([]byte("fake go binary"))

	fw2, _ := zw.Create(topDir + "/bin/gofmt.exe")
	_, _ = fw2.Write([]byte("fake gofmt binary"))

	_ = zw.Close()
	return buf.Bytes()
}

func TestNewManager(t *testing.T) {
	m1 := NewManager("/tmp/test_go_uvm")
	if m1.BaseDir != "/tmp/test_go_uvm" {
		t.Errorf("expected BaseDir /tmp/test_go_uvm, got %s", m1.BaseDir)
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
		"1.22.0":   "go1.22.0",
		"go1.22.0": "go1.22.0",
		"latest":   "latest",
		"stable":   "stable",
		"current":  "current",
		"lts":      "lts",
		"":         "",
		" 1.21.5 ": "go1.21.5",
	}

	for in, expected := range tests {
		if got := m.NormalizeVersion(in); got != expected {
			t.Errorf("NormalizeVersion(%q) = %q, expected %q", in, got, expected)
		}
	}
}

func TestResolveVersion(t *testing.T) {
	m := NewManager("/tmp/test")
	v, err := m.ResolveVersion("1.22.0")
	if err != nil || v != "go1.22.0" {
		t.Fatalf("unexpected result: %s, %v", v, err)
	}

	_, err = m.ResolveVersion("")
	if err == nil {
		t.Fatalf("expected error for empty version")
	}

	mockReleases := []GoRelease{
		{
			Version: "go1.23.0",
			Stable:  true,
			Files: []GoFileEntry{
				{Filename: "go1.23.0.darwin-arm64.tar.gz", OS: "darwin", Arch: "arm64", Version: "go1.23.0", Kind: "archive"},
			},
		},
		{
			Version: "go1.22.6",
			Stable:  true,
			Files: []GoFileEntry{
				{Filename: "go1.22.6.darwin-arm64.tar.gz", OS: "darwin", Arch: "arm64", Version: "go1.22.6", Kind: "archive"},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.RawQuery, "mode=json") {
			_ = json.NewEncoder(w).Encode(mockReleases)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	m.GoDistURL = srv.URL
	m.HTTPClient = srv.Client()

	vLatest, err := m.ResolveVersion("latest")
	if err != nil || vLatest != "go1.23.0" {
		t.Errorf("expected go1.23.0 for latest, got %s (err: %v)", vLatest, err)
	}

	vStable, err := m.ResolveVersion("stable")
	if err != nil || vStable != "go1.23.0" {
		t.Errorf("expected go1.23.0 for stable, got %s (err: %v)", vStable, err)
	}

	// Empty releases mock
	emptySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]GoRelease{})
	}))
	defer emptySrv.Close()
	m.GoDistURL = emptySrv.URL
	_, err = m.ResolveVersion("latest")
	if err == nil {
		t.Errorf("expected error for empty releases")
	}

	// Bad JSON
	badJsonSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer badJsonSrv.Close()
	m.GoDistURL = badJsonSrv.URL
	_, err = m.ResolveVersion("latest")
	if err == nil {
		t.Errorf("expected error for bad json")
	}

	// HTTP error
	errSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "error", http.StatusInternalServerError)
	}))
	defer errSrv.Close()
	m.GoDistURL = errSrv.URL
	_, err = m.ResolveVersion("latest")
	if err == nil {
		t.Errorf("expected error for 500 status")
	}

	m.GoDistURL = "http://127.0.0.1:1"
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
	f1, isZip, err := m.GetArchiveTarget("go1.22.0")
	if err != nil || f1 != "go1.22.0.darwin-arm64.tar.gz" || isZip {
		t.Errorf("unexpected darwin arm64: %s, %v", f1, err)
	}

	m.GOARCH = "amd64"
	f2, isZip, err := m.GetArchiveTarget("go1.22.0")
	if err != nil || f2 != "go1.22.0.darwin-amd64.tar.gz" || isZip {
		t.Errorf("unexpected darwin amd64: %s, %v", f2, err)
	}

	// Windows
	m.GOOS = "windows"
	m.GOARCH = "amd64"
	f3, isZip, err := m.GetArchiveTarget("go1.22.0")
	if err != nil || f3 != "go1.22.0.windows-amd64.zip" || !isZip {
		t.Errorf("unexpected windows amd64: %s, %v", f3, err)
	}

	m.GOARCH = "arm64"
	f4, isZip, err := m.GetArchiveTarget("go1.22.0")
	if err != nil || f4 != "go1.22.0.windows-arm64.zip" || !isZip {
		t.Errorf("unexpected windows arm64: %s, %v", f4, err)
	}

	// Linux
	m.GOOS = "linux"
	m.GOARCH = "amd64"
	f5, _, err := m.GetArchiveTarget("go1.22.0")
	if err != nil || f5 != "go1.22.0.linux-amd64.tar.gz" {
		t.Errorf("unexpected linux amd64: %s, %v", f5, err)
	}

	m.GOARCH = "arm64"
	f6, _, err := m.GetArchiveTarget("go1.22.0")
	if err != nil || f6 != "go1.22.0.linux-arm64.tar.gz" {
		t.Errorf("unexpected linux arm64: %s, %v", f6, err)
	}

	m.GOARCH = "arm"
	f7, _, err := m.GetArchiveTarget("go1.22.0")
	if err != nil || f7 != "go1.22.0.linux-armv6l.tar.gz" {
		t.Errorf("unexpected linux arm: %s, %v", f7, err)
	}

	// Unsupported OS
	m.GOOS = "unsupported_os"
	_, _, err = m.GetArchiveTarget("go1.22.0")
	if err == nil {
		t.Errorf("expected error for unsupported OS")
	}
}

func TestInstallTarGz(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)
	m.GOOS = "darwin"
	m.GOARCH = "arm64"

	tarData := createMockGoTarGz("go")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "go1.22.0") && strings.HasSuffix(r.URL.Path, ".tar.gz") {
			w.Header().Set("Content-Type", "application/gzip")
			_, _ = w.Write(tarData)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	m.GoDistURL = srv.URL
	m.HTTPClient = srv.Client()

	outBuf := new(bytes.Buffer)
	err := m.Install("1.22.0", outBuf)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	if !strings.Contains(outBuf.String(), "installed successfully") {
		t.Errorf("unexpected install output: %s", outBuf.String())
	}

	// Re-install should detect already installed
	outBuf.Reset()
	err = m.Install("1.22.0", outBuf)
	if err != nil || !strings.Contains(outBuf.String(), "already installed") {
		t.Errorf("expected already installed output, got: %s, err: %v", outBuf.String(), err)
	}

	// 404 download error
	err = m.Install("1.19.0", outBuf)
	if err == nil {
		t.Errorf("expected error on 404 download")
	}

	// Target error test with unsupported OS
	m.GOOS = "unsupported"
	err = m.Install("1.21.0", outBuf)
	if err == nil {
		t.Errorf("expected error on unsupported OS")
	}
}

func TestInstallZip(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)
	m.GOOS = "windows"
	m.GOARCH = "amd64"

	zipData := createMockGoZip("go")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".zip") {
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(zipData)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	m.GoDistURL = srv.URL
	m.HTTPClient = srv.Client()

	outBuf := new(bytes.Buffer)
	err := m.Install("1.22.0", outBuf)
	if err != nil {
		t.Fatalf("Install zip failed: %v", err)
	}

	installedExe := filepath.Join(m.VersionsDir(), "go1.22.0", "bin", "go.exe")
	if _, err := os.Stat(installedExe); err != nil {
		t.Errorf("expected installed go.exe at %s", installedExe)
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
	err = m.Use("1.22.0", outBuf)
	if err == nil {
		t.Errorf("expected error when version is not installed")
	}

	// Mock installation
	versionDir := filepath.Join(m.VersionsDir(), "go1.22.0", "bin")
	_ = os.MkdirAll(versionDir, 0755)
	_ = os.WriteFile(filepath.Join(versionDir, "go"), []byte("#!/bin/sh\n"), 0755)
	_ = os.WriteFile(filepath.Join(versionDir, "gofmt"), []byte("#!/bin/sh\n"), 0755)

	outBuf.Reset()
	err = m.Use("1.22.0", outBuf)
	if err != nil {
		t.Fatalf("Use failed: %v", err)
	}

	cur, err := m.Current()
	if err != nil || cur != "go1.22.0" {
		t.Errorf("expected active version go1.22.0, got %s (err: %v)", cur, err)
	}

	// Test Use on Windows
	m.GOOS = "windows"
	_ = os.WriteFile(filepath.Join(versionDir, "go.exe"), []byte("exe"), 0755)
	_ = os.WriteFile(filepath.Join(versionDir, "gofmt.exe"), []byte("exe"), 0755)
	outBuf.Reset()
	err = m.Use("1.22.0", outBuf)
	if err != nil {
		t.Fatalf("Use on Windows failed: %v", err)
	}
	shimPath := filepath.Join(m.BinDir(), "go.exe")
	if _, err := os.Stat(shimPath); err != nil {
		t.Errorf("expected windows go.exe shim at %s", shimPath)
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
	v1 := filepath.Join(m.VersionsDir(), "go1.21.0")
	v2 := filepath.Join(m.VersionsDir(), "go1.22.0")
	_ = os.MkdirAll(v1, 0755)
	_ = os.MkdirAll(v2, 0755)
	_ = os.WriteFile(filepath.Join(m.VersionsDir(), "ignore.txt"), []byte("test"), 0644)

	// Set go1.22.0 as active
	_ = m.Use("1.22.0", new(bytes.Buffer))

	list, err = m.ListInstalled()
	if err != nil || len(list) != 2 {
		t.Fatalf("expected 2 versions, got %d (err: %v)", len(list), err)
	}

	foundActive := false
	for _, v := range list {
		if v.Version == "go1.22.0" && v.IsActive {
			foundActive = true
		}
	}
	if !foundActive {
		t.Errorf("expected go1.22.0 to be marked active")
	}
}

func TestRemove(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	// Remove uninstalled version
	outBuf := new(bytes.Buffer)
	err := m.Remove("1.22.0", outBuf)
	if err == nil {
		t.Errorf("expected error removing non-existent version")
	}

	// Create and activate version
	versionDir := filepath.Join(m.VersionsDir(), "go1.22.0", "bin")
	_ = os.MkdirAll(versionDir, 0755)
	_ = m.Use("1.22.0", new(bytes.Buffer))

	// Remove active version
	outBuf.Reset()
	err = m.Remove("1.22.0", outBuf)
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
	versionDirWin := filepath.Join(m.VersionsDir(), "go1.21.0", "bin")
	_ = os.MkdirAll(versionDirWin, 0755)
	_ = m.Use("1.21.0", new(bytes.Buffer))
	err = m.Remove("1.21.0", outBuf)
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
	mockReleases := []GoRelease{
		{Version: "go1.24.0", Stable: true},
		{Version: "go1.23.6", Stable: true},
		{Version: "go1.22.12", Stable: true},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(mockReleases)
	}))
	defer srv.Close()

	m.GoDistURL = srv.URL
	m.HTTPClient = srv.Client()

	list, err := m.ListRemote(2)
	if err != nil || len(list) != 2 {
		t.Fatalf("expected 2 remote releases, got %d (err: %v)", len(list), err)
	}
	if list[0].Version != "go1.24.0" {
		t.Errorf("expected go1.24.0, got %s", list[0].Version)
	}
}

func TestResolveInstalledVersionPartial(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	// Create installed versions
	_ = os.MkdirAll(filepath.Join(m.VersionsDir(), "go1.22.0"), 0755)
	_ = os.MkdirAll(filepath.Join(m.VersionsDir(), "go1.22.6"), 0755)
	_ = os.MkdirAll(filepath.Join(m.VersionsDir(), "go1.21.0"), 0755)

	res, err := m.ResolveInstalledVersion("1.22")
	if err != nil || res != "go1.22.6" {
		t.Errorf("expected go1.22.6 for 1.22 prefix, got %s (err: %v)", res, err)
	}

	res, err = m.ResolveInstalledVersion("go1.22")
	if err != nil || res != "go1.22.6" {
		t.Errorf("expected go1.22.6 for go1.22 prefix, got %s (err: %v)", res, err)
	}

	res, err = m.ResolveInstalledVersion("1.21")
	if err != nil || res != "go1.21.0" {
		t.Errorf("expected go1.21.0 for 1.21 prefix, got %s (err: %v)", res, err)
	}
}

func TestResolveRemoteVersionPartial(t *testing.T) {
	m := NewManager("/tmp/test")
	mockReleases := []GoRelease{
		{Version: "go1.24.0", Stable: true},
		{Version: "go1.22.12", Stable: true},
		{Version: "go1.22.6", Stable: true},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(mockReleases)
	}))
	defer srv.Close()

	m.GoDistURL = srv.URL
	m.HTTPClient = srv.Client()

	v, err := m.ResolveRemoteVersion("1.22")
	if err != nil || v != "go1.22.12" {
		t.Errorf("expected go1.22.12 for 1.22 remote resolve, got %s (err: %v)", v, err)
	}
}

func TestCompareVersions(t *testing.T) {
	if compareVersions("1.22.6", "1.22.0") <= 0 {
		t.Errorf("expected 1.22.6 > 1.22.0")
	}
	if compareVersions("1.21.0", "1.22.0") >= 0 {
		t.Errorf("expected 1.21.0 < 1.22.0")
	}
	if compareVersions("go1.22.0", "1.22.0") != 0 {
		t.Errorf("expected go1.22.0 == 1.22.0")
	}
	if compareVersions("1.22", "1.22.0") != 0 {
		t.Errorf("expected 1.22 == 1.22.0")
	}
}


