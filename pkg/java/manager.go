package java

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

// DefaultAdoptiumAPIURL is the official Adoptium API v3 endpoint for Eclipse Temurin JDKs
const DefaultAdoptiumAPIURL = "https://api.adoptium.net/v3"

// KnownJavaReleases is the list of standard supported OpenJDK / Temurin versions
var KnownJavaReleases = []RemoteVersion{
	{Version: "23.0.2", Feature: 23, LTS: "Latest Stable"},
	{Version: "21.0.6", Feature: 21, LTS: "LTS"},
	{Version: "17.0.14", Feature: 17, LTS: "LTS"},
	{Version: "11.0.26", Feature: 11, LTS: "LTS"},
	{Version: "8.0.442", Feature: 8, LTS: "LTS"},
}

// AvailableReleasesResponse matches Adoptium /v3/info/available_releases
type AvailableReleasesResponse struct {
	AvailableLTSReleases     []int `json:"available_lts_releases"`
	AvailableReleases        []int `json:"available_releases"`
	MostRecentFeatureRelease int   `json:"most_recent_feature_release"`
	MostRecentLTS            int   `json:"most_recent_lts"`
}

// AdoptiumBinary matches binary entries in feature release assets
type AdoptiumBinary struct {
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	ImageType    string `json:"image_type"`
	Package      struct {
		Name     string `json:"name"`
		Link     string `json:"link"`
		Checksum string `json:"checksum"`
	} `json:"package"`
}

// AdoptiumRelease matches release items in /v3/assets/feature_releases
type AdoptiumRelease struct {
	ReleaseName string           `json:"release_name"`
	VersionData struct {
		Semver string `json:"semver"`
		OpenJdkVersion string `json:"openjdk_version"`
	} `json:"version_data"`
	Binaries []AdoptiumBinary `json:"binaries"`
}

// RemoteVersion represents a remote Java release available for installation
type RemoteVersion struct {
	Version string `json:"version"`
	Feature int    `json:"feature,omitempty"`
	LTS     string `json:"lts,omitempty"`
}

// InstalledVersion represents a locally installed Java version
type InstalledVersion struct {
	Version  string `json:"version"`
	IsActive bool   `json:"isActive"`
	Path     string `json:"path"`
}

// Manager handles installation, switching, and deletion of Java (JDK) runtimes
type Manager struct {
	BaseDir        string
	AdoptiumAPIURL string
	HTTPClient     *http.Client
	GOOS           string
	GOARCH         string
}

// NewManager creates a Java manager rooted at baseDir (e.g., ~/.uvm)
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
		AdoptiumAPIURL: DefaultAdoptiumAPIURL,
		HTTPClient:     http.DefaultClient,
		GOOS:           runtime.GOOS,
		GOARCH:         runtime.GOARCH,
	}
}

// VersionsDir returns the folder where all Java versions are stored
func (m *Manager) VersionsDir() string {
	return filepath.Join(m.BaseDir, "versions", "java")
}

// CurrentDir returns the folder for the active runtime symlink
func (m *Manager) CurrentDir() string {
	return filepath.Join(m.BaseDir, "current", "java")
}

// BinDir returns the active binary directory where java, javac, jar shims live
func (m *Manager) BinDir() string {
	return filepath.Join(m.BaseDir, "bin")
}

// NormalizeVersion cleans version strings (e.g. jdk-21.0.6 -> 21.0.6, java21 -> 21, 1.8 -> 8)
func (m *Manager) NormalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	v = strings.TrimPrefix(v, "openjdk-")
	v = strings.TrimPrefix(v, "openjdk")
	v = strings.TrimPrefix(v, "jdk-")
	v = strings.TrimPrefix(v, "jdk")
	v = strings.TrimPrefix(v, "java-")
	v = strings.TrimPrefix(v, "java")
	v = strings.TrimPrefix(v, "v")

	if v == "1.8" || v == "1.8.0" {
		return "8"
	}
	return strings.TrimSpace(v)
}

// ListRemote returns available remote Java releases
func (m *Manager) ListRemote(limit int) ([]RemoteVersion, error) {
	if m.AdoptiumAPIURL != "" {
		reqURL := fmt.Sprintf("%s/info/available_releases", strings.TrimSuffix(m.AdoptiumAPIURL, "/"))
		resp, err := m.HTTPClient.Get(reqURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			var avail AvailableReleasesResponse
			if err := json.NewDecoder(resp.Body).Decode(&avail); err == nil && len(avail.AvailableReleases) > 0 {
				var list []RemoteVersion
				ltsSet := make(map[int]bool)
				for _, l := range avail.AvailableLTSReleases {
					ltsSet[l] = true
				}

				// Sort descending
				sort.Slice(avail.AvailableReleases, func(i, j int) bool {
					return avail.AvailableReleases[i] > avail.AvailableReleases[j]
				})

				for _, feat := range avail.AvailableReleases {
					lts := ""
					if ltsSet[feat] {
						lts = "LTS"
					} else if feat == avail.MostRecentFeatureRelease {
						lts = "Latest Feature"
					}

					// Find concrete version string if available in known releases
					verStr := fmt.Sprintf("%d", feat)
					for _, k := range KnownJavaReleases {
						if k.Feature == feat {
							verStr = k.Version
							break
						}
					}

					list = append(list, RemoteVersion{
						Version: verStr,
						Feature: feat,
						LTS:     lts,
					})
					if limit > 0 && len(list) >= limit {
						break
					}
				}
				return list, nil
			}
		}
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}

	list := KnownJavaReleases
	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}
	return list, nil
}

// ResolveInstalledVersion finds the best locally installed version matching input (exact or partial prefix e.g. "21")
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

	// 2. Prefix match (e.g. "21" matching "21.0.6")
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

// ResolveRemoteVersion resolves alias keywords or feature versions (e.g. "21", "latest") into concrete semantic versions
func (m *Manager) ResolveRemoteVersion(versionInput string) (string, error) {
	norm := m.NormalizeVersion(versionInput)
	if norm == "" {
		return "", fmt.Errorf("version cannot be empty")
	}

	if norm == "latest" || norm == "current" {
		return "23.0.2", nil
	}
	if norm == "lts" || norm == "stable" {
		return "21.0.6", nil
	}

	// If already a full version (e.g. 21.0.6 with 2 dots), return directly
	if strings.Count(norm, ".") >= 2 {
		return norm, nil
	}

	// Prefix matching against known releases (e.g. "21" -> "21.0.6", "17" -> "17.0.14")
	prefix := norm
	if !strings.HasSuffix(prefix, ".") {
		prefix = prefix + "."
	}

	for _, rel := range KnownJavaReleases {
		if strings.HasPrefix(rel.Version, prefix) || rel.Version == norm || fmt.Sprintf("%d", rel.Feature) == norm {
			return rel.Version, nil
		}
	}

	return norm, nil
}

// ResolveVersion resolves alias keywords into concrete versions
func (m *Manager) ResolveVersion(versionInput string) (string, error) {
	return m.ResolveRemoteVersion(versionInput)
}

// ExtractFeatureVersion parses the major feature number from a semver (e.g. 21.0.6 -> 21, 8.0.442 -> 8)
func ExtractFeatureVersion(v string) int {
	parts := strings.Split(v, ".")
	if len(parts) > 0 {
		if n, err := strconv.Atoi(parts[0]); err == nil {
			return n
		}
	}
	return 21
}

// GetArchiveTarget determines download URL, filename, and archive format based on OS/Arch
func (m *Manager) GetArchiveTarget(version string) (downloadURL string, fileName string, isZip bool, err error) {
	var osStr, archStr string

	switch m.GOOS {
	case "darwin":
		osStr = "mac"
		if m.GOARCH == "arm64" {
			archStr = "aarch64"
		} else {
			archStr = "x64"
		}
		isZip = false
	case "linux":
		osStr = "linux"
		if m.GOARCH == "arm64" {
			archStr = "aarch64"
		} else if m.GOARCH == "arm" {
			archStr = "arm"
		} else {
			archStr = "x64"
		}
		isZip = false
	case "windows":
		osStr = "windows"
		if m.GOARCH == "arm64" {
			archStr = "aarch64"
		} else if m.GOARCH == "386" {
			archStr = "x86"
		} else {
			archStr = "x64"
		}
		isZip = true
	default:
		return "", "", false, fmt.Errorf("unsupported OS: %s", m.GOOS)
	}

	feat := ExtractFeatureVersion(version)
	downloadURL = fmt.Sprintf("%s/binary/latest/%d/ga/%s/%s/jdk/hotspot/normal/eclipse", strings.TrimSuffix(m.AdoptiumAPIURL, "/"), feat, osStr, archStr)

	ext := "tar.gz"
	if isZip {
		ext = "zip"
	}
	fileName = fmt.Sprintf("OpenJDK%dU-jdk_%s_%s_hotspot_%s.%s", feat, archStr, osStr, version, ext)
	return downloadURL, fileName, isZip, nil
}

// Install downloads and extracts a Java (JDK) version
func (m *Manager) Install(versionInput string, out io.Writer) error {
	version, err := m.ResolveRemoteVersion(versionInput)
	if err != nil {
		return err
	}

	destDir := filepath.Join(m.VersionsDir(), version)
	if _, err := os.Stat(destDir); err == nil {
		fmt.Fprintf(out, "Java %s is already installed at %s\n", version, destDir)
		return m.Use(version, out)
	}

	downloadURL, fileName, isZip, err := m.GetArchiveTarget(version)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "Downloading Java (Eclipse Temurin) %s from %s...\n", version, downloadURL)

	resp, err := m.HTTPClient.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download Java archive: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download Java %s (HTTP %d from %s)", version, resp.StatusCode, downloadURL)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create version directory %s: %w", destDir, err)
	}

	fmt.Fprintf(out, "Extracting %s...\n", fileName)

	if isZip {
		tmpZip, err := os.CreateTemp("", "java-*.zip")
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

	fmt.Fprintf(out, "✓ Java %s installed successfully to %s\n", version, destDir)
	return m.Use(version, out)
}

// LocateJavaHome finds the true JAVA_HOME directory inside an extracted JDK (accounting for macOS Contents/Home)
func (m *Manager) LocateJavaHome(versionDir string) string {
	// 1. Check for macOS bundle structure (Contents/Home)
	macHome := filepath.Join(versionDir, "Contents", "Home")
	if _, err := os.Stat(filepath.Join(macHome, "bin")); err == nil {
		return macHome
	}

	// Check one level deep for macOS bundle structure (e.g. jdk-21.0.6+7/Contents/Home)
	entries, err := os.ReadDir(versionDir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				nestedMacHome := filepath.Join(versionDir, e.Name(), "Contents", "Home")
				if _, err := os.Stat(filepath.Join(nestedMacHome, "bin")); err == nil {
					return nestedMacHome
				}
				nestedBin := filepath.Join(versionDir, e.Name(), "bin")
				if _, err := os.Stat(nestedBin); err == nil {
					return filepath.Join(versionDir, e.Name())
				}
			}
		}
	}

	return versionDir
}

// Use switches the active Java version (supporting partial prefixes e.g. "21")
func (m *Manager) Use(versionInput string, out io.Writer) error {
	version, err := m.ResolveInstalledVersion(versionInput)
	if err != nil {
		return err
	}

	sourceDir := filepath.Join(m.VersionsDir(), version)
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return fmt.Errorf("Java %s is not installed. Run 'uvm install java %s' first", version, version)
	}

	javaHome := m.LocateJavaHome(sourceDir)

	if err := os.MkdirAll(m.BinDir(), 0755); err != nil {
		return err
	}

	currentParent := filepath.Dir(m.CurrentDir())
	if err := os.MkdirAll(currentParent, 0755); err != nil {
		return err
	}

	// Update current active link to the effective JAVA_HOME
	_ = os.Remove(m.CurrentDir())
	_ = os.Symlink(javaHome, m.CurrentDir())

	// Write active version state file
	versionStateFile := filepath.Join(currentParent, "java.version")
	if err := os.WriteFile(versionStateFile, []byte(version), 0644); err != nil {
		return err
	}

	// Create / Update active shims in bin/
	binDir := filepath.Join(javaHome, "bin")
	if m.GOOS == "windows" {
		winBinaries := []string{"java", "javac", "jar", "javadoc", "jshell"}
		for _, b := range winBinaries {
			srcExe := filepath.Join(binDir, b+".exe")
			dstExe := filepath.Join(m.BinDir(), b+".exe")
			if _, err := os.Stat(srcExe); err == nil {
				_ = os.Remove(dstExe)
				_ = copyFile(srcExe, dstExe)
			} else {
				_ = os.WriteFile(dstExe, []byte(b), 0755)
			}

			shimCmd := filepath.Join(m.BinDir(), b+".cmd")
			content := fmt.Sprintf("@ECHO off\r\n\"%s\" %%*\r\n", srcExe)
			_ = os.WriteFile(shimCmd, []byte(content), 0755)
		}
	} else {
		// Unix: link java, javac, jar, javadoc, jshell
		binNames := []string{"java", "javac", "jar", "javadoc", "jshell"}
		for _, b := range binNames {
			shimPath := filepath.Join(m.BinDir(), b)
			targetExe := filepath.Join(binDir, b)

			_ = os.Remove(shimPath)
			if _, err := os.Stat(targetExe); err == nil {
				_ = os.Symlink(targetExe, shimPath)
			}
		}
	}

	fmt.Fprintf(out, "Now using Java %s\n", version)

	pathEnv := os.Getenv("PATH")
	if !strings.Contains(pathEnv, m.BinDir()) {
		fmt.Fprintf(out, "\nℹ Note: %s is not in your current PATH.\n", m.BinDir())
		if m.GOOS == "windows" {
			fmt.Fprintf(out, "To use java immediately in this terminal, run:\n  $env:JAVA_HOME = \"%s\"\n  $env:Path = \"%s;\" + $env:Path\n", javaHome, m.BinDir())
		} else {
			fmt.Fprintf(out, "To use java immediately in this terminal, run:\n  export JAVA_HOME=\"%s\"\n  export PATH=\"%s:$PATH\"\n", javaHome, m.BinDir())
		}
	}
	return nil
}

// Current returns the currently active Java version
func (m *Manager) Current() (string, error) {
	versionStateFile := filepath.Join(filepath.Dir(m.CurrentDir()), "java.version")
	data, err := os.ReadFile(versionStateFile)
	if err != nil {
		return "", fmt.Errorf("no active Java version selected (run 'uvm use java <version>')")
	}
	return strings.TrimSpace(string(data)), nil
}

// ListInstalled returns all locally installed Java versions
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

// Remove uninstalls a Java version (supporting partial prefixes e.g. "21")
func (m *Manager) Remove(versionInput string, out io.Writer) error {
	version, err := m.ResolveInstalledVersion(versionInput)
	if err != nil {
		return err
	}

	destDir := filepath.Join(m.VersionsDir(), version)
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		return fmt.Errorf("Java %s is not installed", version)
	}

	// If removing active version, remove active state and shims
	active, _ := m.Current()
	if active == version {
		_ = os.Remove(filepath.Join(filepath.Dir(m.CurrentDir()), "java.version"))
		_ = os.Remove(m.CurrentDir())
		binNames := []string{"java", "javac", "jar", "javadoc", "jshell"}
		for _, b := range binNames {
			if m.GOOS == "windows" {
				_ = os.Remove(filepath.Join(m.BinDir(), b+".exe"))
				_ = os.Remove(filepath.Join(m.BinDir(), b+".cmd"))
			} else {
				_ = os.Remove(filepath.Join(m.BinDir(), b))
			}
		}
	}

	if err := os.RemoveAll(destDir); err != nil {
		return fmt.Errorf("failed to remove Java %s: %w", version, err)
	}

	fmt.Fprintf(out, "✓ Java %s removed successfully\n", version)
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
	v = strings.TrimPrefix(v, "openjdk")
	v = strings.TrimPrefix(v, "jdk")
	v = strings.TrimPrefix(v, "java")
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
