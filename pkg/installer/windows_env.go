package installer

import (
	"path/filepath"
	"strings"
)

// PathManager defines an interface for managing environment PATH across OSes.
type PathManager interface {
	GetPath() (string, error)
	SetPath(newPath string) error
	AddEntry(entry string) error
	RemoveEntries(entries []string) error
	BroadcastChange() error
}

// CleanPathEntry normalizes a path entry for comparison by stripping quotes,
// trailing slashes, and converting to clean filepath.
func CleanPathEntry(p string) string {
	p = strings.TrimSpace(p)
	p = strings.Trim(p, "\"")
	p = strings.TrimRight(p, `/\`)
	if p == "" {
		return ""
	}
	return filepath.Clean(p)
}

// PathMatches compares two path entries. On Windows, it performs case-insensitive comparison.
func PathMatches(a, b string, goos string) bool {
	ca := CleanPathEntry(a)
	cb := CleanPathEntry(b)
	if ca == "" || cb == "" {
		return false
	}
	if goos == "windows" {
		return strings.EqualFold(ca, cb)
	}
	return ca == cb
}

// FilterPathList removes specified entries and duplicates from a list of path entries,
// preserving the order of unrelated entries.
func FilterPathList(currentEntries []string, toRemove []string, goos string) []string {
	var result []string
	seen := make(map[string]bool)

	for _, entry := range currentEntries {
		cleaned := CleanPathEntry(entry)
		if cleaned == "" {
			continue
		}

		key := cleaned
		if goos == "windows" {
			key = strings.ToLower(cleaned)
		}

		// Check if it matches any entry in toRemove
		shouldRemove := false
		for _, rem := range toRemove {
			if PathMatches(cleaned, rem, goos) {
				shouldRemove = true
				break
			}
		}

		if !shouldRemove && !seen[key] {
			seen[key] = true
			result = append(result, entry)
		}
	}

	return result
}

// AddPathEntry adds an entry to the path list if not already present.
func AddPathEntry(currentEntries []string, newEntry string, goos string) []string {
	cleanedNew := CleanPathEntry(newEntry)
	if cleanedNew == "" {
		return currentEntries
	}

	for _, entry := range currentEntries {
		if PathMatches(entry, cleanedNew, goos) {
			return currentEntries // Already present
		}
	}

	return append(currentEntries, newEntry)
}
