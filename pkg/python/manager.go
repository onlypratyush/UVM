package python

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

// DefaultPythonDistURL is the default standalone Python releases base URL
const DefaultPythonDistURL = "https://github.com/astral-sh/python-build-standalone/releases/download"

// DefaultLatestReleaseMetadataURL is the raw metadata endpoint for latest-release
const DefaultLatestReleaseMetadataURL = "https://raw.githubusercontent.com/astral-sh/python-build-standalone/latest-release/latest-release.json"

// DefaultReleaseTag is the fallback release tag if online metadata resolution is unavailable
const DefaultReleaseTag = "20250212"

// KnownPythonReleases is the list of standard supported Python versions
var KnownPythonReleases = []RemoteVersion{
	{Version: "3.13.2", LTS: "Latest Stable"},
	{Version: "3.12.9", LTS: "LTS"},
	{Version: "3.11.11", LTS: "Security Support"},
	{Version: "3.10.16", LTS: "Security Support"},
	{Version: "3.9.21", LTS: "Security Support"},
	{Version: "3.8.20", LTS: "EOL"},
}

// PythonReleaseMetadata holds metadata returned by latest-release.json
type PythonReleaseMetadata struct {
	Version        int    `json:"version"`
	Tag            string `json:"tag"`
	ReleaseURL     string `json:"release_url"`
	AssetURLPrefix string `json:"asset_url_prefix"`
}

// RemoteVersion represents a remote Python release available for installation
type RemoteVersion struct {
	Version string `json:"version"`
	LTS     string `json:"lts,omitempty"`
}

// InstalledVersion represents a locally installed Python version
type InstalledVersion struct {
	Version  string `json:"version"`
	IsActive bool   `json:"isActive"`
	Path     string `json:"path"`
}

// Manager handles installation, switching, and deletion of Python runtimes
type Manager struct {
	BaseDir       string
	PythonDistURL string
	MetadataURL   string
	DefaultTag    string
	HTTPClient    *http.Client
	GOOS          string
	GOARCH        string
}

// NewManager creates a Python manager rooted at baseDir (e.g., ~/.uvm)
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
		BaseDir:       baseDir,
		PythonDistURL: DefaultPythonDistURL,
		MetadataURL:   DefaultLatestReleaseMetadataURL,
		DefaultTag:    DefaultReleaseTag,
		HTTPClient:    http.DefaultClient,
		GOOS:          runtime.GOOS,
		GOARCH:        runtime.GOARCH,
	}
}

// VersionsDir returns the folder where all Python versions are stored
func (m *Manager) VersionsDir() string {
	return filepath.Join(m.BaseDir, "versions", "python")
}

// CurrentDir returns the folder for the active runtime symlink
func (m *Manager) CurrentDir() string {
	return filepath.Join(m.BaseDir, "current", "python")
}

// BinDir returns the active binary directory where python, pip shims live
func (m *Manager) BinDir() string {
	return filepath.Join(m.BaseDir, "bin")
}

// NormalizeVersion cleans version string (e.g. v3.12.2 -> 3.12.2, python3.11 -> 3.11)
func (m *Manager) NormalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	v = strings.TrimPrefix(v, "python")
	v = strings.TrimPrefix(v, "py")
	v = strings.TrimPrefix(v, "v")
	return strings.TrimSpace(v)
}

// ListRemote returns available remote Python releases
func (m *Manager) ListRemote(limit int) ([]RemoteVersion, error) {
	list := KnownPythonReleases
	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}
	return list, nil
}

// ResolveInstalledVersion finds the best locally installed version matching input (exact or partial prefix e.g. "3.12")
func (m *Manager) ResolveInstalledVersion(versionInput string) (string, error) {
	norm := m.NormalizeVersion(versionInput)
	if norm == "" {
		return "", fmt.Errorf("version cannot be empty")
	}

	installed, err := m.ListInstalled()
	if err != nil || len(installed) == 0 {
		return norm, nil
	}

	// 1. Exact match
	for _, v := range installed {
		if v.Version == norm {
			return v.Version, nil
		}
	}

	// 2. Prefix match (e.g. "3.12" matching "3.12.2")
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

// ResolveRemoteVersion resolves alias keywords or partial prefixes (e.g. "3.12", "latest") into concrete semantic versions
func (m *Manager) ResolveRemoteVersion(versionInput string) (string, error) {
	norm := m.NormalizeVersion(versionInput)
	if norm == "" {
		return "", fmt.Errorf("version cannot be empty")
	}

	if norm == "latest" || norm == "current" {
		return "3.13.2", nil
	}
	if norm == "lts" {
		return "3.12.9", nil
	}

	// If already a full version (e.g. 3.12.2 with 2 dots), return directly
	if strings.Count(norm, ".") >= 2 {
		return norm, nil
	}

	// Prefix matching against known releases (e.g. "3.12" -> "3.12.9", "3.11" -> "3.11.11")
	prefix := norm
	if !strings.HasSuffix(prefix, ".") {
		prefix = prefix + "."
	}

	for _, rel := range KnownPythonReleases {
		if strings.HasPrefix(rel.Version, prefix) || rel.Version == norm {
			return rel.Version, nil
		}
	}

	return norm, nil
}

// ResolveVersion resolves alias keywords into concrete versions
func (m *Manager) ResolveVersion(versionInput string) (string, error) {
	return m.ResolveRemoteVersion(versionInput)
}

// GetReleaseTagForVersion maps Python versions to known release tags or falls back to latest tag
func (m *Manager) GetReleaseTagForVersion(v string) string {
	versionTagMap := map[string]string{
		"3.13.2":  "20250212",
		"3.12.9":  "20250212",
		"3.11.11": "20250212",
		"3.10.16": "20250212",
		"3.9.21":  "20250212",
		"3.8.20":  "20241016",
	}
	if tag, ok := versionTagMap[v]; ok {
		return tag
	}
	return m.FetchLatestReleaseTag()
}

// FetchLatestReleaseTag retrieves the active tag from latest-release metadata or falls back to DefaultTag
func (m *Manager) FetchLatestReleaseTag() string {
	if m.MetadataURL == "" {
		return m.DefaultTag
	}

	resp, err := m.HTTPClient.Get(m.MetadataURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		return m.DefaultTag
	}
	defer resp.Body.Close()

	var meta PythonReleaseMetadata
	if err := json.NewDecoder(resp.Body).Decode(&meta); err == nil && meta.Tag != "" {
		return meta.Tag
	}

	return m.DefaultTag
}

// GetArchiveTarget determines filename and archive type based on OS/Arch
func (m *Manager) GetArchiveTarget(version string, tag string) (fileName string, isZip bool, err error) {
	if tag == "" {
		tag = m.DefaultTag
	}

	var archStr, platformStr string

	switch m.GOOS {
	case "darwin":
		platformStr = "apple-darwin"
		if m.GOARCH == "arm64" {
			archStr = "aarch64"
		} else {
			archStr = "x86_64"
		}
		fileName = fmt.Sprintf("cpython-%s+%s-%s-%s-install_only.tar.gz", version, tag, archStr, platformStr)
		return fileName, false, nil

	case "windows":
		platformStr = "pc-windows-msvc-shared"
		if m.GOARCH == "arm64" {
			archStr = "aarch64"
		} else {
			archStr = "x86_64"
		}
		fileName = fmt.Sprintf("cpython-%s+%s-%s-%s-install_only.tar.gz", version, tag, archStr, platformStr)
		return fileName, false, nil

	case "linux":
		platformStr = "unknown-linux-gnu"
		if m.GOARCH == "arm64" {
			archStr = "aarch64"
		} else if m.GOARCH == "arm" {
			archStr = "armv7"
		} else {
			archStr = "x86_64"
		}
		fileName = fmt.Sprintf("cpython-%s+%s-%s-%s-install_only.tar.gz", version, tag, archStr, platformStr)
		return fileName, false, nil

	default:
		return "", false, fmt.Errorf("unsupported OS: %s", m.GOOS)
	}
}

// Install downloads and extracts a Python version
func (m *Manager) Install(versionInput string, out io.Writer) error {
	version, err := m.ResolveRemoteVersion(versionInput)
	if err != nil {
		return err
	}

	destDir := filepath.Join(m.VersionsDir(), version)
	if _, err := os.Stat(destDir); err == nil {
		fmt.Fprintf(out, "Python %s is already installed at %s\n", version, destDir)
		return m.Use(version, out)
	}

	tag := m.GetReleaseTagForVersion(version)
	fileName, isZip, err := m.GetArchiveTarget(version, tag)
	if err != nil {
		return err
	}

	downloadURL := fmt.Sprintf("%s/%s/%s", strings.TrimSuffix(m.PythonDistURL, "/"), tag, fileName)
	fmt.Fprintf(out, "Downloading Python %s from %s...\n", version, downloadURL)

	resp, err := m.HTTPClient.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download Python archive: %w", err)
	}

	// Fallback to default tag if initial tag returns 404
	if resp.StatusCode != http.StatusOK && tag != m.DefaultTag {
		resp.Body.Close()
		tag = m.DefaultTag
		fileName, isZip, _ = m.GetArchiveTarget(version, tag)
		downloadURL = fmt.Sprintf("%s/%s/%s", strings.TrimSuffix(m.PythonDistURL, "/"), tag, fileName)
		fmt.Fprintf(out, "Retrying with tag %s from %s...\n", tag, downloadURL)
		resp, err = m.HTTPClient.Get(downloadURL)
		if err != nil {
			return fmt.Errorf("failed to download Python archive: %w", err)
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download Python %s (HTTP %d from %s)", version, resp.StatusCode, downloadURL)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create version directory %s: %w", destDir, err)
	}

	fmt.Fprintf(out, "Extracting %s...\n", fileName)

	if isZip {
		tmpZip, err := os.CreateTemp("", "python-*.zip")
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

	fmt.Fprintf(out, "✓ Python %s installed successfully to %s\n", version, destDir)
	return m.Use(version, out)
}

// Use switches the active Python version (supporting partial prefixes e.g. "3.12")
func (m *Manager) Use(versionInput string, out io.Writer) error {
	version, err := m.ResolveInstalledVersion(versionInput)
	if err != nil {
		return err
	}

	sourceDir := filepath.Join(m.VersionsDir(), version)
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return fmt.Errorf("Python %s is not installed. Run 'uvm install python %s' first", version, version)
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
	versionStateFile := filepath.Join(currentParent, "python.version")
	if err := os.WriteFile(versionStateFile, []byte(version), 0644); err != nil {
		return err
	}

	// Create / Update active shims in bin/
	if m.GOOS == "windows" {
		// Standalone Python on Windows puts python.exe directly in root or bin/
		srcPythonExe := filepath.Join(sourceDir, "python.exe")
		if _, err := os.Stat(srcPythonExe); os.IsNotExist(err) {
			srcPythonExe = filepath.Join(sourceDir, "bin", "python.exe")
		}

		srcPipExe := filepath.Join(sourceDir, "Scripts", "pip.exe")
		if _, err := os.Stat(srcPipExe); os.IsNotExist(err) {
			srcPipExe = filepath.Join(sourceDir, "bin", "pip.exe")
		}

		dstPythonExe := filepath.Join(m.BinDir(), "python.exe")
		dstPython3Exe := filepath.Join(m.BinDir(), "python3.exe")
		if _, err := os.Stat(srcPythonExe); err == nil {
			_ = os.Remove(dstPythonExe)
			_ = copyFile(srcPythonExe, dstPythonExe)
			_ = os.Remove(dstPython3Exe)
			_ = copyFile(srcPythonExe, dstPython3Exe)
		} else {
			_ = os.WriteFile(dstPythonExe, []byte("python"), 0755)
			_ = os.WriteFile(dstPython3Exe, []byte("python3"), 0755)
		}

		dstPipExe := filepath.Join(m.BinDir(), "pip.exe")
		dstPip3Exe := filepath.Join(m.BinDir(), "pip3.exe")
		if _, err := os.Stat(srcPipExe); err == nil {
			_ = os.Remove(dstPipExe)
			_ = copyFile(srcPipExe, dstPipExe)
			_ = os.Remove(dstPip3Exe)
			_ = copyFile(srcPipExe, dstPip3Exe)
		} else {
			_ = os.WriteFile(dstPipExe, []byte("pip"), 0755)
			_ = os.WriteFile(dstPip3Exe, []byte("pip3"), 0755)
		}

		cmdBinaries := []string{"python.cmd", "python3.cmd", "pip.cmd", "pip3.cmd"}
		for _, b := range cmdBinaries {
			shimPath := filepath.Join(m.BinDir(), b)
			targetExe := srcPythonExe
			if strings.HasPrefix(b, "pip") {
				targetExe = srcPipExe
			}
			content := fmt.Sprintf("@ECHO off\r\n\"%s\" %%*\r\n", targetExe)
			_ = os.WriteFile(shimPath, []byte(content), 0755)
		}
	} else {
		// Unix: link python, python3, pip, pip3
		binNames := []string{"python", "python3", "pip", "pip3"}
		for _, b := range binNames {
			shimPath := filepath.Join(m.BinDir(), b)
			targetExe := filepath.Join(sourceDir, "bin", b)

			// If specific binary doesn't exist, check alternatives (e.g. python3 -> python)
			if _, err := os.Stat(targetExe); os.IsNotExist(err) {
				if b == "python" {
					targetExe = filepath.Join(sourceDir, "bin", "python3")
				} else if b == "pip" {
					targetExe = filepath.Join(sourceDir, "bin", "pip3")
				}
			}

			_ = os.Remove(shimPath)
			_ = os.Symlink(targetExe, shimPath)
		}
	}

	fmt.Fprintf(out, "Now using Python %s\n", version)

	pathEnv := os.Getenv("PATH")
	if !strings.Contains(pathEnv, m.BinDir()) {
		fmt.Fprintf(out, "\nℹ Note: %s is not in your current PATH.\n", m.BinDir())
		if m.GOOS == "windows" {
			fmt.Fprintf(out, "To use python immediately in this terminal, run:\n  $env:Path = \"%s;\" + $env:Path\n", m.BinDir())
		} else {
			fmt.Fprintf(out, "To use python immediately in this terminal, run:\n  export PATH=\"%s:$PATH\"\n", m.BinDir())
		}
	}
	return nil
}

// Current returns the currently active Python version
func (m *Manager) Current() (string, error) {
	versionStateFile := filepath.Join(filepath.Dir(m.CurrentDir()), "python.version")
	data, err := os.ReadFile(versionStateFile)
	if err != nil {
		return "", fmt.Errorf("no active Python version selected (run 'uvm use python <version>')")
	}
	return strings.TrimSpace(string(data)), nil
}

// ListInstalled returns all locally installed Python versions
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

// Remove uninstalls a Python version (supporting partial prefixes e.g. "3.12")
func (m *Manager) Remove(versionInput string, out io.Writer) error {
	version, err := m.ResolveInstalledVersion(versionInput)
	if err != nil {
		return err
	}

	destDir := filepath.Join(m.VersionsDir(), version)
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		return fmt.Errorf("Python %s is not installed", version)
	}

	// If removing active version, remove active state and shims
	active, _ := m.Current()
	if active == version {
		_ = os.Remove(filepath.Join(filepath.Dir(m.CurrentDir()), "python.version"))
		_ = os.Remove(m.CurrentDir())
		if m.GOOS == "windows" {
			_ = os.Remove(filepath.Join(m.BinDir(), "python.exe"))
			_ = os.Remove(filepath.Join(m.BinDir(), "python3.exe"))
			_ = os.Remove(filepath.Join(m.BinDir(), "pip.exe"))
			_ = os.Remove(filepath.Join(m.BinDir(), "pip3.exe"))
			_ = os.Remove(filepath.Join(m.BinDir(), "python.cmd"))
			_ = os.Remove(filepath.Join(m.BinDir(), "python3.cmd"))
			_ = os.Remove(filepath.Join(m.BinDir(), "pip.cmd"))
			_ = os.Remove(filepath.Join(m.BinDir(), "pip3.cmd"))
		} else {
			_ = os.Remove(filepath.Join(m.BinDir(), "python"))
			_ = os.Remove(filepath.Join(m.BinDir(), "python3"))
			_ = os.Remove(filepath.Join(m.BinDir(), "pip"))
			_ = os.Remove(filepath.Join(m.BinDir(), "pip3"))
		}
	}

	if err := os.RemoveAll(destDir); err != nil {
		return fmt.Errorf("failed to remove Python %s: %w", version, err)
	}

	fmt.Fprintf(out, "✓ Python %s removed successfully\n", version)
	return nil
}

// extractTarGz unpacks a .tar.gz archive, stripping the top-level directory prefix (e.g. python/bin/python3 -> bin/python3)
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

// stripTopDir strips the first directory element in a path (e.g. python/bin/python -> bin/python)
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
