package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// UvmrcFileName is the standard configuration file for project runtime versions
const UvmrcFileName = ".uvmrc"

// FindUvmrc searches for a .uvmrc file starting from startDir and walking up parents
func FindUvmrc(startDir string) (string, error) {
	if startDir == "" {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}

	currentDir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for {
		candidate := filepath.Join(currentDir, UvmrcFileName)
		if stat, err := os.Stat(candidate); err == nil && !stat.IsDir() {
			return candidate, nil
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// Reached filesystem root
			break
		}
		currentDir = parentDir
	}

	return "", fmt.Errorf("no %s file found in %s or parent directories", UvmrcFileName, startDir)
}

// ParseUvmrc reads and parses a .uvmrc file into a map of runtime -> version
func ParseUvmrc(filePath string) (map[string]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", filePath, err)
	}
	defer f.Close()

	runtimes := make(map[string]string)
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		// Support formats:
		// 1. "node 20.11.0" or "node 20"
		// 2. "node=20.11.0" or "node: 20"
		// 3. Just a version "20.11.0" (defaults to node for legacy compatibility)
		var runtime, version string

		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			runtime = strings.ToLower(strings.TrimSpace(parts[0]))
			version = strings.TrimSpace(parts[1])
		} else if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			runtime = strings.ToLower(strings.TrimSpace(parts[0]))
			version = strings.TrimSpace(parts[1])
		} else {
			fields := strings.Fields(line)
			if len(fields) == 1 {
				// Single version without runtime name -> assume node
				runtime = "node"
				version = fields[0]
			} else if len(fields) >= 2 {
				runtime = strings.ToLower(fields[0])
				version = fields[1]
			}
		}

		// Normalize runtime aliases
		switch runtime {
		case "nodejs":
			runtime = "node"
		case "golang":
			runtime = "go"
		case "py", "python3":
			runtime = "python"
		case "jdk", "openjdk":
			runtime = "java"
		}

		if runtime != "" && version != "" {
			runtimes[runtime] = version
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", filePath, err)
	}

	return runtimes, nil
}

// WriteUvmrc writes or updates a .uvmrc file in the target directory
func WriteUvmrc(dir string, runtimes map[string]string) error {
	filePath := filepath.Join(dir, UvmrcFileName)

	var sb strings.Builder
	sb.WriteString("# UVM Project Runtime Configuration\n")
	sb.WriteString("# Automatically detected by 'uvm use'\n\n")

	// Order keys consistently: node, go, python, php, java, bun, then others
	order := []string{"node", "go", "python", "php", "java", "bun"}
	written := make(map[string]bool)

	for _, rt := range order {
		if ver, ok := runtimes[rt]; ok && ver != "" {
			sb.WriteString(fmt.Sprintf("%s %s\n", rt, ver))
			written[rt] = true
		}
	}

	for rt, ver := range runtimes {
		if !written[rt] && ver != "" {
			sb.WriteString(fmt.Sprintf("%s %s\n", rt, ver))
		}
	}

	return os.WriteFile(filePath, []byte(sb.String()), 0644)
}

// DetectProjectRuntimes checks current directory for .uvmrc, or fallback project files (.nvmrc, go.mod, etc.)
func DetectProjectRuntimes(dir string) (map[string]string, string, error) {
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return nil, "", err
		}
	}

	// 1. Try finding .uvmrc
	uvmrcPath, err := FindUvmrc(dir)
	if err == nil {
		runtimes, err := ParseUvmrc(uvmrcPath)
		if err == nil && len(runtimes) > 0 {
			return runtimes, uvmrcPath, nil
		}
	}

	// 2. Fallbacks in directory
	results := make(map[string]string)
	var matchedFile string

	// Node fallbacks: .nvmrc or .node-version
	nvmrcPath := filepath.Join(dir, ".nvmrc")
	nodeVerPath := filepath.Join(dir, ".node-version")
	if data, err := os.ReadFile(nvmrcPath); err == nil {
		ver := strings.TrimSpace(string(data))
		if ver != "" {
			results["node"] = ver
			matchedFile = nvmrcPath
		}
	} else if data, err := os.ReadFile(nodeVerPath); err == nil {
		ver := strings.TrimSpace(string(data))
		if ver != "" {
			results["node"] = ver
			matchedFile = nodeVerPath
		}
	}

	// Go fallback: go.mod
	goModPath := filepath.Join(dir, "go.mod")
	if data, err := os.ReadFile(goModPath); err == nil {
		re := regexp.MustCompile(`(?m)^go\s+([0-9]+\.[0-9]+(\.[0-9]+)?)`)
		if matches := re.FindStringSubmatch(string(data)); len(matches) > 1 {
			results["go"] = matches[1]
			if matchedFile == "" {
				matchedFile = goModPath
			}
		}
	}

	// Python fallback: .python-version
	pyVerPath := filepath.Join(dir, ".python-version")
	if data, err := os.ReadFile(pyVerPath); err == nil {
		ver := strings.TrimSpace(string(data))
		if ver != "" {
			results["python"] = ver
			if matchedFile == "" {
				matchedFile = pyVerPath
			}
		}
	}

	// PHP fallbacks: .php-version, composer.json
	phpVerPath := filepath.Join(dir, ".php-version")
	composerPath := filepath.Join(dir, "composer.json")
	if data, err := os.ReadFile(phpVerPath); err == nil {
		ver := strings.TrimSpace(string(data))
		if ver != "" {
			results["php"] = ver
			if matchedFile == "" {
				matchedFile = phpVerPath
			}
		}
	} else if data, err := os.ReadFile(composerPath); err == nil {
		re := regexp.MustCompile(`"php"\s*:\s*"[^0-9]*([0-9]+\.[0-9]+(\.[0-9]+)?)"`)
		if matches := re.FindStringSubmatch(string(data)); len(matches) > 1 {
			results["php"] = matches[1]
			if matchedFile == "" {
				matchedFile = composerPath
			}
		}
	}

	// Java fallbacks: .java-version, pom.xml, build.gradle, build.gradle.kts
	javaVerPath := filepath.Join(dir, ".java-version")
	pomPath := filepath.Join(dir, "pom.xml")
	gradlePath := filepath.Join(dir, "build.gradle")
	gradleKtsPath := filepath.Join(dir, "build.gradle.kts")

	if data, err := os.ReadFile(javaVerPath); err == nil {
		ver := strings.TrimSpace(string(data))
		if ver != "" {
			results["java"] = ver
			if matchedFile == "" {
				matchedFile = javaVerPath
			}
		}
	} else if data, err := os.ReadFile(pomPath); err == nil {
		re := regexp.MustCompile(`<(?:java\.version|maven\.compiler\.source|maven\.compiler\.target|release)>([0-9]+(\.[0-9]+)*)</`)
		if matches := re.FindStringSubmatch(string(data)); len(matches) > 1 {
			results["java"] = matches[1]
			if matchedFile == "" {
				matchedFile = pomPath
			}
		}
	} else if data, err := os.ReadFile(gradlePath); err == nil {
		re := regexp.MustCompile(`(?:sourceCompatibility|targetCompatibility|jvmToolchain\(?)\s*=?\s*['"]?([0-9]+(\.[0-9]+)*)`)
		if matches := re.FindStringSubmatch(string(data)); len(matches) > 1 {
			results["java"] = matches[1]
			if matchedFile == "" {
				matchedFile = gradlePath
			}
		}
	} else if data, err := os.ReadFile(gradleKtsPath); err == nil {
		re := regexp.MustCompile(`(?:sourceCompatibility|targetCompatibility|jvmToolchain\(?)\s*=?\s*['"]?([0-9]+(\.[0-9]+)*)`)
		if matches := re.FindStringSubmatch(string(data)); len(matches) > 1 {
			results["java"] = matches[1]
			if matchedFile == "" {
				matchedFile = gradleKtsPath
			}
		}
	}

	// Bun fallbacks: .bun-version, package.json (packageManager / engines.bun), bun.lock, bun.lockb
	bunVerPath := filepath.Join(dir, ".bun-version")
	pkgJsonPath := filepath.Join(dir, "package.json")
	bunLockPath := filepath.Join(dir, "bun.lock")
	bunLockbPath := filepath.Join(dir, "bun.lockb")

	if data, err := os.ReadFile(bunVerPath); err == nil {
		ver := strings.TrimSpace(string(data))
		if ver != "" {
			results["bun"] = ver
			if matchedFile == "" {
				matchedFile = bunVerPath
			}
		}
	} else if data, err := os.ReadFile(pkgJsonPath); err == nil {
		re := regexp.MustCompile(`"packageManager"\s*:\s*"bun@([0-9]+\.[0-9]+(\.[0-9]+)?)"`)
		if matches := re.FindStringSubmatch(string(data)); len(matches) > 1 {
			results["bun"] = matches[1]
			if matchedFile == "" {
				matchedFile = pkgJsonPath
			}
		} else {
			reEng := regexp.MustCompile(`"bun"\s*:\s*"[^0-9]*([0-9]+\.[0-9]+(\.[0-9]+)?)"`)
			if matchesEng := reEng.FindStringSubmatch(string(data)); len(matchesEng) > 1 {
				results["bun"] = matchesEng[1]
				if matchedFile == "" {
					matchedFile = pkgJsonPath
				}
			}
		}
	} else if _, err := os.Stat(bunLockPath); err == nil {
		results["bun"] = "latest"
		if matchedFile == "" {
			matchedFile = bunLockPath
		}
	} else if _, err := os.Stat(bunLockbPath); err == nil {
		results["bun"] = "latest"
		if matchedFile == "" {
			matchedFile = bunLockbPath
		}
	}

	if len(results) == 0 {
		return nil, "", fmt.Errorf("no runtime configuration found in %s", dir)
	}

	return results, matchedFile, nil
}
