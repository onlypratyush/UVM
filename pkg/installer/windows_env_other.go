//go:build !windows

package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// POSIXPathManager implements PathManager on macOS and Linux by managing shell configuration files.
type POSIXPathManager struct {
	HomeDir   string
	UserShell string
}

// NewPlatformPathManager creates a PathManager for the current platform.
func NewPlatformPathManager(homeDir string, userShell string) PathManager {
	return &POSIXPathManager{
		HomeDir:   homeDir,
		UserShell: userShell,
	}
}

// GetShellConfigFiles returns candidate shell configuration files for the user.
func (m *POSIXPathManager) GetShellConfigFiles() []string {
	var files []string
	switch m.UserShell {
	case "zsh":
		files = append(files, filepath.Join(m.HomeDir, ".zshrc"))
	case "fish":
		files = append(files, filepath.Join(m.HomeDir, ".config", "fish", "config.fish"))
	case "bash":
		bashProfile := filepath.Join(m.HomeDir, ".bash_profile")
		if _, err := os.Stat(bashProfile); err == nil {
			files = append(files, bashProfile)
		}
		files = append(files, filepath.Join(m.HomeDir, ".bashrc"))
	default:
		files = append(files, filepath.Join(m.HomeDir, ".profile"), filepath.Join(m.HomeDir, ".bashrc"))
	}
	return files
}

// GetPath returns the current PATH from the process environment.
func (m *POSIXPathManager) GetPath() (string, error) {
	return os.Getenv("PATH"), nil
}

// SetPath sets the process environment PATH (for the current process).
func (m *POSIXPathManager) SetPath(newPath string) error {
	return os.Setenv("PATH", newPath)
}

// AddEntry ensures the entry is added to the shell profile configuration files.
func (m *POSIXPathManager) AddEntry(entry string) error {
	configs := m.GetShellConfigFiles()
	for _, conf := range configs {
		if err := os.MkdirAll(filepath.Dir(conf), 0755); err != nil {
			return err
		}

		content := ""
		if data, err := os.ReadFile(conf); err == nil {
			content = string(data)
		}

		if strings.Contains(content, entry) {
			continue
		}

		var block string
		if strings.Contains(conf, "fish") {
			block = fmt.Sprintf("\n# uvm (Universal Version Manager)\nfish_add_path %s\n", entry)
		} else {
			parentDir := filepath.Dir(entry)
			block = fmt.Sprintf("\n# uvm (Universal Version Manager)\nexport UVM_INSTALL=\"%s\"\nexport PATH=\"%s:$PATH\"\n", parentDir, entry)
		}

		f, err := os.OpenFile(conf, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		_, writeErr := f.WriteString(block)
		f.Close()
		if writeErr != nil {
			return writeErr
		}
	}

	// Also update current process PATH
	curPath, _ := m.GetPath()
	entries := strings.Split(curPath, ":")
	updated := AddPathEntry(entries, entry, "posix")
	_ = m.SetPath(strings.Join(updated, ":"))

	return nil
}

// RemoveEntries removes specific entries from shell config files.
func (m *POSIXPathManager) RemoveEntries(toRemove []string) error {
	configs := m.GetShellConfigFiles()
	for _, conf := range configs {
		data, err := os.ReadFile(conf)
		if err != nil {
			continue
		}

		lines := strings.Split(string(data), "\n")
		var newLines []string
		for _, line := range lines {
			shouldRemove := false
			for _, rem := range toRemove {
				if strings.Contains(line, rem) {
					shouldRemove = true
					break
				}
			}
			if !shouldRemove {
				newLines = append(newLines, line)
			}
		}

		_ = os.WriteFile(conf, []byte(strings.Join(newLines, "\n")), 0644)
	}

	// Update current process PATH
	curPath, _ := m.GetPath()
	entries := strings.Split(curPath, ":")
	filtered := FilterPathList(entries, toRemove, "posix")
	_ = m.SetPath(strings.Join(filtered, ":"))

	return nil
}

// BroadcastChange is a no-op on POSIX systems.
func (m *POSIXPathManager) BroadcastChange() error {
	return nil
}
