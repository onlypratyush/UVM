package php

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
	"strconv"
	"strings"
)

// DefaultPhpDistURL is the default mirror for Windows official PHP archives
const DefaultPhpDistURL = "https://windows.php.net/downloads/releases"

// DefaultPhpUnixDistURL is the standalone portable PHP release mirror for macOS and Linux
const DefaultPhpUnixDistURL = "https://github.com/static-php/static-php-cli/releases/download"

// DefaultPhpReleasesAPIURL is the official php.net release index API
const DefaultPhpReleasesAPIURL = "https://www.php.net/releases/index.php?json"

// KnownPhpReleases is the list of curated standard PHP releases
var KnownPhpReleases = []RemoteVersion{
	{Version: "8.4.4", LTS: "Latest Stable"},
	{Version: "8.3.17", LTS: "LTS"},
	{Version: "8.2.28", LTS: "Security Support"},
	{Version: "8.1.31", LTS: "Security Support"},
	{Version: "8.0.30", LTS: "EOL"},
	{Version: "7.4.33", LTS: "EOL"},
}

// RemoteVersion represents a remote PHP release available for installation
type RemoteVersion struct {
	Version string `json:"version"`
	LTS     string `json:"lts,omitempty"`
}

// InstalledVersion represents a locally installed PHP version
type InstalledVersion struct {
	Version  string `json:"version"`
	IsActive bool   `json:"isActive"`
	Path     string `json:"path"`
}

// Manager handles installation, switching, and deletion of PHP runtimes
type Manager struct {
	BaseDir        string
	PhpDistURL     string
	PhpUnixDistURL string
	ReleasesAPIURL string
	HTTPClient     *http.Client
	GOOS           string
	GOARCH         string
}

// NewManager creates a PHP manager rooted at baseDir (e.g., ~/.uvm)
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
		BaseDir:        baseDir,
		PhpDistURL:     DefaultPhpDistURL,
		PhpUnixDistURL: DefaultPhpUnixDistURL,
		ReleasesAPIURL: DefaultPhpReleasesAPIURL,
		HTTPClient:     http.DefaultClient,
		GOOS:           runtime.GOOS,
		GOARCH:         runtime.GOARCH,
	}
}

// VersionsDir returns the folder where all PHP versions are stored
func (m *Manager) VersionsDir() string {
	return filepath.Join(m.BaseDir, "versions", "php")
}

// CurrentDir returns the folder for the active runtime symlink
func (m *Manager) CurrentDir() string {
	return filepath.Join(m.BaseDir, "current", "php")
}

// BinDir returns the active binary directory where php shims live
func (m *Manager) BinDir() string {
	return filepath.Join(m.BaseDir, "bin")
}

// NormalizeVersion cleans and normalizes version strings (e.g. php8.3.17 -> 8.3.17, v8.4 -> 8.4)
func (m *Manager) NormalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	v = strings.TrimPrefix(v, "php-")
	v = strings.TrimPrefix(v, "php")
	v = strings.TrimPrefix(v, "v")
	return strings.TrimSpace(v)
}

// ListRemote returns available remote PHP releases
func (m *Manager) ListRemote(limit int) ([]RemoteVersion, error) {
	if m.ReleasesAPIURL != "" {
		resp, err := m.HTTPClient.Get(m.ReleasesAPIURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			var rawMap map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&rawMap); err == nil && len(rawMap) > 0 {
				var versions []string
				for k := range rawMap {
					if strings.Count(k, ".") >= 2 {
						versions = append(versions, k)
					}
				}
				if len(versions) > 0 {
					sort.Slice(versions, func(i, j int) bool {
						return compareVersions(versions[i], versions[j]) > 0
					})
					var list []RemoteVersion
					for _, v := range versions {
						lts := ""
						if strings.HasPrefix(v, "8.4.") {
							lts = "Latest Stable"
						} else if strings.HasPrefix(v, "8.3.") {
							lts = "LTS"
						} else if strings.HasPrefix(v, "8.2.") || strings.HasPrefix(v, "8.1.") {
							lts = "Security Support"
						}
						list = append(list, RemoteVersion{
							Version: v,
							LTS:     lts,
						})
						if limit > 0 && len(list) >= limit {
							break
						}
					}
					return list, nil
				}
			}
		}
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}

	list := KnownPhpReleases
	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}
	return list, nil
}

// ResolveInstalledVersion finds the best locally installed version matching input (exact or partial prefix e.g. "8.3")
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

	// 2. Prefix match (e.g. "8.3" matching "8.3.17")
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

// ResolveRemoteVersion resolves alias keywords or partial prefixes (e.g. "8.3", "latest") into concrete semantic versions
func (m *Manager) ResolveRemoteVersion(versionInput string) (string, error) {
	norm := m.NormalizeVersion(versionInput)
	if norm == "" {
		return "", fmt.Errorf("version cannot be empty")
	}

	if norm == "latest" || norm == "current" {
		return "8.4.4", nil
	}
	if norm == "lts" || norm == "stable" {
		return "8.3.17", nil
	}

	// If already a full version (e.g. 8.3.17 with 2 dots), return directly
	if strings.Count(norm, ".") >= 2 {
		return norm, nil
	}

	// Prefix matching against known releases (e.g. "8.3" -> "8.3.17", "8.2" -> "8.2.28")
	prefix := norm
	if !strings.HasSuffix(prefix, ".") {
		prefix = prefix + "."
	}

	for _, rel := range KnownPhpReleases {
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

// GetArchiveTarget determines filename, URL, and archive format based on OS/Arch
func (m *Manager) GetArchiveTarget(version string) (downloadURL string, fileName string, isZip bool, err error) {
	if m.GOOS == "windows" {
		archStr := "x64"
		if m.GOARCH == "386" {
			archStr = "x86"
		}
		// Windows official releases pattern: php-8.3.17-nts-Win32-vs16-x64.zip
		fileName = fmt.Sprintf("php-%s-nts-Win32-vs16-%s.zip", version, archStr)
		downloadURL = fmt.Sprintf("%s/%s", strings.TrimSuffix(m.PhpDistURL, "/"), fileName)
		return downloadURL, fileName, true, nil
	}

	var archStr, osStr string
	switch m.GOOS {
	case "darwin":
		osStr = "macos"
		if m.GOARCH == "arm64" {
			archStr = "aarch64"
		} else {
			archStr = "x86_64"
		}
	case "linux":
		osStr = "linux"
		if m.GOARCH == "arm64" {
			archStr = "aarch64"
		} else {
			archStr = "x86_64"
		}
	default:
		return "", "", false, fmt.Errorf("unsupported OS: %s", m.GOOS)
	}

	fileName = fmt.Sprintf("php-%s-%s-%s.tar.gz", version, osStr, archStr)
	downloadURL = fmt.Sprintf("%s/v%s/%s", strings.TrimSuffix(m.PhpUnixDistURL, "/"), version, fileName)
	return downloadURL, fileName, false, nil
}

// Install downloads and extracts a PHP version
func (m *Manager) Install(versionInput string, out io.Writer) error {
	version, err := m.ResolveRemoteVersion(versionInput)
	if err != nil {
		return err
	}

	destDir := filepath.Join(m.VersionsDir(), version)
	if _, err := os.Stat(destDir); err == nil {
		fmt.Fprintf(out, "PHP %s is already installed at %s\n", version, destDir)
		return m.Use(version, out)
	}

	downloadURL, fileName, isZip, err := m.GetArchiveTarget(version)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "Downloading PHP %s from %s...\n", version, downloadURL)

	resp, err := m.HTTPClient.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download PHP archive: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Try fallback on Windows to thread-safe build or archives folder
		if m.GOOS == "windows" {
			archStr := "x64"
			if m.GOARCH == "386" {
				archStr = "x86"
			}
			fallbackFileName := fmt.Sprintf("php-%s-Win32-vs16-%s.zip", version, archStr)
			fallbackURL := fmt.Sprintf("%s/archives/%s", strings.TrimSuffix(m.PhpDistURL, "/"), fallbackFileName)
			fmt.Fprintf(out, "Retrying from archives: %s...\n", fallbackURL)
			respFallback, errFallback := m.HTTPClient.Get(fallbackURL)
			if errFallback == nil && respFallback.StatusCode == http.StatusOK {
				resp.Body.Close()
				resp = respFallback
				defer resp.Body.Close()
				fileName = fallbackFileName
			} else {
				if respFallback != nil && respFallback.Body != nil {
					respFallback.Body.Close()
				}
				return fmt.Errorf("failed to download PHP %s (HTTP %d from %s)", version, resp.StatusCode, downloadURL)
			}
		} else {
			return fmt.Errorf("failed to download PHP %s (HTTP %d from %s)", version, resp.StatusCode, downloadURL)
		}
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create version directory %s: %w", destDir, err)
	}

	fmt.Fprintf(out, "Extracting %s...\n", fileName)

	if isZip {
		tmpZip, err := os.CreateTemp("", "php-*.zip")
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

	fmt.Fprintf(out, "✓ PHP %s installed successfully to %s\n", version, destDir)
	return m.Use(version, out)
}

// Use switches the active PHP version (supporting partial prefixes e.g. "8.3")
func (m *Manager) Use(versionInput string, out io.Writer) error {
	version, err := m.ResolveInstalledVersion(versionInput)
	if err != nil {
		return err
	}

	sourceDir := filepath.Join(m.VersionsDir(), version)
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return fmt.Errorf("PHP %s is not installed. Run 'uvm install php %s' first", version, version)
	}

	if err := os.MkdirAll(m.BinDir(), 0755); err != nil {
		return err
	}

	currentParent := filepath.Dir(m.CurrentDir())
	if err := os.MkdirAll(currentParent, 0755); err != nil {
		return err
	}

	// Update current active link
	_ = os.Remove(m.CurrentDir())
	_ = os.Symlink(sourceDir, m.CurrentDir())

	// Write active version state file
	versionStateFile := filepath.Join(currentParent, "php.version")
	if err := os.WriteFile(versionStateFile, []byte(version), 0644); err != nil {
		return err
	}

	// Create / Update active shims in bin/
	if m.GOOS == "windows" {
		srcPhpExe := filepath.Join(sourceDir, "php.exe")
		if _, err := os.Stat(srcPhpExe); os.IsNotExist(err) {
			srcPhpExe = filepath.Join(sourceDir, "bin", "php.exe")
		}

		dstPhpExe := filepath.Join(m.BinDir(), "php.exe")
		if _, err := os.Stat(srcPhpExe); err == nil {
			_ = os.Remove(dstPhpExe)
			_ = copyFile(srcPhpExe, dstPhpExe)
		} else {
			_ = os.WriteFile(dstPhpExe, []byte("php"), 0755)
		}

		shimPath := filepath.Join(m.BinDir(), "php.cmd")
		content := fmt.Sprintf("@ECHO off\r\n\"%s\" %%*\r\n", srcPhpExe)
		_ = os.WriteFile(shimPath, []byte(content), 0755)
	} else {
		// Unix: link php, php-cgi, phar
		binNames := []string{"php", "php-cgi", "phar"}
		for _, b := range binNames {
			shimPath := filepath.Join(m.BinDir(), b)
			targetExe := filepath.Join(sourceDir, "bin", b)
			if _, err := os.Stat(targetExe); os.IsNotExist(err) {
				targetExe = filepath.Join(sourceDir, b)
			}

			_ = os.Remove(shimPath)
			if _, err := os.Stat(targetExe); err == nil {
				_ = os.Symlink(targetExe, shimPath)
			}
		}
	}

	fmt.Fprintf(out, "Now using PHP %s\n", version)

	pathEnv := os.Getenv("PATH")
	if !strings.Contains(pathEnv, m.BinDir()) {
		fmt.Fprintf(out, "\nℹ Note: %s is not in your current PATH.\n", m.BinDir())
		if m.GOOS == "windows" {
			fmt.Fprintf(out, "To use php immediately in this terminal, run:\n  $env:Path = \"%s;\" + $env:Path\n", m.BinDir())
		} else {
			fmt.Fprintf(out, "To use php immediately in this terminal, run:\n  export PATH=\"%s:$PATH\"\n", m.BinDir())
		}
	}
	return nil
}

// Current returns the currently active PHP version
func (m *Manager) Current() (string, error) {
	versionStateFile := filepath.Join(filepath.Dir(m.CurrentDir()), "php.version")
	data, err := os.ReadFile(versionStateFile)
	if err != nil {
		return "", fmt.Errorf("no active PHP version selected (run 'uvm use php <version>')")
	}
	return strings.TrimSpace(string(data)), nil
}

// ListInstalled returns all locally installed PHP versions
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

// Remove uninstalls a PHP version (supporting partial prefixes e.g. "8.3")
func (m *Manager) Remove(versionInput string, out io.Writer) error {
	version, err := m.ResolveInstalledVersion(versionInput)
	if err != nil {
		return err
	}

	destDir := filepath.Join(m.VersionsDir(), version)
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		return fmt.Errorf("PHP %s is not installed", version)
	}

	// If removing active version, remove active state and shims
	active, _ := m.Current()
	if active == version {
		_ = os.Remove(filepath.Join(filepath.Dir(m.CurrentDir()), "php.version"))
		_ = os.Remove(m.CurrentDir())
		if m.GOOS == "windows" {
			_ = os.Remove(filepath.Join(m.BinDir(), "php.exe"))
			_ = os.Remove(filepath.Join(m.BinDir(), "php.cmd"))
		} else {
			_ = os.Remove(filepath.Join(m.BinDir(), "php"))
			_ = os.Remove(filepath.Join(m.BinDir(), "php-cgi"))
			_ = os.Remove(filepath.Join(m.BinDir(), "phar"))
		}
	}

	if err := os.RemoveAll(destDir); err != nil {
		return fmt.Errorf("failed to remove PHP %s: %w", version, err)
	}

	fmt.Fprintf(out, "✓ PHP %s removed successfully\n", version)
	return nil
}

// extractTarGz unpacks a .tar.gz archive, stripping top directory if appropriate
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
			f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, header.FileInfo().Mode())
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}
			_ = os.Remove(targetPath)
			if err := os.Symlink(header.Linkname, targetPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// extractZip unpacks a .zip archive to destDir
func extractZip(zipPath string, destDir string) error {
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

		outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}

		if _, err := io.Copy(outFile, rc); err != nil {
			outFile.Close()
			rc.Close()
			return err
		}

		outFile.Close()
		rc.Close()
	}

	return nil
}

func stripTopDir(p string) string {
	parts := strings.Split(filepath.ToSlash(p), "/")
	if len(parts) > 1 && parts[0] != "" {
		return filepath.Join(parts[1:]...)
	}
	return p
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func compareVersions(a, b string) int {
	pa := parseVersionParts(a)
	pb := parseVersionParts(b)

	maxLen := len(pa)
	if len(pb) > maxLen {
		maxLen = len(pb)
	}

	for i := 0; i < maxLen; i++ {
		var va, vb int
		if i < len(pa) {
			va = pa[i]
		}
		if i < len(pb) {
			vb = pb[i]
		}

		if va > vb {
			return 1
		}
		if va < vb {
			return -1
		}
	}

	return 0
}

func parseVersionParts(v string) []int {
	v = strings.TrimPrefix(v, "php")
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "-")
	parts := strings.Split(v, ".")
	var res []int
	for _, p := range parts {
		clean := ""
		for _, r := range p {
			if r >= '0' && r <= '9' {
				clean += string(r)
			} else {
				break
			}
		}
		if n, err := strconv.Atoi(clean); err == nil {
			res = append(res, n)
		} else {
			res = append(res, 0)
		}
	}
	return res
}
