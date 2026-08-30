package scaffold

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScaffoldNodeExpressTS(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "my-express-ts")

	mgr := NewManager()
	out := new(bytes.Buffer)

	err := mgr.Scaffold(ProjectOptions{
		ProjectName: "my-express-ts",
		TargetDir:   target,
		Language:    "node",
		Dialect:     "ts",
		Framework:   "express",
		IncludeCRUD: true,
		Version:     "20.11.0",
	}, out)

	if err != nil {
		t.Fatalf("Scaffold failed: %v", err)
	}

	// Assert files exist
	assertFileExists(t, filepath.Join(target, "package.json"))
	assertFileExists(t, filepath.Join(target, "tsconfig.json"))
	assertFileExists(t, filepath.Join(target, "src", "app.ts"))
	assertFileExists(t, filepath.Join(target, "src", "controllers", "itemController.ts"))
	assertFileExists(t, filepath.Join(target, ".uvmrc"))

	// Verify .uvmrc content
	data, _ := os.ReadFile(filepath.Join(target, ".uvmrc"))
	if !strings.Contains(string(data), "node 20.11.0") {
		t.Errorf("expected .uvmrc to contain 'node 20.11.0', got: %s", string(data))
	}
}

func TestScaffoldNodeExpressJS(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "my-express-js")

	mgr := NewManager()
	out := new(bytes.Buffer)

	err := mgr.Scaffold(ProjectOptions{
		ProjectName: "my-express-js",
		TargetDir:   target,
		Language:    "node",
		Dialect:     "js",
		Framework:   "express",
	}, out)

	if err != nil {
		t.Fatalf("Scaffold failed: %v", err)
	}

	assertFileExists(t, filepath.Join(target, "package.json"))
	assertFileExists(t, filepath.Join(target, "src", "app.js"))
	assertFileExists(t, filepath.Join(target, "src", "controllers", "itemController.js"))
}

func TestScaffoldNodeFastifyTS(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "my-fastify-ts")

	mgr := NewManager()
	out := new(bytes.Buffer)

	err := mgr.Scaffold(ProjectOptions{
		ProjectName: "my-fastify-ts",
		TargetDir:   target,
		Language:    "node",
		Dialect:     "ts",
		Framework:   "fastify",
	}, out)

	if err != nil {
		t.Fatalf("Scaffold failed: %v", err)
	}

	assertFileExists(t, filepath.Join(target, "src", "server.ts"))
	assertFileExists(t, filepath.Join(target, "src", "schemas", "itemSchema.ts"))
}

func TestScaffoldNodeFastifyJS(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "my-fastify-js")

	mgr := NewManager()
	out := new(bytes.Buffer)

	err := mgr.Scaffold(ProjectOptions{
		ProjectName: "my-fastify-js",
		TargetDir:   target,
		Language:    "node",
		Dialect:     "js",
		Framework:   "fastify",
	}, out)

	if err != nil {
		t.Fatalf("Scaffold failed: %v", err)
	}

	assertFileExists(t, filepath.Join(target, "src", "server.js"))
	assertFileExists(t, filepath.Join(target, "src", "routes", "itemRoutes.js"))
}

func TestScaffoldGoGin(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "my-go-gin")

	mgr := NewManager()
	out := new(bytes.Buffer)

	err := mgr.Scaffold(ProjectOptions{
		ProjectName: "my-go-gin",
		TargetDir:   target,
		Language:    "go",
		Framework:   "gin",
		Version:     "1.22.6",
	}, out)

	if err != nil {
		t.Fatalf("Scaffold failed: %v", err)
	}

	assertFileExists(t, filepath.Join(target, "go.mod"))
	assertFileExists(t, filepath.Join(target, "cmd", "server", "main.go"))
	assertFileExists(t, filepath.Join(target, "internal", "handlers", "item_handler.go"))
	assertFileExists(t, filepath.Join(target, "internal", "repository", "item_repository.go"))
	assertFileExists(t, filepath.Join(target, ".uvmrc"))

	data, _ := os.ReadFile(filepath.Join(target, ".uvmrc"))
	if !strings.Contains(string(data), "go 1.22.6") {
		t.Errorf("expected .uvmrc to contain 'go 1.22.6', got: %s", string(data))
	}
}

func TestScaffoldGoChi(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "my-go-chi")

	mgr := NewManager()
	out := new(bytes.Buffer)

	err := mgr.Scaffold(ProjectOptions{
		ProjectName: "my-go-chi",
		TargetDir:   target,
		Language:    "go",
		Framework:   "chi",
	}, out)

	if err != nil {
		t.Fatalf("Scaffold failed: %v", err)
	}

	assertFileExists(t, filepath.Join(target, "go.mod"))
	assertFileExists(t, filepath.Join(target, "cmd", "server", "main.go"))
}

func TestScaffoldGoFiber(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "my-go-fiber")

	mgr := NewManager()
	out := new(bytes.Buffer)

	err := mgr.Scaffold(ProjectOptions{
		ProjectName: "my-go-fiber",
		TargetDir:   target,
		Language:    "go",
		Framework:   "fiber",
	}, out)

	if err != nil {
		t.Fatalf("Scaffold failed: %v", err)
	}

	assertFileExists(t, filepath.Join(target, "go.mod"))
	assertFileExists(t, filepath.Join(target, "cmd", "server", "main.go"))
}

func TestScaffoldValidationErrors(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager()
	out := new(bytes.Buffer)

	// 1. Empty name
	err := mgr.Scaffold(ProjectOptions{ProjectName: ""}, out)
	if err == nil {
		t.Errorf("expected error for empty project name")
	}

	// 2. Unsupported language
	err = mgr.Scaffold(ProjectOptions{ProjectName: "demo", Language: "rust"}, out)
	if err == nil {
		t.Errorf("expected error for unsupported language")
	}

	// 3. Target dir exists and is not empty
	nonEmpty := filepath.Join(tmpDir, "non_empty")
	_ = os.MkdirAll(nonEmpty, 0755)
	_ = os.WriteFile(filepath.Join(nonEmpty, "test.txt"), []byte("data"), 0644)

	err = mgr.Scaffold(ProjectOptions{
		ProjectName: "demo",
		TargetDir:   nonEmpty,
		Language:    "node",
	}, out)
	if err == nil {
		t.Errorf("expected error for non-empty target directory")
	}
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist, but was not found", path)
	}
}
