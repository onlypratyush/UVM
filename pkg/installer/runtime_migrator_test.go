package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type MockPathManager struct {
	CurrentPath string
	AddHistory  []string
	RemHistory  []string
}

func (m *MockPathManager) GetPath() (string, error) {
	return m.CurrentPath, nil
}

func (m *MockPathManager) SetPath(newPath string) error {
	m.CurrentPath = newPath
	return nil
}

func (m *MockPathManager) AddEntry(entry string) error {
	m.AddHistory = append(m.AddHistory, entry)
	entries := strings.Split(m.CurrentPath, ";")
	updated := AddPathEntry(entries, entry, "windows")
	m.CurrentPath = strings.Join(updated, ";")
	return nil
}

func (m *MockPathManager) RemoveEntries(entries []string) error {
	m.RemHistory = append(m.RemHistory, entries...)
	curEntries := strings.Split(m.CurrentPath, ";")
	filtered := FilterPathList(curEntries, entries, "windows")
	m.CurrentPath = strings.Join(filtered, ";")
	return nil
}

func (m *MockPathManager) BroadcastChange() error {
	return nil
}

func TestMigrateNodeWindowsSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	sourceNodeDir := filepath.Join(tmpDir, "Program Files", "nodejs")
	_ = os.MkdirAll(sourceNodeDir, 0755)
	_ = os.WriteFile(filepath.Join(sourceNodeDir, "node.exe"), []byte("node-bin"), 0755)
	_ = os.WriteFile(filepath.Join(sourceNodeDir, "npm.cmd"), []byte("npm-cmd"), 0755)

	uvmBase := filepath.Join(tmpDir, ".uvm")
	pathMgr := &MockPathManager{
		CurrentPath: fmt.Sprintf(`C:\Windows\System32;%s`, sourceNodeDir),
	}

	migrator := NewRuntimeMigrator(uvmBase, "windows", pathMgr)
	migrator.RunCmd = func(name string, args ...string) (string, error) {
		if len(args) > 0 && args[0] == "-v" {
			return "v22.18.0", nil
		}
		return "", fmt.Errorf("command failed")
	}

	var logMsgs []string
	migrator.Log = func(msg string) {
		logMsgs = append(logMsgs, msg)
	}

	detected := DetectedRuntime{
		Name:           "node",
		Found:          true,
		Version:        "v22.18.0",
		ExecutablePath: filepath.Join(sourceNodeDir, "node.exe"),
		InstallDir:     sourceNodeDir,
		PathEntries:    []string{sourceNodeDir},
	}

	res, err := migrator.MigrateNode(detected)
	if err != nil {
		t.Fatalf("MigrateNode failed: %v", err)
	}

	if !res.Success {
		t.Errorf("expected migration success, got: %+v", res)
	}

	// Target directory should have files
	targetNode := filepath.Join(uvmBase, "versions", "node", "v22.18.0", "node.exe")
	if _, err := os.Stat(targetNode); err != nil {
		t.Errorf("expected copied node.exe at %s", targetNode)
	}

	// UVM bin should have active node.exe binary
	uvmBinNode := filepath.Join(uvmBase, "bin", "node.exe")
	if _, err := os.Stat(uvmBinNode); err != nil {
		t.Errorf("expected active node.exe in uvm bin at %s", uvmBinNode)
	}

	// PATH should contain uvm/bin and not sourceNodeDir
	if strings.Contains(pathMgr.CurrentPath, sourceNodeDir) {
		t.Errorf("expected sourceNodeDir removed from PATH: %s", pathMgr.CurrentPath)
	}
	if !strings.Contains(pathMgr.CurrentPath, filepath.Join(uvmBase, "bin")) {
		t.Errorf("expected uvm/bin in PATH: %s", pathMgr.CurrentPath)
	}
}

func TestMigrateNodeRollbackOnVerificationFailure(t *testing.T) {
	tmpDir := t.TempDir()
	sourceNodeDir := filepath.Join(tmpDir, "custom_node")
	_ = os.MkdirAll(sourceNodeDir, 0755)
	_ = os.WriteFile(filepath.Join(sourceNodeDir, "node.exe"), []byte("node-bin"), 0755)

	uvmBase := filepath.Join(tmpDir, ".uvm")
	originalPath := fmt.Sprintf(`C:\Windows\System32;%s`, sourceNodeDir)
	pathMgr := &MockPathManager{CurrentPath: originalPath}

	migrator := NewRuntimeMigrator(uvmBase, "windows", pathMgr)
	// Return mismatch version to simulate corrupted copy
	migrator.RunCmd = func(name string, args ...string) (string, error) {
		return "v18.0.0", nil // Mismatch from v22.18.0
	}

	detected := DetectedRuntime{
		Name:           "node",
		Found:          true,
		Version:        "v22.18.0",
		ExecutablePath: filepath.Join(sourceNodeDir, "node.exe"),
		InstallDir:     sourceNodeDir,
		PathEntries:    []string{sourceNodeDir},
	}

	_, err := migrator.MigrateNode(detected)
	if err == nil {
		t.Fatalf("expected migration to fail and rollback")
	}

	// Original directory should remain intact
	if _, err := os.Stat(sourceNodeDir); err != nil {
		t.Errorf("expected original directory preserved after rollback")
	}

	// PATH should remain original
	if pathMgr.CurrentPath != originalPath {
		t.Errorf("expected PATH restored to original, got: %s", pathMgr.CurrentPath)
	}

	// Copied directory in UVM should be cleaned
	targetDir := filepath.Join(uvmBase, "versions", "node", "v22.18.0")
	if _, err := os.Stat(targetDir); !os.IsNotExist(err) {
		t.Errorf("expected target directory removed after rollback")
	}
}

func TestDeleteExistingNode(t *testing.T) {
	tmpDir := t.TempDir()
	sourceNodeDir := filepath.Join(tmpDir, "node_to_delete")
	_ = os.MkdirAll(sourceNodeDir, 0755)
	_ = os.WriteFile(filepath.Join(sourceNodeDir, "node.exe"), []byte("data"), 0755)

	pathMgr := &MockPathManager{CurrentPath: fmt.Sprintf("C:\\System;%s", sourceNodeDir)}
	migrator := NewRuntimeMigrator(filepath.Join(tmpDir, ".uvm"), "windows", pathMgr)

	detected := DetectedRuntime{
		Name:        "node",
		Found:       true,
		InstallDir:  sourceNodeDir,
		PathEntries: []string{sourceNodeDir},
	}

	// 1. Fail without confirmation
	err := migrator.DeleteExistingNode(detected, false)
	if err == nil {
		t.Errorf("expected error deleting without confirmation")
	}

	// 2. Succeed with confirmation
	err = migrator.DeleteExistingNode(detected, true)
	if err != nil {
		t.Fatalf("DeleteExistingNode failed: %v", err)
	}

	if _, err := os.Stat(sourceNodeDir); !os.IsNotExist(err) {
		t.Errorf("expected source directory to be deleted")
	}
}

func TestCopyDir(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src")
	dst := filepath.Join(tmpDir, "dst")

	_ = os.MkdirAll(filepath.Join(src, "sub"), 0755)
	_ = os.WriteFile(filepath.Join(src, "file1.txt"), []byte("hello"), 0644)
	_ = os.WriteFile(filepath.Join(src, "sub", "file2.txt"), []byte("world"), 0644)

	if err := CopyDir(src, dst); err != nil {
		t.Fatalf("CopyDir failed: %v", err)
	}

	f1, _ := os.ReadFile(filepath.Join(dst, "file1.txt"))
	if string(f1) != "hello" {
		t.Errorf("unexpected file1 content: %s", string(f1))
	}

	f2, _ := os.ReadFile(filepath.Join(dst, "sub", "file2.txt"))
	if string(f2) != "world" {
		t.Errorf("unexpected file2 content: %s", string(f2))
	}

	// Copy single file
	singleSrc := filepath.Join(tmpDir, "single.txt")
	singleDst := filepath.Join(tmpDir, "single_dst.txt")
	_ = os.WriteFile(singleSrc, []byte("single"), 0644)
	if err := CopyDir(singleSrc, singleDst); err != nil {
		t.Fatalf("CopyDir single file failed: %v", err)
	}
}
