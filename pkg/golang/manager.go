package golang

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// DefaultGoDistURL is the official Go distribution mirror
const DefaultGoDistURL = "https://go.dev/dl"

// GoRelease represents a release entry from the Go dl API
type GoRelease struct {
	Version string        `json:"version"`
	Stable  bool          `json:"stable"`
	Files   []GoFileEntry `json:"files"`
}

// GoFileEntry represents a file in a Go release
type GoFileEntry struct {
	Filename string `json:"filename"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Version  string `json:"version"`
	Kind     string `json:"kind"`
}

// RemoteVersion represents a remote Go release available for installation
type RemoteVersion struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

// InstalledVersion represents a locally installed Go version
type InstalledVersion struct {
	Version  string `json:"version"`
	IsActive bool   `json:"isActive"`
	Path     string `json:"path"`
}

// Manager handles installation, switching, and deletion of Go runtimes
type Manager struct {
	BaseDir    string
	GoDistURL  string
	HTTPClient *http.Client
	GOOS       string
	GOARCH     string
}

// NewManager creates a Go manager rooted at baseDir (e.g., ~/.uvm)
func NewManager(baseDir string) *Manager {
	if baseDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = os.Getenv("HOME")
			if home == "" {
				home = os.Getenv("USERPROFILE")
			}
		}
		baseDir = filepath.Join(home, ".uvm")
	}

	return &Manager{
		BaseDir:    baseDir,
		GoDistURL:  DefaultGoDistURL,
		HTTPClient: http.DefaultClient,
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
	}
}

// VersionsDir returns the folder where all Go versions are stored
func (m *Manager) VersionsDir() string {
	return filepath.Join(m.BaseDir, "versions", "go")
}

// CurrentDir returns the folder for the active runtime symlink
func (m *Manager) CurrentDir() string {
	return filepath.Join(m.BaseDir, "current", "go")
}

// BinDir returns the active binary directory where go and gofmt shims live
func (m *Manager) BinDir() string {
	return filepath.Join(m.BaseDir, "bin")
}

// NormalizeVersion cleans and ensures standard version format (e.g. 1.22.0 -> go1.22.0)
func (m *Manager) NormalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if v == "latest" || v == "current" || v == "stable" || v == "lts" {
		return v
	}
	if !strings.HasPrefix(v, "go") {
		return "go" + v
	}
	return v
}

// FetchRemoteReleases queries the remote Go release API
func (m *Manager) FetchRemoteReleases(includeAll bool) ([]GoRelease, error) {
	query := "mode=json"
	if includeAll {
		query = "mode=json&include=all"
	}
	reqURL := fmt.Sprintf("%s/?%s", strings.TrimSuffix(m.GoDistURL, "/"), query)
	resp, err := m.HTTPClient.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Go version index from %s: %w", reqURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch Go versions (HTTP %d)", resp.StatusCode)
	}

	var releases []GoRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to parse Go version index: %w", err)
	}

	if len(releases) == 0 {
		return nil, fmt.Errorf("no Go releases found")
	}

	return releases, nil
}

// ListRemote returns available remote Go releases
func (m *Manager) ListRemote(limit int) ([]RemoteVersion, error) {
	releases, err := m.FetchRemoteReleases(true)
	if err != nil {
		// Try basic releases
		releases, err = m.FetchRemoteReleases(false)
		if err != nil {
			return nil, err
		}
	}

	var list []RemoteVersion
	for _, r := range releases {
		list = append(list, RemoteVersion{
			Version: r.Version,
			Stable:  r.Stable,
		})
		if limit > 0 && len(list) >= limit {
			break
		}
	}
	return list, nil
}

// ResolveInstalledVersion finds the best locally installed version matching input (exact or partial prefix e.g. "1.22")
func (m *Manager) ResolveInstalledVersion(versionInput string) (string, error) {
	norm := m.NormalizeVersion(versionInput)
	if norm == "" {
		return "", fmt.Errorf("version cannot be empty")
	}

	installed, err := m.ListInstalled()
	if err != nil || len(installed) == 0 {
		return norm, nil
	}

	// 1. Exact match check
	for _, v := range installed {
		if v.Version == norm {
			return v.Version, nil
		}
	}

	// 2. Prefix / partial match (e.g. "1.22" / "go1.22" matching "go1.22.0")
	prefix := norm
	if !strings.HasSuffix(prefix, ".") {
		prefix = prefix + "."
	}

	var matches []string
	for _, v := range installed {
		if strings.HasPrefix(v.Version, prefix) || v.Version == norm {
			matches = append(matches, v.Version)
		}
	}

	if len(matches) > 0 {
		sort.Slice(matches, func(i, j int) bool {
			return compareVersions(matches[i], matches[j]) > 0
		})
		return matches[0], nil
	}

	return norm, nil
}

// ResolveRemoteVersion resolves alias keywords or partial prefixes (e.g. "1.22", "latest") to concrete Go versions
func (m *Manager) ResolveRemoteVersion(versionInput string) (string, error) {
	norm := m.NormalizeVersion(versionInput)
	if norm == "" {
		return "", fmt.Errorf("version cannot be empty")
	}

	if norm != "latest" && norm != "current" && norm != "stable" && norm != "lts" && strings.Count(norm, ".") >= 2 {
		return norm, nil
	}

	releases, err := m.FetchRemoteReleases(true)
	if err != nil {
		releases, err = m.FetchRemoteReleases(false)
	}

	if err != nil {
		if norm == "latest" || norm == "current" || norm == "stable" || norm == "lts" {
			return "", err
		}
		return norm, nil
	}

	if norm == "latest" || norm == "current" || norm == "stable" || norm == "lts" {
		for _, rel := range releases {
			if rel.Stable && rel.Version != "" {
				return rel.Version, nil
			}
		}
		return releases[0].Version, nil
	}

	// Prefix / partial match (e.g. "go1.22" -> "go1.22.12")
	prefix := norm
	if !strings.HasSuffix(prefix, ".") {
		prefix = prefix + "."
	}

	for _, rel := range releases {
		if strings.HasPrefix(rel.Version, prefix) || rel.Version == norm {
			return rel.Version, nil
		}
	}

	return norm, nil
}

// ResolveVersion resolves a Go version string
func (m *Manager) ResolveVersion(versionInput string) (string, error) {
	return m.ResolveRemoteVersion(versionInput)
}

// GetArchiveTarget determines filename and archive type based on OS/Arch
func (m *Manager) GetArchiveTarget(version string) (fileName string, isZip bool, err error) {
	var osName, archName string

	switch m.GOOS {
	case "darwin":
		osName = "darwin"
		if m.GOARCH == "arm64" {
			archName = "arm64"
		} else {
			archName = "amd64"
		}
		fileName = fmt.Sprintf("%s.%s-%s.tar.gz", version, osName, archName)
		return fileName, false, nil

	case "windows":
		osName = "windows"
		if m.GOARCH == "arm64" {
			archName = "arm64"
		} else {
			archName = "amd64"
		}
		fileName = fmt.Sprintf("%s.%s-%s.zip", version, osName, archName)
		return fileName, true, nil

	case "linux":
		osName = "linux"
		if m.GOARCH == "arm64" {
			archName = "arm64"
		} else if m.GOARCH == "arm" {
			archName = "armv6l"
		} else {
			archName = "amd64"
		}
		fileName = fmt.Sprintf("%s.%s-%s.tar.gz", version, osName, archName)
		return fileName, false, nil

	default:
		return "", false, fmt.Errorf("unsupported OS: %s", m.GOOS)
	}
}

// Install downloads and extracts a Go version
func (m *Manager) Install(versionInput string, out io.Writer) error {
	version, err := m.ResolveRemoteVersion(versionInput)
	if err != nil {
		return err
	}

	destDir := filepath.Join(m.VersionsDir(), version)
	if _, err := os.Stat(destDir); err == nil {
		fmt.Fprintf(out, "Go %s is already installed at %s\n", version, destDir)
		return m.Use(version, out)
	}

	fileName, isZip, err := m.GetArchiveTarget(version)
	if err != nil {
		return err
	}

	downloadURL := fmt.Sprintf("%s/%s", strings.TrimSuffix(m.GoDistURL, "/"), fileName)
	fmt.Fprintf(out, "Downloading Go %s from %s...\n", version, downloadURL)

	resp, err := m.HTTPClient.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download Go archive: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download Go %s (HTTP %d from %s)", version, resp.StatusCode, downloadURL)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create version directory %s: %w", destDir, err)
	}

	fmt.Fprintf(out, "Extracting %s...\n", fileName)

	if isZip {
		tmpZip, err := os.CreateTemp("", "go-*.zip")
		if err != nil {
			return err
		}
		defer os.Remove(tmpZip.Name())

		if _, err := io.Copy(tmpZip, resp.Body); err != nil {
			tmpZip.Close()
			return err
		}
		tmpZip.Close()

		if err := extractZip(tmpZip.Name(), destDir); err != nil {
			_ = os.RemoveAll(destDir)
			return fmt.Errorf("failed to extract zip archive: %w", err)
		}
	} else {
		if err := extractTarGz(resp.Body, destDir); err != nil {
			_ = os.RemoveAll(destDir)
			return fmt.Errorf("failed to extract tar.gz archive: %w", err)
		}
	}

	fmt.Fprintf(out, "✓ Go %s installed successfully to %s\n", version, destDir)
	return m.Use(version, out)
}

// Use switches the active Go version (supporting partial prefixes e.g. "1.22")
func (m *Manager) Use(versionInput string, out io.Writer) error {
	version, err := m.ResolveInstalledVersion(versionInput)
	if err != nil {
		return err
	}

	sourceDir := filepath.Join(m.VersionsDir(), version)
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return fmt.Errorf("Go %s is not installed. Run 'uvm install go %s' first", version, version)
	}

	if err := os.MkdirAll(m.BinDir(), 0755); err != nil {
		return err
	}

	currentParent := filepath.Dir(m.CurrentDir())
	if err := os.MkdirAll(currentParent, 0755); err != nil {
		return err
	}

	// Update current active link or active record
	_ = os.Remove(m.CurrentDir())
	_ = os.Symlink(sourceDir, m.CurrentDir())

	// Write active version state file
	versionStateFile := filepath.Join(currentParent, "go.version")
	if err := os.WriteFile(versionStateFile, []byte(version), 0644); err != nil {
		return err
	}

	// Create / Update active shims in bin/
	if m.GOOS == "windows" {
		binaries := []string{"go.exe", "gofmt.exe"}
		for _, b := range binaries {
			srcBin := filepath.Join(sourceDir, "bin", b)
			dstBin := filepath.Join(m.BinDir(), b)
			if _, err := os.Stat(srcBin); err == nil {
				_ = os.Remove(dstBin)
				_ = copyFile(srcBin, dstBin)
			} else {
				_ = os.WriteFile(dstBin, []byte(b), 0755)
			}
		}

		cmdBinaries := []string{"go.cmd", "gofmt.cmd"}
		for _, b := range cmdBinaries {
			shimPath := filepath.Join(m.BinDir(), b)
			exeName := strings.TrimSuffix(b, ".cmd") + ".exe"
			targetExe := filepath.Join(sourceDir, "bin", exeName)
			content := fmt.Sprintf("@ECHO off\r\n\"%s\" %%*\r\n", targetExe)
			_ = os.WriteFile(shimPath, []byte(content), 0755)
		}
	} else {
		binaries := []string{"go", "gofmt"}
		for _, b := range binaries {
			shimPath := filepath.Join(m.BinDir(), b)
			targetExe := filepath.Join(sourceDir, "bin", b)
			_ = os.Remove(shimPath)
			_ = os.Symlink(targetExe, shimPath)
		}
	}

	fmt.Fprintf(out, "Now using Go %s\n", version)

	pathEnv := os.Getenv("PATH")
	if !strings.Contains(pathEnv, m.BinDir()) {
		fmt.Fprintf(out, "\nℹ Note: %s is not in your current PATH.\n", m.BinDir())
		if m.GOOS == "windows" {
			fmt.Fprintf(out, "To use go immediately in this terminal, run:\n  $env:Path = \"%s;\" + $env:Path\n", m.BinDir())
		} else {
			fmt.Fprintf(out, "To use go immediately in this terminal, run:\n  export PATH=\"%s:$PATH\"\n", m.BinDir())
		}
	}
	return nil
}

// Current returns the currently active Go version
func (m *Manager) Current() (string, error) {
	versionStateFile := filepath.Join(filepath.Dir(m.CurrentDir()), "go.version")
	data, err := os.ReadFile(versionStateFile)
	if err != nil {
		return "", fmt.Errorf("no active Go version selected (run 'uvm use go <version>')")
	}
	return strings.TrimSpace(string(data)), nil
}

// ListInstalled returns all locally installed Go versions
func (m *Manager) ListInstalled() ([]InstalledVersion, error) {
	vDir := m.VersionsDir()
	if _, err := os.Stat(vDir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(vDir)
	if err != nil {
		return nil, err
	}

	activeVer, _ := m.Current()
	var list []InstalledVersion

	for _, entry := range entries {
		if entry.IsDir() {
			vName := entry.Name()
			list = append(list, InstalledVersion{
				Version:  vName,
				IsActive: vName == activeVer,
				Path:     filepath.Join(vDir, vName),
			})
		}
	}

	return list, nil
}

// Remove uninstalls a Go version (supporting partial prefixes e.g. "1.22")
func (m *Manager) Remove(versionInput string, out io.Writer) error {
	version, err := m.ResolveInstalledVersion(versionInput)
	if err != nil {
		return err
	}

	destDir := filepath.Join(m.VersionsDir(), version)
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		return fmt.Errorf("Go %s is not installed", version)
	}

	// If removing active version, remove active state and shims
	active, _ := m.Current()
	if active == version {
		_ = os.Remove(filepath.Join(filepath.Dir(m.CurrentDir()), "go.version"))
		_ = os.Remove(m.CurrentDir())
		if m.GOOS == "windows" {
			_ = os.Remove(filepath.Join(m.BinDir(), "go.exe"))
			_ = os.Remove(filepath.Join(m.BinDir(), "gofmt.exe"))
			_ = os.Remove(filepath.Join(m.BinDir(), "go.cmd"))
			_ = os.Remove(filepath.Join(m.BinDir(), "gofmt.cmd"))
		} else {
			_ = os.Remove(filepath.Join(m.BinDir(), "go"))
			_ = os.Remove(filepath.Join(m.BinDir(), "gofmt"))
		}
	}

	if err := os.RemoveAll(destDir); err != nil {
		return fmt.Errorf("failed to remove Go %s: %w", version, err)
	}

	fmt.Fprintf(out, "✓ Go %s removed successfully\n", version)
	return nil
}

// extractTarGz unpacks a .tar.gz archive, stripping the top-level directory prefix (e.g. go/bin/go -> bin/go)
func extractTarGz(r io.Reader, destDir string) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		relPath := stripTopDir(header.Name)
		if relPath == "" || relPath == "." {
			continue
		}

		targetPath := filepath.Join(destDir, relPath)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, header.FileInfo().Mode())
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}
			_ = os.Remove(targetPath)
			_ = os.Symlink(header.Linkname, targetPath)
		}
	}

	return nil
}

// extractZip unpacks a .zip archive, stripping the top-level directory prefix
func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		relPath := stripTopDir(f.Name)
		if relPath == "" || relPath == "." {
			continue
		}

		targetPath := filepath.Join(destDir, relPath)

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

// stripTopDir strips the first directory element in a path (e.g. go/bin/go -> bin/go)
func stripTopDir(p string) string {
	clean := filepath.ToSlash(p)
	parts := strings.Split(clean, "/")
	if len(parts) <= 1 {
		return ""
	}
	return filepath.Join(parts[1:]...)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func compareVersions(v1, v2 string) int {
	clean1 := strings.TrimLeft(strings.TrimSpace(v1), "vgoPython ")
	clean2 := strings.TrimLeft(strings.TrimSpace(v2), "vgoPython ")
	parts1 := strings.Split(clean1, ".")
	parts2 := strings.Split(clean2, ".")
	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}
	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		if i < len(parts1) {
			_, _ = fmt.Sscanf(parts1[i], "%d", &n1)
		}
		if i < len(parts2) {
			_, _ = fmt.Sscanf(parts2[i], "%d", &n2)
		}
		if n1 > n2 {
			return 1
		}
		if n1 < n2 {
			return -1
		}
	}
	return 0
}
