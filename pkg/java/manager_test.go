package java

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

func createMockJavaTarGz(topDir string, isMacBundle bool) []byte {
	buf := new(bytes.Buffer)
	gw := gzip.NewWriter(buf)
	tw := tar.NewWriter(gw)

	binPrefix := topDir + "/bin/"
	if isMacBundle {
		binPrefix = topDir + "/Contents/Home/bin/"
	}

	// Add directory
	dirHeader := &tar.Header{
		Name:     binPrefix,
		Mode:     0755,
		Typeflag: tar.TypeDir,
	}
	_ = tw.WriteHeader(dirHeader)

	// Add java binary
	javaContent := []byte("#!/bin/sh\necho openjdk version 21.0.6\n")
	javaHeader := &tar.Header{
		Name:     binPrefix + "java",
		Mode:     0755,
		Size:     int64(len(javaContent)),
		Typeflag: tar.TypeReg,
	}
	_ = tw.WriteHeader(javaHeader)
	_, _ = tw.Write(javaContent)

	// Add javac binary
	javacContent := []byte("#!/bin/sh\necho javac 21.0.6\n")
	javacHeader := &tar.Header{
		Name:     binPrefix + "javac",
		Mode:     0755,
		Size:     int64(len(javacContent)),
		Typeflag: tar.TypeReg,
	}
	_ = tw.WriteHeader(javacHeader)
	_, _ = tw.Write(javacContent)

	_ = tw.Close()
	_ = gw.Close()
	return buf.Bytes()
}

func createMockJavaZip(topDir string) []byte {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	// Directory
	_, _ = zw.Create(topDir + "/bin/")

	// Binaries
	fw, _ := zw.Create(topDir + "/bin/java.exe")
	_, _ = fw.Write([]byte("fake java binary"))

	fw2, _ := zw.Create(topDir + "/bin/javac.exe")
	_, _ = fw2.Write([]byte("fake javac binary"))

	_ = zw.Close()
	return buf.Bytes()
}

func TestNewManager(t *testing.T) {
	m1 := NewManager("/tmp/test_java_uvm")
	if m1.BaseDir != "/tmp/test_java_uvm" {
		t.Errorf("expected BaseDir /tmp/test_java_uvm, got %s", m1.BaseDir)
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
		{"21.0.6", "21.0.6"},
		{"v21.0.6", "21.0.6"},
		{"jdk-21.0.6", "21.0.6"},
		{"jdk21", "21"},
		{"openjdk-21", "21"},
		{"java-17", "17"},
		{"java21", "21"},
		{"1.8", "8"},
		{"1.8.0", "8"},
		{"8", "8"},
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

	if v, _ := m.ResolveRemoteVersion("latest"); v != "23.0.2" {
		t.Errorf("expected latest to resolve to 23.0.2, got %s", v)
	}
	if v, _ := m.ResolveRemoteVersion("lts"); v != "21.0.6" {
		t.Errorf("expected lts to resolve to 21.0.6, got %s", v)
	}
	if v, _ := m.ResolveRemoteVersion("stable"); v != "21.0.6" {
		t.Errorf("expected stable to resolve to 21.0.6, got %s", v)
	}
	if v, _ := m.ResolveRemoteVersion("21"); v != "21.0.6" {
		t.Errorf("expected 21 to resolve to 21.0.6, got %s", v)
	}
	if v, _ := m.ResolveRemoteVersion("17"); v != "17.0.14" {
		t.Errorf("expected 17 to resolve to 17.0.14, got %s", v)
	}
	if v, _ := m.ResolveRemoteVersion("11"); v != "11.0.26" {
		t.Errorf("expected 11 to resolve to 11.0.26, got %s", v)
	}
	if v, _ := m.ResolveRemoteVersion("8"); v != "8.0.442" {
		t.Errorf("expected 8 to resolve to 8.0.442, got %s", v)
	}
	if v, _ := m.ResolveRemoteVersion("21.0.2"); v != "21.0.2" {
		t.Errorf("expected 21.0.2 to stay 21.0.2, got %s", v)
	}
	if _, err := m.ResolveRemoteVersion(""); err == nil {
		t.Errorf("expected error for empty version")
	}

	alias, _ := m.ResolveVersion("21")
	if alias != "21.0.6" {
		t.Errorf("expected ResolveVersion alias 21.0.6, got %s", alias)
	}
}

func TestGetArchiveTarget(t *testing.T) {
	m := NewManager("")

	m.GOOS = "darwin"
	m.GOARCH = "arm64"
	url, file, isZip, err := m.GetArchiveTarget("21.0.6")
	if err != nil || isZip || !strings.Contains(file, "aarch64_mac") || !strings.Contains(url, "/21/ga/mac/aarch64") {
		t.Errorf("unexpected darwin arm64 target: url=%s, file=%s, isZip=%v, err=%v", url, file, isZip, err)
	}

	m.GOOS = "darwin"
	m.GOARCH = "amd64"
	url, file, _, _ = m.GetArchiveTarget("21.0.6")
	if !strings.Contains(file, "x64_mac") || !strings.Contains(url, "/mac/x64") {
		t.Errorf("expected x64_mac in darwin amd64 filename/url, got: %s / %s", file, url)
	}

	m.GOOS = "linux"
	m.GOARCH = "amd64"
	url, file, _, _ = m.GetArchiveTarget("17.0.14")
	if !strings.Contains(file, "x64_linux") || !strings.Contains(url, "/17/ga/linux/x64") {
		t.Errorf("expected x64_linux in linux filename/url, got: %s / %s", file, url)
	}

	m.GOOS = "windows"
	m.GOARCH = "amd64"
	url, file, isZip, err = m.GetArchiveTarget("21.0.6")
	if err != nil || !isZip || !strings.Contains(file, "x64_windows") || !strings.Contains(url, "/windows/x64") {
		t.Errorf("unexpected windows amd64 target: url=%s, file=%s, isZip=%v, err=%v", url, file, isZip, err)
	}

	m.GOOS = "unsupported_os"
	_, _, _, err = m.GetArchiveTarget("21.0.6")
	if err == nil {
		t.Errorf("expected error for unsupported OS")
	}
}

func TestInstallTarGzStandardAndMacBundle(t *testing.T) {
	// 1. Standard linux layout
	mockArchive := createMockJavaTarGz("jdk-21.0.6", false)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(mockArchive)
	}))
	defer server.Close()

	tmpBase := t.TempDir()
	m := NewManager(tmpBase)
	m.AdoptiumAPIURL = server.URL
	m.GOOS = "linux"
	m.GOARCH = "amd64"

	outBuf := new(bytes.Buffer)
	err := m.Install("21.0.6", outBuf)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	outStr := outBuf.String()
	if !strings.Contains(outStr, "installed successfully") {
		t.Errorf("expected success output, got: %s", outStr)
	}

	// 2. Mac bundle layout
	mockMacArchive := createMockJavaTarGz("jdk-21.0.6+7", true)
	serverMac := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(mockMacArchive)
	}))
	defer serverMac.Close()

	tmpBaseMac := t.TempDir()
	mMac := NewManager(tmpBaseMac)
	mMac.AdoptiumAPIURL = serverMac.URL
	mMac.GOOS = "darwin"
	mMac.GOARCH = "arm64"

	err = mMac.Install("21.0.6", new(bytes.Buffer))
	if err != nil {
		t.Fatalf("Mac Install failed: %v", err)
	}

	// Verify LocateJavaHome found Contents/Home
	sourceDir := filepath.Join(mMac.VersionsDir(), "21.0.6")
	home := mMac.LocateJavaHome(sourceDir)
	if !strings.Contains(home, filepath.Join("Contents", "Home")) {
		t.Errorf("expected LocateJavaHome to point to Contents/Home, got: %s", home)
	}
}

func TestInstallZip(t *testing.T) {
	mockZip := createMockJavaZip("jdk-21.0.6")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(mockZip)
	}))
	defer server.Close()

	tmpBase := t.TempDir()
	m := NewManager(tmpBase)
	m.AdoptiumAPIURL = server.URL
	m.GOOS = "windows"
	m.GOARCH = "amd64"

	outBuf := new(bytes.Buffer)
	err := m.Install("21.0.6", outBuf)
	if err != nil {
		t.Fatalf("Install zip failed: %v", err)
	}

	// Verify windows shims created
	shimExe := filepath.Join(m.BinDir(), "java.exe")
	if _, err := os.Stat(shimExe); err != nil {
		t.Errorf("expected %s to exist", shimExe)
	}
	shimCmd := filepath.Join(m.BinDir(), "java.cmd")
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
	v1Dir := filepath.Join(m.VersionsDir(), "21.0.6", "bin")
	_ = os.MkdirAll(v1Dir, 0755)
	_ = os.WriteFile(filepath.Join(v1Dir, "java"), []byte("bin1"), 0755)
	_ = os.WriteFile(filepath.Join(v1Dir, "javac"), []byte("javac1"), 0755)

	v2Dir := filepath.Join(m.VersionsDir(), "17.0.14", "bin")
	_ = os.MkdirAll(v2Dir, 0755)
	_ = os.WriteFile(filepath.Join(v2Dir, "java"), []byte("bin2"), 0755)

	// Test Current before Use
	_, err := m.Current()
	if err == nil {
		t.Errorf("expected error when no version is selected")
	}

	// Use v1
	outBuf := new(bytes.Buffer)
	err = m.Use("21", outBuf)
	if err != nil {
		t.Fatalf("Use 21 failed: %v", err)
	}

	curr, err := m.Current()
	if err != nil || curr != "21.0.6" {
		t.Errorf("expected current version 21.0.6, got %s (err: %v)", curr, err)
	}

	// Verify Unix shim
	shim := filepath.Join(m.BinDir(), "java")
	if _, err := os.Lstat(shim); err != nil {
		t.Errorf("expected shim symlink at %s", shim)
	}

	// Use uninstalled
	err = m.Use("11", new(bytes.Buffer))
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
	_ = os.MkdirAll(filepath.Join(m.VersionsDir(), "21.0.6"), 0755)
	_ = os.MkdirAll(filepath.Join(m.VersionsDir(), "17.0.14"), 0755)

	_ = m.Use("21.0.6", new(bytes.Buffer))

	list, err = m.ListInstalled()
	if err != nil || len(list) != 2 {
		t.Fatalf("expected 2 installed versions, got %d (err: %v)", len(list), err)
	}

	foundActive := false
	for _, v := range list {
		if v.Version == "21.0.6" && v.IsActive {
			foundActive = true
		}
	}
	if !foundActive {
		t.Errorf("expected 21.0.6 to be marked as active")
	}
}

func TestRemove(t *testing.T) {
	tmpBase := t.TempDir()
	m := NewManager(tmpBase)
	m.GOOS = "linux"

	_ = os.MkdirAll(filepath.Join(m.VersionsDir(), "21.0.6", "bin"), 0755)
	_ = os.WriteFile(filepath.Join(m.VersionsDir(), "21.0.6", "bin", "java"), []byte("bin"), 0755)
	_ = m.Use("21.0.6", new(bytes.Buffer))

	outBuf := new(bytes.Buffer)
	err := m.Remove("21", outBuf)
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
	err = m.Remove("21.0.6", new(bytes.Buffer))
	if err == nil {
		t.Errorf("expected error removing non-installed version")
	}
}

func TestListRemote(t *testing.T) {
	avail := AvailableReleasesResponse{
		AvailableLTSReleases:     []int{21, 17, 11, 8},
		AvailableReleases:        []int{23, 22, 21, 20, 19, 18, 17, 16, 11, 8},
		MostRecentFeatureRelease: 23,
		MostRecentLTS:            21,
	}
	data, _ := json.Marshal(avail)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}))
	defer server.Close()

	m := NewManager(t.TempDir())
	m.AdoptiumAPIURL = server.URL

	releases, err := m.ListRemote(3)
	if err != nil || len(releases) != 3 {
		t.Fatalf("expected 3 releases from server, got %d (err: %v)", len(releases), err)
	}

	// Test fallback with invalid server
	m.AdoptiumAPIURL = "http://invalid-url-that-fails-12345.com"
	fallbackList, err := m.ListRemote(5)
	if err != nil || len(fallbackList) == 0 {
		t.Errorf("expected fallback releases, got: %+v (err: %v)", fallbackList, err)
	}
}

func TestResolveInstalledVersionPartial(t *testing.T) {
	tmpBase := t.TempDir()
	m := NewManager(tmpBase)

	_ = os.MkdirAll(filepath.Join(m.VersionsDir(), "21.0.6"), 0755)
	_ = os.MkdirAll(filepath.Join(m.VersionsDir(), "21.0.2"), 0755)
	_ = os.MkdirAll(filepath.Join(m.VersionsDir(), "17.0.14"), 0755)

	res, err := m.ResolveInstalledVersion("21")
	if err != nil || res != "21.0.6" {
		t.Errorf("expected 21 -> 21.0.6, got %s (err: %v)", res, err)
	}

	res, err = m.ResolveInstalledVersion("17")
	if err != nil || res != "17.0.14" {
		t.Errorf("expected 17 -> 17.0.14, got %s (err: %v)", res, err)
	}

	res, err = m.ResolveInstalledVersion("21.0.2")
	if err != nil || res != "21.0.2" {
		t.Errorf("expected exact 21.0.2, got %s (err: %v)", res, err)
	}
}

func TestCompareVersions(t *testing.T) {
	if compareVersions("23.0.2", "21.0.6") <= 0 {
		t.Errorf("expected 23.0.2 > 21.0.6")
	}
	if compareVersions("21.0.6", "21.0.6") != 0 {
		t.Errorf("expected 21.0.6 == 21.0.6")
	}
	if compareVersions("8.0.442", "11.0.0") >= 0 {
		t.Errorf("expected 8.0.442 < 11.0.0")
	}
}
