package bun

import (
	"archive/zip"
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

// DefaultBunDistURL is the official GitHub Releases download base for Bun
const DefaultBunDistURL = "https://github.com/oven-sh/bun/releases/download"

// DefaultBunReleasesAPIURL is the GitHub API endpoint for Bun releases
const DefaultBunReleasesAPIURL = "https://api.github.com/repos/oven-sh/bun/releases"

// KnownBunReleases is the list of curated standard Bun releases
var KnownBunReleases = []RemoteVersion{
	{Version: "1.2.4", LTS: "Latest Stable"},
	{Version: "1.2.0"},
	{Version: "1.1.45", LTS: "Stable"},
	{Version: "1.1.0"},
	{Version: "1.0.36"},
	{Version: "1.0.0"},
}

// GitHubRelease matches release tag items from GitHub API
type GitHubRelease struct {
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	Prerelease bool   `json:"prerelease"`
	Draft      bool   `json:"draft"`
}

// RemoteVersion represents a remote Bun release available for installation
type RemoteVersion struct {
	Version string `json:"version"`
	LTS     string `json:"lts,omitempty"`
}

// InstalledVersion represents a locally installed Bun version
type InstalledVersion struct {
	Version  string `json:"version"`
	IsActive bool   `json:"isActive"`
	Path     string `json:"path"`
}

// Manager handles installation, switching, and deletion of Bun runtimes
type Manager struct {
	BaseDir        string
	BunDistURL     string
	ReleasesAPIURL string
	HTTPClient     *http.Client
	GOOS           string
	GOARCH         string
}

// NewManager creates a Bun manager rooted at baseDir (e.g., ~/.uvm)
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
		BunDistURL:     DefaultBunDistURL,
		ReleasesAPIURL: DefaultBunReleasesAPIURL,
		HTTPClient:     http.DefaultClient,
		GOOS:           runtime.GOOS,
		GOARCH:         runtime.GOARCH,
	}
}

// VersionsDir returns the folder where all Bun versions are stored
func (m *Manager) VersionsDir() string {
	return filepath.Join(m.BaseDir, "versions", "bun")
}

// CurrentDir returns the folder for the active runtime symlink
func (m *Manager) CurrentDir() string {
	return filepath.Join(m.BaseDir, "current", "bun")
}

// BinDir returns the active binary directory where bun and bunx shims live
func (m *Manager) BinDir() string {
	return filepath.Join(m.BaseDir, "bin")
}

// NormalizeVersion cleans and normalizes version strings (e.g. bun-v1.2.4 -> 1.2.4, v1.2.4 -> 1.2.4)
func (m *Manager) NormalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	v = strings.TrimPrefix(v, "bun-v")
	v = strings.TrimPrefix(v, "bun-")
	v = strings.TrimPrefix(v, "bun")
	v = strings.TrimPrefix(v, "v")
	return strings.TrimSpace(v)
}

// ListRemote returns available remote Bun releases
func (m *Manager) ListRemote(limit int) ([]RemoteVersion, error) {
	if m.ReleasesAPIURL != "" {
		req, err := http.NewRequest("GET", m.ReleasesAPIURL, nil)
		if err == nil {
			req.Header.Set("User-Agent", "uvm-cli")
			resp, err := m.HTTPClient.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				defer resp.Body.Close()
				var ghReleases []GitHubRelease
				if err := json.NewDecoder(resp.Body).Decode(&ghReleases); err == nil && len(ghReleases) > 0 {
					var list []RemoteVersion
					for i, r := range ghReleases {
						if r.Draft || r.Prerelease {
							continue
						}
						cleanVer := m.NormalizeVersion(r.TagName)
						if cleanVer == "" {
							continue
						}
						lts := ""
						if i == 0 {
							lts = "Latest Stable"
						}
						list = append(list, RemoteVersion{
							Version: cleanVer,
							LTS:     lts,
						})
						if limit > 0 && len(list) >= limit {
							break
						}
					}
					if len(list) > 0 {
						return list, nil
					}
				}
			}
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
		}
	}

	list := KnownBunReleases
	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}
	return list, nil
}

// ResolveInstalledVersion finds the best locally installed version matching input (exact or partial prefix e.g. "1.2")
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

	// 2. Prefix match (e.g. "1.2" matching "1.2.4")
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

// ResolveRemoteVersion resolves alias keywords or partial prefixes (e.g. "1.2", "latest") into concrete semantic versions
func (m *Manager) ResolveRemoteVersion(versionInput string) (string, error) {
	norm := m.NormalizeVersion(versionInput)
	if norm == "" {
		return "", fmt.Errorf("version cannot be empty")
	}

	if norm == "latest" || norm == "current" || norm == "lts" || norm == "stable" {
		return "1.2.4", nil
	}

	// If already a full version (e.g. 1.2.4 with 2 dots), return directly
	if strings.Count(norm, ".") >= 2 {
		return norm, nil
	}

	// Prefix matching against known releases (e.g. "1.2" -> "1.2.4", "1.1" -> "1.1.45")
	prefix := norm
	if !strings.HasSuffix(prefix, ".") {
		prefix = prefix + "."
	}

	for _, rel := range KnownBunReleases {
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

// GetArchiveTarget determines filename, URL, and archive details based on OS/Arch
func (m *Manager) GetArchiveTarget(version string) (downloadURL string, fileName string, err error) {
	var target string

	switch m.GOOS {
	case "darwin":
		if m.GOARCH == "arm64" {
			target = "bun-darwin-aarch64"
		} else {
			target = "bun-darwin-x64"
		}
	case "linux":
		if m.GOARCH == "arm64" {
			target = "bun-linux-aarch64"
		} else {
			target = "bun-linux-x64"
		}
	case "windows":
		target = "bun-windows-x64"
	default:
		return "", "", fmt.Errorf("unsupported OS: %s", m.GOOS)
	}

	fileName = fmt.Sprintf("%s.zip", target)
	downloadURL = fmt.Sprintf("%s/bun-v%s/%s", strings.TrimSuffix(m.BunDistURL, "/"), version, fileName)
	return downloadURL, fileName, nil
}

// Install downloads and extracts a Bun version
func (m *Manager) Install(versionInput string, out io.Writer) error {
	version, err := m.ResolveRemoteVersion(versionInput)
	if err != nil {
		return err
	}

	destDir := filepath.Join(m.VersionsDir(), version)
	if _, err := os.Stat(destDir); err == nil {
		fmt.Fprintf(out, "Bun %s is already installed at %s\n", version, destDir)
		return m.Use(version, out)
	}

	downloadURL, fileName, err := m.GetArchiveTarget(version)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "Downloading Bun %s from %s...\n", version, downloadURL)

	resp, err := m.HTTPClient.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download Bun archive: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download Bun %s (HTTP %d from %s)", version, resp.StatusCode, downloadURL)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create version directory %s: %w", destDir, err)
	}

	fmt.Fprintf(out, "Extracting %s...\n", fileName)

	tmpZip, err := os.CreateTemp("", "bun-*.zip")
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

	fmt.Fprintf(out, "✓ Bun %s installed successfully to %s\n", version, destDir)
	return m.Use(version, out)
}

// LocateBunExe finds the bun executable inside extracted version directory
func (m *Manager) LocateBunExe(versionDir string) string {
	if m.GOOS == "windows" {
		direct := filepath.Join(versionDir, "bun.exe")
		if _, err := os.Stat(direct); err == nil {
			return direct
		}
		inBin := filepath.Join(versionDir, "bin", "bun.exe")
		if _, err := os.Stat(inBin); err == nil {
			return inBin
		}
	} else {
		direct := filepath.Join(versionDir, "bun")
		if _, err := os.Stat(direct); err == nil {
			return direct
		}
		inBin := filepath.Join(versionDir, "bin", "bun")
		if _, err := os.Stat(inBin); err == nil {
			return inBin
		}
	}

	// Check subdirectories
	entries, err := os.ReadDir(versionDir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				cand := filepath.Join(versionDir, e.Name(), "bun")
				if m.GOOS == "windows" {
					cand = filepath.Join(versionDir, e.Name(), "bun.exe")
				}
				if _, err := os.Stat(cand); err == nil {
					return cand
				}
			}
		}
	}

	if m.GOOS == "windows" {
		return filepath.Join(versionDir, "bun.exe")
	}
	return filepath.Join(versionDir, "bun")
}

// Use switches the active Bun version (supporting partial prefixes e.g. "1.2")
func (m *Manager) Use(versionInput string, out io.Writer) error {
	version, err := m.ResolveInstalledVersion(versionInput)
	if err != nil {
		return err
	}

	sourceDir := filepath.Join(m.VersionsDir(), version)
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return fmt.Errorf("Bun %s is not installed. Run 'uvm install bun %s' first", version, version)
	}

	srcBunExe := m.LocateBunExe(sourceDir)

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
	versionStateFile := filepath.Join(currentParent, "bun.version")
	if err := os.WriteFile(versionStateFile, []byte(version), 0644); err != nil {
		return err
	}

	// Create / Update active shims in bin/ (bun and bunx)
	if m.GOOS == "windows" {
		dstBunExe := filepath.Join(m.BinDir(), "bun.exe")
		dstBunxExe := filepath.Join(m.BinDir(), "bunx.exe")
		if _, err := os.Stat(srcBunExe); err == nil {
			_ = os.Remove(dstBunExe)
			_ = copyFile(srcBunExe, dstBunExe)
			_ = os.Remove(dstBunxExe)
			_ = copyFile(srcBunExe, dstBunxExe)
		} else {
			_ = os.WriteFile(dstBunExe, []byte("bun"), 0755)
			_ = os.WriteFile(dstBunxExe, []byte("bunx"), 0755)
		}

		bunCmd := filepath.Join(m.BinDir(), "bun.cmd")
		bunxCmd := filepath.Join(m.BinDir(), "bunx.cmd")
		content := fmt.Sprintf("@ECHO off\r\n\"%s\" %%*\r\n", srcBunExe)
		_ = os.WriteFile(bunCmd, []byte(content), 0755)
		_ = os.WriteFile(bunxCmd, []byte(content), 0755)
	} else {
		// Unix: link bun and bunx
		binNames := []string{"bun", "bunx"}
		for _, b := range binNames {
			shimPath := filepath.Join(m.BinDir(), b)
			_ = os.Remove(shimPath)
			if _, err := os.Stat(srcBunExe); err == nil {
				_ = os.Symlink(srcBunExe, shimPath)
			}
		}
	}

	fmt.Fprintf(out, "Now using Bun %s\n", version)

	pathEnv := os.Getenv("PATH")
	if !strings.Contains(pathEnv, m.BinDir()) {
		fmt.Fprintf(out, "\nℹ Note: %s is not in your current PATH.\n", m.BinDir())
		if m.GOOS == "windows" {
			fmt.Fprintf(out, "To use bun immediately in this terminal, run:\n  $env:Path = \"%s;\" + $env:Path\n", m.BinDir())
		} else {
			fmt.Fprintf(out, "To use bun immediately in this terminal, run:\n  export PATH=\"%s:$PATH\"\n", m.BinDir())
		}
	}
	return nil
}

// Current returns the currently active Bun version
func (m *Manager) Current() (string, error) {
	versionStateFile := filepath.Join(filepath.Dir(m.CurrentDir()), "bun.version")
	data, err := os.ReadFile(versionStateFile)
	if err != nil {
		return "", fmt.Errorf("no active Bun version selected (run 'uvm use bun <version>')")
	}
	return strings.TrimSpace(string(data)), nil
}

// ListInstalled returns all locally installed Bun versions
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

// Remove uninstalls a Bun version (supporting partial prefixes e.g. "1.2")
func (m *Manager) Remove(versionInput string, out io.Writer) error {
	version, err := m.ResolveInstalledVersion(versionInput)
	if err != nil {
		return err
	}

	destDir := filepath.Join(m.VersionsDir(), version)
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		return fmt.Errorf("Bun %s is not installed", version)
	}

	// If removing active version, remove active state and shims
	active, _ := m.Current()
	if active == version {
		_ = os.Remove(filepath.Join(filepath.Dir(m.CurrentDir()), "bun.version"))
		_ = os.Remove(m.CurrentDir())
		if m.GOOS == "windows" {
			_ = os.Remove(filepath.Join(m.BinDir(), "bun.exe"))
			_ = os.Remove(filepath.Join(m.BinDir(), "bunx.exe"))
			_ = os.Remove(filepath.Join(m.BinDir(), "bun.cmd"))
			_ = os.Remove(filepath.Join(m.BinDir(), "bunx.cmd"))
		} else {
			_ = os.Remove(filepath.Join(m.BinDir(), "bun"))
			_ = os.Remove(filepath.Join(m.BinDir(), "bunx"))
		}
	}

	if err := os.RemoveAll(destDir); err != nil {
		return fmt.Errorf("failed to remove Bun %s: %w", version, err)
	}

	fmt.Fprintf(out, "✓ Bun %s removed successfully\n", version)
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
	v = strings.TrimPrefix(v, "bun-v")
	v = strings.TrimPrefix(v, "bun")
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
