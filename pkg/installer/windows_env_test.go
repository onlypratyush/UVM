package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCleanPathEntry(t *testing.T) {
	tests := map[string]string{
		`"C:\Program Files\nodejs\"`: `C:\Program Files\nodejs`,
		`  /usr/local/bin/  `:        `/usr/local/bin`,
		`""`:                         ``,
		`   `:                        ``,
	}

	for in, expected := range tests {
		got := CleanPathEntry(in)
		if filepath.Clean(got) != filepath.Clean(expected) && (got != "" || expected != "") {
			t.Errorf("CleanPathEntry(%q) = %q, expected %q", in, got, expected)
		}
	}
}

func TestPathMatches(t *testing.T) {
	// Windows case-insensitivity
	if !PathMatches(`C:\Program Files\Nodejs`, `c:\program files\nodejs\`, "windows") {
		t.Errorf("expected match on windows with case differences and trailing slash")
	}

	// POSIX case-sensitivity
	if PathMatches(`/usr/local/bin`, `/usr/Local/bin`, "linux") {
		t.Errorf("expected no match on linux for case differences")
	}
	if !PathMatches(`/usr/local/bin/`, `/usr/local/bin`, "linux") {
		t.Errorf("expected match on linux for trailing slash difference")
	}

	// Empty checks
	if PathMatches("", "/usr/local/bin", "linux") || PathMatches("/usr/local/bin", "", "linux") {
		t.Errorf("expected no match for empty paths")
	}
}

func TestFilterPathList(t *testing.T) {
	entries := []string{
		`C:\Windows\System32`,
		`C:\Program Files\nodejs`,
		`C:\Users\user\AppData\Roaming\npm`,
		`C:\Program Files\Git\bin`,
		`C:\Program Files\nodejs\`, // duplicate with trailing slash
	}

	toRemove := []string{
		`c:\program files\nodejs`,
		`C:\Users\user\AppData\Roaming\npm\`,
	}

	filtered := FilterPathList(entries, toRemove, "windows")

	if len(filtered) != 2 {
		t.Fatalf("expected 2 remaining entries, got %d: %+v", len(filtered), filtered)
	}

	if filtered[0] != `C:\Windows\System32` || filtered[1] != `C:\Program Files\Git\bin` {
		t.Errorf("unexpected filtered entries: %+v", filtered)
	}
}

func TestAddPathEntry(t *testing.T) {
	entries := []string{
		`C:\Windows\System32`,
		`C:\Users\user\.uvm\bin`,
	}

	// Add existing entry (case-insensitive on windows)
	updated := AddPathEntry(entries, `c:\users\user\.uvm\bin\`, "windows")
	if len(updated) != 2 {
		t.Errorf("expected no duplicate to be added, got %d entries", len(updated))
	}

	// Add new entry
	updated2 := AddPathEntry(entries, `C:\Go\bin`, "windows")
	if len(updated2) != 3 || updated2[2] != `C:\Go\bin` {
		t.Errorf("expected new entry to be added: %+v", updated2)
	}

	// Empty entry
	updated3 := AddPathEntry(entries, "", "windows")
	if len(updated3) != 2 {
		t.Errorf("expected empty entry to be ignored")
	}
}

func TestPOSIXPathManager(t *testing.T) {
	tmpHome := t.TempDir()

	mgrZsh := NewPlatformPathManager(tmpHome, "zsh")
	if err := mgrZsh.AddEntry("/custom/uvm/bin"); err != nil {
		t.Fatalf("AddEntry failed: %v", err)
	}

	zshContent, _ := os.ReadFile(filepath.Join(tmpHome, ".zshrc"))
	if !strings.Contains(string(zshContent), "/custom/uvm/bin") {
		t.Errorf("expected entry in .zshrc: %s", string(zshContent))
	}

	// Test idempotency
	_ = mgrZsh.AddEntry("/custom/uvm/bin")

	// Test RemoveEntries
	if err := mgrZsh.RemoveEntries([]string{"/custom/uvm/bin"}); err != nil {
		t.Fatalf("RemoveEntries failed: %v", err)
	}
	zshContentAfter, _ := os.ReadFile(filepath.Join(tmpHome, ".zshrc"))
	if strings.Contains(string(zshContentAfter), "/custom/uvm/bin") {
		t.Errorf("expected entry to be removed from .zshrc: %s", string(zshContentAfter))
	}

	// Test fish shell
	mgrFish := NewPlatformPathManager(tmpHome, "fish")
	_ = mgrFish.AddEntry("/fish/uvm/bin")
	fishContent, _ := os.ReadFile(filepath.Join(tmpHome, ".config", "fish", "config.fish"))
	if !strings.Contains(string(fishContent), "fish_add_path /fish/uvm/bin") {
		t.Errorf("expected fish_add_path in config.fish: %s", string(fishContent))
	}

	// Test GetPath and SetPath
	_ = mgrZsh.SetPath("/new/path")
	p, _ := mgrZsh.GetPath()
	if !strings.Contains(p, "/new/path") {
		t.Errorf("unexpected path: %s", p)
	}

	_ = mgrZsh.BroadcastChange()
}
