package scaffold

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"uvm/pkg/config"
)

// ProjectOptions defines parameters for scaffolding a new project
type ProjectOptions struct {
	ProjectName string
	TargetDir   string
	Language    string // "node" | "go"
	Dialect     string // "ts" | "js" (for Node)
	Framework   string // "express" | "fastify" | "gin" | "chi" | "fiber"
	IncludeCRUD bool
	Version     string // runtime version to pin in .uvmrc
}

// Manager handles project scaffolding and template creation
type Manager struct{}

// NewManager creates a new Scaffold Manager
func NewManager() *Manager {
	return &Manager{}
}

// Scaffold creates a new project based on the provided options
func (m *Manager) Scaffold(opts ProjectOptions, out io.Writer) error {
	if opts.ProjectName == "" {
		return fmt.Errorf("project name cannot be empty")
	}

	opts.Language = strings.ToLower(strings.TrimSpace(opts.Language))
	opts.Framework = strings.ToLower(strings.TrimSpace(opts.Framework))
	opts.Dialect = strings.ToLower(strings.TrimSpace(opts.Dialect))

	// Normalize defaults
	if opts.Language == "" {
		opts.Language = "node"
	}

	if opts.Language == "node" || opts.Language == "nodejs" {
		opts.Language = "node"
		if opts.Framework == "" {
			opts.Framework = "express"
		}
		if opts.Dialect == "" {
			opts.Dialect = "ts"
		}
		if opts.Version == "" {
			opts.Version = "20"
		}
	} else if opts.Language == "go" || opts.Language == "golang" {
		opts.Language = "go"
		if opts.Framework == "" {
			opts.Framework = "gin"
		}
		if opts.Version == "" {
			opts.Version = "1.22"
		}
	} else {
		return fmt.Errorf("unsupported language %q (supported: node, go)", opts.Language)
	}

	if opts.TargetDir == "" {
		opts.TargetDir = opts.ProjectName
	}

	targetAbs, err := filepath.Abs(opts.TargetDir)
	if err != nil {
		return err
	}

	// Check if directory exists and is not empty
	if entries, err := os.ReadDir(targetAbs); err == nil && len(entries) > 0 {
		return fmt.Errorf("target directory %s already exists and is not empty", targetAbs)
	}

	if err := os.MkdirAll(targetAbs, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", targetAbs, err)
	}

	var files map[string]string
	if opts.Language == "node" {
		files = GetNodeTemplates(opts.ProjectName, opts.Framework, opts.Dialect, opts.IncludeCRUD)
	} else {
		files = GetGoTemplates(opts.ProjectName, opts.Framework, opts.Version, opts.IncludeCRUD)
	}

	fmt.Fprintf(out, "🚀 Scaffolding new %s project (%s)...\n", strings.ToUpper(opts.Language), opts.Framework)

	for relPath, content := range files {
		fullPath := filepath.Join(targetAbs, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return err
		}
		fmt.Fprintf(out, "  ✓ Created %s\n", relPath)
	}

	// Write .uvmrc to project
	uvmrcRuntimes := map[string]string{
		opts.Language: opts.Version,
	}
	if err := config.WriteUvmrc(targetAbs, uvmrcRuntimes); err == nil {
		fmt.Fprintf(out, "  ✓ Created .uvmrc (%s %s)\n", opts.Language, opts.Version)
	}

	fmt.Fprintf(out, "\n🎉 Project %s created successfully!\n\n", opts.ProjectName)
	fmt.Fprintf(out, "Next steps:\n")
	fmt.Fprintf(out, "  1. cd %s\n", opts.ProjectName)
	fmt.Fprintf(out, "  2. uvm use          # (auto-activates %s %s from .uvmrc)\n", opts.Language, opts.Version)

	if opts.Language == "node" {
		fmt.Fprintf(out, "  3. npm install\n")
		fmt.Fprintf(out, "  4. npm run dev\n")
	} else {
		fmt.Fprintf(out, "  3. go mod tidy\n")
		fmt.Fprintf(out, "  4. go run ./cmd/server\n")
	}

	return nil
}
