package node

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
	"strings"
)

// DefaultNodeDistURL is the official Node.js distribution mirror
const DefaultNodeDistURL = "https://nodejs.org/dist"

// NodeRelease is an entry in the Node.js dist/index.json file
type NodeRelease struct {
	Version string      `json:"version"`
	LTS     interface{} `json:"lts"` // boolean false or string name like "Iron"
}

// InstalledVersion represents a locally installed Node.js version
type InstalledVersion struct {
	Version  string `json:"version"`
	IsActive bool   `json:"isActive"`
	Path     string `json:"path"`
}

// Manager handles installation, switching, and deletion of Node.js runtimes
type Manager struct {
	BaseDir     string
	NodeDistURL string
	HTTPClient  *http.Client
	GOOS        string
	GOARCH      string
}

// NewManager creates a Node manager rooted at baseDir (e.g., ~/.uvm)
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
		BaseDir:     baseDir,
		NodeDistURL: DefaultNodeDistURL,
		HTTPClient:  http.DefaultClient,
		GOOS:        runtime.GOOS,
		GOARCH:      runtime.GOARCH,
	}
}

// VersionsDir returns the folder where all Node versions are stored
func (m *Manager) VersionsDir() string {
	return filepath.Join(m.BaseDir, "versions", "node")
}

// CurrentDir returns the folder for the active runtime symlink
func (m *Manager) CurrentDir() string {
	return filepath.Join(m.BaseDir, "current", "node")
}

// BinDir returns the active binary directory where node, npm, npx shims live
func (m *Manager) BinDir() string {
	return filepath.Join(m.BaseDir, "bin")
}

// NormalizeVersion cleans and ensures standard version format (e.g. 20.11.0 -> v20.11.0)
func (m *Manager) NormalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if !strings.HasPrefix(v, "v") && v != "latest" && v != "current" && v != "lts" {
		return "v" + v
	}
	return v
}

// ResolveVersion resolves alias keywords (latest, lts, current) into concrete semantic versions
func (m *Manager) ResolveVersion(versionInput string) (string, error) {
	norm := m.NormalizeVersion(versionInput)
	if norm == "" {
		return "", fmt.Errorf("version cannot be empty")
	}

	if norm != "latest" && norm != "current" && norm != "lts" {
		return norm, nil
	}

	// Query remote index.json
	reqURL := fmt.Sprintf("%s/index.json", strings.TrimSuffix(m.NodeDistURL, "/"))
	resp, err := m.HTTPClient.Get(reqURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch Node.js version index from %s: %w", reqURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch Node.js versions (HTTP %d)", resp.StatusCode)
	}

	var releases []NodeRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", fmt.Errorf("failed to parse Node.js version index: %w", err)
	}

	if len(releases) == 0 {
		return "", fmt.Errorf("no Node.js releases found")
	}

	if norm == "latest" || norm == "current" {
		return releases[0].Version, nil
	}

	// Find first LTS release
	for _, rel := range releases {
		switch lts := rel.LTS.(type) {
		case bool:
			if lts {
				return rel.Version, nil
			}
		case string:
			if lts != "" {
				return rel.Version, nil
			}
		}
	}

	return releases[0].Version, nil
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
			archName = "x64"
		}
		fileName = fmt.Sprintf("node-%s-%s-%s.tar.gz", version, osName, archName)
		return fileName, false, nil

	case "windows":
		osName = "win"
		if m.GOARCH == "arm64" {
			archName = "arm64"
		} else {
			archName = "x64"
		}
		fileName = fmt.Sprintf("node-%s-%s-%s.zip", version, osName, archName)
		return fileName, true, nil

	case "linux":
		osName = "linux"
		if m.GOARCH == "arm64" {
			archName = "arm64"
		} else if m.GOARCH == "arm" {
			archName = "armv7l"
		} else {
			archName = "x64"
		}
		fileName = fmt.Sprintf("node-%s-%s-%s.tar.gz", version, osName, archName)
		return fileName, false, nil

	default:
		return "", false, fmt.Errorf("unsupported OS: %s", m.GOOS)
	}
}

// Install downloads and extracts a Node.js version
func (m *Manager) Install(versionInput string, out io.Writer) error {
	version, err := m.ResolveVersion(versionInput)
	if err != nil {
		return err
	}

	destDir := filepath.Join(m.VersionsDir(), version)
	if _, err := os.Stat(destDir); err == nil {
		fmt.Fprintf(out, "Node.js %s is already installed at %s\n", version, destDir)
		return nil
	}

	fileName, isZip, err := m.GetArchiveTarget(version)
	if err != nil {
		return err
	}

	downloadURL := fmt.Sprintf("%s/%s/%s", strings.TrimSuffix(m.NodeDistURL, "/"), version, fileName)
	fmt.Fprintf(out, "Downloading Node.js %s from %s...\n", version, downloadURL)

	resp, err := m.HTTPClient.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download Node.js archive: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download Node.js %s (HTTP %d from %s)", version, resp.StatusCode, downloadURL)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create version directory %s: %w", destDir, err)
	}

	fmt.Fprintf(out, "Extracting %s...\n", fileName)

	if isZip {
		// Save to temporary file for zip decompression
		tmpZip, err := os.CreateTemp("", "node-*.zip")
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

	fmt.Fprintf(out, "✓ Node.js %s installed successfully to %s\n", version, destDir)
	return nil
}

// Use switches the active Node.js version
func (m *Manager) Use(versionInput string, out io.Writer) error {
	version, err := m.ResolveVersion(versionInput)
	if err != nil {
		return err
	}

	sourceDir := filepath.Join(m.VersionsDir(), version)
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return fmt.Errorf("Node.js %s is not installed. Run 'uvm install node %s' first", version, version)
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
	versionStateFile := filepath.Join(currentParent, "node.version")
	if err := os.WriteFile(versionStateFile, []byte(version), 0644); err != nil {
		return err
	}

	// Create / Update active shims in bin/
	if m.GOOS == "windows" {
		srcNodeExe := filepath.Join(sourceDir, "node.exe")
		if _, err := os.Stat(srcNodeExe); os.IsNotExist(err) {
			if _, err := os.Stat(filepath.Join(sourceDir, "bin", "node.exe")); err == nil {
				srcNodeExe = filepath.Join(sourceDir, "bin", "node.exe")
			} else if _, err := os.Stat(filepath.Join(sourceDir, "bin", "node")); err == nil {
				srcNodeExe = filepath.Join(sourceDir, "bin", "node")
			}
		}

		dstNodeExe := filepath.Join(m.BinDir(), "node.exe")
		if _, err := os.Stat(srcNodeExe); err == nil {
			_ = os.Remove(dstNodeExe)
			_ = copyFile(srcNodeExe, dstNodeExe)
		} else {
			_ = os.WriteFile(dstNodeExe, []byte("node"), 0755)
		}

		cmdBinaries := []string{"node.cmd", "npm.cmd", "npx.cmd", "corepack.cmd"}
		for _, b := range cmdBinaries {
			shimPath := filepath.Join(m.BinDir(), b)
			targetExe := filepath.Join(sourceDir, b)
			if b == "node.cmd" {
				targetExe = srcNodeExe
			}
			content := fmt.Sprintf("@ECHO off\r\n\"%s\" %%*\r\n", targetExe)
			_ = os.WriteFile(shimPath, []byte(content), 0755)
		}
	} else {
		binaries := []string{"node", "npm", "npx", "corepack"}
		for _, b := range binaries {
			shimPath := filepath.Join(m.BinDir(), b)
			targetExe := filepath.Join(sourceDir, "bin", b)
			_ = os.Remove(shimPath)
			_ = os.Symlink(targetExe, shimPath)
		}
	}

	fmt.Fprintf(out, "Now using Node.js %s\n", version)
	return nil
}

// Current returns the currently active Node.js version
func (m *Manager) Current() (string, error) {
	versionStateFile := filepath.Join(filepath.Dir(m.CurrentDir()), "node.version")
	data, err := os.ReadFile(versionStateFile)
	if err != nil {
		return "", fmt.Errorf("no active Node.js version selected (run 'uvm use node <version>')")
	}
	return strings.TrimSpace(string(data)), nil
}

// ListInstalled returns all locally installed Node.js versions
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

// Remove uninstalls a Node.js version
func (m *Manager) Remove(versionInput string, out io.Writer) error {
	version, err := m.ResolveVersion(versionInput)
	if err != nil {
		return err
	}

	destDir := filepath.Join(m.VersionsDir(), version)
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		return fmt.Errorf("Node.js %s is not installed", version)
	}

	// If removing active version, remove active state and shims
	active, _ := m.Current()
	if active == version {
		_ = os.Remove(filepath.Join(filepath.Dir(m.CurrentDir()), "node.version"))
		_ = os.Remove(m.CurrentDir())
		if m.GOOS == "windows" {
			_ = os.Remove(filepath.Join(m.BinDir(), "node.exe"))
			_ = os.Remove(filepath.Join(m.BinDir(), "npm.cmd"))
			_ = os.Remove(filepath.Join(m.BinDir(), "npx.cmd"))
			_ = os.Remove(filepath.Join(m.BinDir(), "corepack.cmd"))
		} else {
			_ = os.Remove(filepath.Join(m.BinDir(), "node"))
			_ = os.Remove(filepath.Join(m.BinDir(), "npm"))
			_ = os.Remove(filepath.Join(m.BinDir(), "npx"))
			_ = os.Remove(filepath.Join(m.BinDir(), "corepack"))
		}
	}

	if err := os.RemoveAll(destDir); err != nil {
		return fmt.Errorf("failed to remove Node.js %s: %w", version, err)
	}

	fmt.Fprintf(out, "✓ Node.js %s removed successfully\n", version)
	return nil
}

// extractTarGz unpacks a .tar.gz archive, stripping the top-level directory prefix
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

		// Strip top-level directory (e.g. node-v20.11.0-darwin-arm64/bin/node -> bin/node)
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

// stripTopDir strips the first directory element in a path (e.g. node-v20/bin/node -> bin/node)
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

