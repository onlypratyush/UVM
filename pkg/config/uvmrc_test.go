package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindUvmrc(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Not found in empty directory
	_, err := FindUvmrc(tmpDir)
	if err == nil {
		t.Errorf("expected error when .uvmrc does not exist")
	}

	// 2. Found in current directory
	uvmrcPath := filepath.Join(tmpDir, ".uvmrc")
	_ = os.WriteFile(uvmrcPath, []byte("node 20.11.0\n"), 0644)

	found, err := FindUvmrc(tmpDir)
	if err != nil || found != uvmrcPath {
		t.Fatalf("expected to find %s, got %s (err: %v)", uvmrcPath, found, err)
	}

	// 3. Found in parent directory
	subDir := filepath.Join(tmpDir, "sub", "deep")
	_ = os.MkdirAll(subDir, 0755)

	foundParent, err := FindUvmrc(subDir)
	if err != nil || foundParent != uvmrcPath {
		t.Fatalf("expected to find parent %s from subdir, got %s (err: %v)", uvmrcPath, foundParent, err)
	}

	// 4. Default to current working directory if empty string
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	_ = os.Chdir(subDir)

	foundWd, err := FindUvmrc("")
	if err != nil {
		t.Fatalf("expected to find uvmrc from empty string cwd (err: %v)", err)
	}
	realWd, _ := filepath.EvalSymlinks(foundWd)
	realParent, _ := filepath.EvalSymlinks(uvmrcPath)
	if realWd != realParent {
		t.Fatalf("expected real path %s, got %s", realParent, realWd)
	}
}

func TestParseUvmrc(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, ".uvmrc")

	content := `# Project config
// comment
node 20.11.0
go=1.22.6
python: 3.12.9
nodejs 18.0.0
golang 1.21.0
py 3.11.0
22.0.0
`
	_ = os.WriteFile(filePath, []byte(content), 0644)

	res, err := ParseUvmrc(filePath)
	if err != nil {
		t.Fatalf("ParseUvmrc failed: %v", err)
	}

	if res["node"] != "22.0.0" { // last node entry wins
		t.Errorf("expected node 22.0.0, got %s", res["node"])
	}
	if res["go"] != "1.21.0" {
		t.Errorf("expected go 1.21.0, got %s", res["go"])
	}
	if res["python"] != "3.11.0" {
		t.Errorf("expected python 3.11.0, got %s", res["python"])
	}

	// Non-existent file
	_, err = ParseUvmrc(filepath.Join(tmpDir, "does_not_exist"))
	if err == nil {
		t.Errorf("expected error for non-existent file")
	}
}

func TestWriteUvmrc(t *testing.T) {
	tmpDir := t.TempDir()

	runtimes := map[string]string{
		"node":   "20.11.0",
		"go":     "1.22.6",
		"python": "3.12.9",
		"ruby":   "3.2.0",
	}

	err := WriteUvmrc(tmpDir, runtimes)
	if err != nil {
		t.Fatalf("WriteUvmrc failed: %v", err)
	}

	parsed, err := ParseUvmrc(filepath.Join(tmpDir, ".uvmrc"))
	if err != nil {
		t.Fatalf("failed to read back written .uvmrc: %v", err)
	}

	if parsed["node"] != "20.11.0" || parsed["go"] != "1.22.6" || parsed["python"] != "3.12.9" {
		t.Errorf("mismatched parsed values: %+v", parsed)
	}
}

func TestDetectProjectRuntimes(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. None found
	_, _, err := DetectProjectRuntimes(tmpDir)
	if err == nil {
		t.Errorf("expected error when no runtime configs exist")
	}

	// 2. .nvmrc fallback
	_ = os.WriteFile(filepath.Join(tmpDir, ".nvmrc"), []byte("v20.11.0\n"), 0644)
	res, matched, err := DetectProjectRuntimes(tmpDir)
	if err != nil || res["node"] != "v20.11.0" || !filepath.IsAbs(matched) {
		t.Errorf("expected .nvmrc detection, got: %+v, matched: %s, err: %v", res, matched, err)
	}

	// 3. .node-version fallback (remove .nvmrc)
	_ = os.Remove(filepath.Join(tmpDir, ".nvmrc"))
	_ = os.WriteFile(filepath.Join(tmpDir, ".node-version"), []byte("18.20.0\n"), 0644)
	res, _, err = DetectProjectRuntimes(tmpDir)
	if err != nil || res["node"] != "18.20.0" {
		t.Errorf("expected .node-version detection, got: %+v, err: %v", res, err)
	}

	// 4. go.mod fallback
	goModContent := `module myapp

go 1.22.6

require github.com/gin-gonic/gin v1.9.1
`
	_ = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644)
	res, _, err = DetectProjectRuntimes(tmpDir)
	if err != nil || res["go"] != "1.22.6" {
		t.Errorf("expected go.mod detection, got: %+v, err: %v", res, err)
	}

	// 5. .python-version fallback
	_ = os.WriteFile(filepath.Join(tmpDir, ".python-version"), []byte("3.12.2\n"), 0644)
	res, _, err = DetectProjectRuntimes(tmpDir)
	if err != nil || res["python"] != "3.12.2" {
		t.Errorf("expected .python-version detection, got: %+v, err: %v", res, err)
	}

	// 6. .php-version fallback
	_ = os.WriteFile(filepath.Join(tmpDir, ".php-version"), []byte("8.3.17\n"), 0644)
	res, _, err = DetectProjectRuntimes(tmpDir)
	if err != nil || res["php"] != "8.3.17" {
		t.Errorf("expected .php-version detection, got: %+v, err: %v", res, err)
	}

	// 7. composer.json fallback (remove .php-version)
	_ = os.Remove(filepath.Join(tmpDir, ".php-version"))
	composerContent := `{
    "name": "vendor/project",
    "require": {
        "php": ">=8.2.0"
    }
}`
	_ = os.WriteFile(filepath.Join(tmpDir, "composer.json"), []byte(composerContent), 0644)
	res, _, err = DetectProjectRuntimes(tmpDir)
	if err != nil || res["php"] != "8.2.0" {
		t.Errorf("expected composer.json detection, got: %+v, err: %v", res, err)
	}

	// 8. .java-version fallback
	_ = os.WriteFile(filepath.Join(tmpDir, ".java-version"), []byte("21.0.6\n"), 0644)
	res, _, err = DetectProjectRuntimes(tmpDir)
	if err != nil || res["java"] != "21.0.6" {
		t.Errorf("expected .java-version detection, got: %+v, err: %v", res, err)
	}

	// 9. pom.xml fallback (remove .java-version)
	_ = os.Remove(filepath.Join(tmpDir, ".java-version"))
	pomContent := `<project>
  <properties>
    <java.version>17</java.version>
  </properties>
</project>`
	_ = os.WriteFile(filepath.Join(tmpDir, "pom.xml"), []byte(pomContent), 0644)
	res, _, err = DetectProjectRuntimes(tmpDir)
	if err != nil || res["java"] != "17" {
		t.Errorf("expected pom.xml detection, got: %+v, err: %v", res, err)
	}

	// 10. build.gradle fallback (remove pom.xml)
	_ = os.Remove(filepath.Join(tmpDir, "pom.xml"))
	gradleContent := `plugins {
    id 'java'
}
sourceCompatibility = '21'
`
	_ = os.WriteFile(filepath.Join(tmpDir, "build.gradle"), []byte(gradleContent), 0644)
	res, _, err = DetectProjectRuntimes(tmpDir)
	if err != nil || res["java"] != "21" {
		t.Errorf("expected build.gradle detection, got: %+v, err: %v", res, err)
	}

	// 11. Bun fallbacks: .bun-version
	_ = os.WriteFile(filepath.Join(tmpDir, ".bun-version"), []byte("1.2.4\n"), 0644)
	res, _, err = DetectProjectRuntimes(tmpDir)
	if err != nil || res["bun"] != "1.2.4" {
		t.Errorf("expected .bun-version detection, got: %+v, err: %v", res, err)
	}

	// 12. Bun fallback: package.json (remove .bun-version)
	_ = os.Remove(filepath.Join(tmpDir, ".bun-version"))
	pkgJsonContent := `{"name": "bun-app", "packageManager": "bun@1.2.0"}`
	_ = os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(pkgJsonContent), 0644)
	res, _, err = DetectProjectRuntimes(tmpDir)
	if err != nil || res["bun"] != "1.2.0" {
		t.Errorf("expected package.json packageManager bun detection, got: %+v, err: %v", res, err)
	}

	// 13. Bun fallback: bun.lock (remove package.json)
	_ = os.Remove(filepath.Join(tmpDir, "package.json"))
	_ = os.WriteFile(filepath.Join(tmpDir, "bun.lock"), []byte("lockfile"), 0644)
	res, _, err = DetectProjectRuntimes(tmpDir)
	if err != nil || res["bun"] != "latest" {
		t.Errorf("expected bun.lock detection, got: %+v, err: %v", res, err)
	}

	// 14. .uvmrc takes priority over all fallbacks
	_ = os.WriteFile(filepath.Join(tmpDir, ".uvmrc"), []byte("node 24\ngo 1.23\nphp 8.4\njava 21\nbun 1.2\n"), 0644)
	res, _, err = DetectProjectRuntimes(tmpDir)
	if err != nil || res["node"] != "24" || res["go"] != "1.23" || res["php"] != "8.4" || res["java"] != "21" || res["bun"] != "1.2" {
		t.Errorf("expected .uvmrc priority, got: %+v, err: %v", res, err)
	}
}
