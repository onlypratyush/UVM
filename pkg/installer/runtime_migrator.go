package installer

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"uvm/pkg/node"
)

// MigrationResult encapsulates the output of a runtime migration.
type MigrationResult struct {
	Success       bool     `json:"success"`
	Runtime       string   `json:"runtime"`
	Version       string   `json:"version"`
	SourceDir     string   `json:"sourceDir"`
	TargetDir     string   `json:"targetDir"`
	PathUpdated   bool     `json:"pathUpdated"`
	OldCleaned    bool     `json:"oldCleaned"`
	WarningNotice string   `json:"warningNotice,omitempty"`
	Steps         []string `json:"steps"`
}

// RuntimeMigrator performs safe migrations, deletions, and rollbacks for detected runtimes.
type RuntimeMigrator struct {
	UVMBaseDir string
	GOOS       string
	PathMgr    PathManager
	RunCmd     func(name string, args ...string) (string, error)
	Log        func(msg string)
}

// NewRuntimeMigrator creates a new migrator instance.
func NewRuntimeMigrator(uvmBaseDir string, goos string, pathMgr PathManager) *RuntimeMigrator {
	if goos == "" {
		goos = runtime.GOOS
	}
	if uvmBaseDir == "" {
		home, _ := os.UserHomeDir()
		uvmBaseDir = filepath.Join(home, ".uvm")
	}

	return &RuntimeMigrator{
		UVMBaseDir: uvmBaseDir,
		GOOS:       goos,
		PathMgr:    pathMgr,
		RunCmd: func(name string, args ...string) (string, error) {
			cmd := exec.Command(name, args...)
			out, err := cmd.Output()
			return strings.TrimSpace(string(out)), err
		},
		Log: func(msg string) {},
	}
}

// MigrateNode executes the safe migration sequence for an existing Node.js installation.
func (m *RuntimeMigrator) MigrateNode(detected DetectedRuntime) (*MigrationResult, error) {
	if !detected.Found {
		return nil, fmt.Errorf("no Node.js installation detected to migrate")
	}

	res := &MigrationResult{
		Runtime:   "node",
		Version:   detected.Version,
		SourceDir: detected.InstallDir,
		Steps:     []string{},
	}

	m.Log(fmt.Sprintf("Starting safe migration of Node.js %s from %s...", detected.Version, detected.InstallDir))

	// Determine target version directory in UVM
	targetDir := filepath.Join(m.UVMBaseDir, "versions", "node", detected.Version)
	res.TargetDir = targetDir

	// STEP 1: Copy Node installation into UVM versions directory
	m.Log(fmt.Sprintf("[1/6] Copying existing installation to %s...", targetDir))
	res.Steps = append(res.Steps, fmt.Sprintf("Copying from %s to %s", detected.InstallDir, targetDir))

	if err := CopyDir(detected.InstallDir, targetDir); err != nil {
		_ = os.RemoveAll(targetDir)
		return res, fmt.Errorf("failed to copy Node installation to %s: %w", targetDir, err)
	}

	// STEP 2: Verify copied binary exists and probe version
	m.Log("[2/6] Verifying copied node executable...")
	res.Steps = append(res.Steps, "Verifying copied executable")

	copiedBin := filepath.Join(targetDir, "node.exe")
	if m.GOOS != "windows" {
		copiedBin = filepath.Join(targetDir, "bin", "node")
		if _, err := os.Stat(copiedBin); os.IsNotExist(err) {
			copiedBin = filepath.Join(targetDir, "node")
		}
	}

	if _, err := os.Stat(copiedBin); err != nil {
		_ = os.RemoveAll(targetDir)
		return res, fmt.Errorf("copied node executable not found at %s", copiedBin)
	}

	probeVer, err := m.RunCmd(copiedBin, "-v")
	if err != nil || probeVer != detected.Version {
		_ = os.RemoveAll(targetDir)
		return res, fmt.Errorf("copied node verification failed (expected %s, got %s, err: %v)", detected.Version, probeVer, err)
	}

	// STEP 3: Save previous PATH for rollback safety and update PATH
	m.Log("[3/6] Configuring PATH environment...")
	res.Steps = append(res.Steps, "Updating PATH")

	prevPath, _ := m.PathMgr.GetPath()

	// Add UVM bin to PATH
	uvmBin := filepath.Join(m.UVMBaseDir, "bin")
	if err := m.PathMgr.AddEntry(uvmBin); err != nil {
		_ = os.RemoveAll(targetDir)
		return res, fmt.Errorf("failed to add UVM bin to PATH: %w", err)
	}

	// Remove old Node PATH entries
	if len(detected.PathEntries) > 0 {
		if err := m.PathMgr.RemoveEntries(detected.PathEntries); err != nil {
			// Rollback PATH
			_ = m.PathMgr.SetPath(prevPath)
			_ = os.RemoveAll(targetDir)
			return res, fmt.Errorf("failed to remove old Node entries from PATH: %w", err)
		}
	}
	res.PathUpdated = true

	// STEP 4: Switch UVM active version to migrated version
	m.Log("[4/6] Activating migrated version in UVM...")
	res.Steps = append(res.Steps, "Activating version in UVM")

	nodeMgr := node.NewManager(m.UVMBaseDir)
	nodeMgr.GOOS = m.GOOS
	if err := nodeMgr.Use(detected.Version, io.Discard); err != nil {
		// Rollback PATH and directory
		_ = m.PathMgr.SetPath(prevPath)
		_ = os.RemoveAll(targetDir)
		return res, fmt.Errorf("failed to activate Node %s in UVM: %w", detected.Version, err)
	}

	// STEP 5: Verify UVM-controlled Node executable
	m.Log("[5/6] Verifying UVM-controlled Node execution...")
	res.Steps = append(res.Steps, "Verifying UVM-controlled binary")

	uvmNodeExe := filepath.Join(uvmBin, "node.exe")
	if m.GOOS != "windows" {
		uvmNodeExe = filepath.Join(uvmBin, "node")
	}

	uvmProbe, err := m.RunCmd(uvmNodeExe, "-v")
	if err != nil || uvmProbe != detected.Version {
		// Rollback
		_ = m.PathMgr.SetPath(prevPath)
		_ = os.RemoveAll(targetDir)
		return res, fmt.Errorf("UVM active node verification failed (expected %s, got %s, err: %v)", detected.Version, uvmProbe, err)
	}

	// STEP 6: Safely remove old installation directory
	m.Log("[6/6] Cleaning up previous installation...")
	res.Steps = append(res.Steps, "Cleaning up old installation")

	oldCleaned := true
	if err := os.RemoveAll(detected.InstallDir); err != nil {
		oldCleaned = false
		res.WarningNotice = fmt.Sprintf("Note: Could not automatically remove %s (%v). You may delete it manually.", detected.InstallDir, err)
		m.Log(res.WarningNotice)
	}
	if detected.NPMPath != "" && detected.NPMPath != detected.InstallDir {
		_ = os.RemoveAll(detected.NPMPath)
	}
	res.OldCleaned = oldCleaned

	res.Success = true
	m.Log(fmt.Sprintf("✓ Successfully migrated Node.js %s to UVM!", detected.Version))
	return res, nil
}

// DeleteExistingNode removes the old Node.js installation with explicit confirmation.
func (m *RuntimeMigrator) DeleteExistingNode(detected DetectedRuntime, confirmed bool) error {
	if !confirmed {
		return fmt.Errorf("deletion requires explicit user confirmation")
	}

	if !detected.Found {
		return nil
	}

	m.Log(fmt.Sprintf("Removing existing Node.js installation at %s...", detected.InstallDir))

	// Remove old PATH entries
	if len(detected.PathEntries) > 0 {
		_ = m.PathMgr.RemoveEntries(detected.PathEntries)
	}

	// Add UVM bin to PATH
	uvmBin := filepath.Join(m.UVMBaseDir, "bin")
	_ = m.PathMgr.AddEntry(uvmBin)

	// Remove old directory
	_ = os.RemoveAll(detected.InstallDir)
	if detected.NPMPath != "" && detected.NPMPath != detected.InstallDir {
		_ = os.RemoveAll(detected.NPMPath)
	}

	m.Log("✓ Existing Node.js installation removed.")
	return nil
}

// CopyDir recursively copies a directory tree from src to dst.
func CopyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !srcInfo.IsDir() {
		// Single file copy
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return err
		}
		return copyFile(src, dst)
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := CopyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}
