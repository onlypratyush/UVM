# 🚀 uvm (Universal Version Manager)

[![Release](https://img.shields.io/badge/release-v0.0.3-blue.svg)](https://github.com/onlypratyush/UVM-)
[![Go Version](https://img.shields.io/badge/go-1.22+-00ADD8.svg?style=flat&logo=go)](https://golang.org)
[![Coverage](https://img.shields.io/badge/coverage-94.6%25-brightgreen.svg)]()
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-brightgreen.svg)]()

**uvm** (Universal Version Manager) is a fast, cross-platform, and lightweight CLI tool designed to install, manage, and switch between multiple programming language runtimes (Node.js, Go, Python, Rust, etc.) seamlessly across **macOS**, **Linux**, and **Windows**.

---

## ⚡ Quick Installation

### 🍎 macOS & 🐧 Linux
```bash
curl -fsSL https://raw.githubusercontent.com/onlypratyush/UVM-/main/install.sh | bash
```

### 🪟 Windows (PowerShell)
```powershell
irm https://raw.githubusercontent.com/onlypratyush/UVM-/main/install.ps1 | iex
```

### 🍺 Homebrew (macOS / Linux)
```bash
brew tap onlypratyush/uvm https://github.com/onlypratyush/UVM-
brew install uvm
```

### 🍨 Scoop (Windows)
```powershell
scoop bucket add uvm https://github.com/onlypratyush/UVM-
scoop install uvm
```

---

## 📦 Full Node.js Runtime Support

`uvm` natively downloads, unpacks, and manages official Node.js releases directly from `nodejs.org/dist`:

### 1. Install Node.js
```bash
# Specific version
uvm install node 20.11.0

# Latest LTS version (e.g. 20.x Iron, 22.x Jod)
uvm install node lts

# Latest current release
uvm install node latest
```

### 2. Switch Active Version
```bash
uvm use node 20.11.0
uvm use node lts
```
*Instantly configures `node`, `npm`, `npx`, and `corepack` in your environment.*

### 3. List Installed Versions
```bash
uvm list node
# or
uvm ls
```
Output:
```text
Installed Node.js versions:
  * v20.11.0 (active)
    v18.20.0
    v22.2.0
```

### 4. Check Current Version
```bash
uvm current node
# Output: Current Node.js version: v20.11.0
```

### 5. Remove / Uninstall Version
```bash
uvm remove node 18.20.0
# or alias
uvm rm node 18.20.0
```

---

## 🧪 Comprehensive Testing Guide (macOS & Windows)

### 🍎 Testing on macOS (Terminal / Zsh / Bash)

1. **Build `uvm` locally**:
   ```bash
   make build
   export PATH="$PWD/bin:$PATH"
   ```
2. **Verify installation & help**:
   ```bash
   uvm --help
   uvm --version
   ```
3. **Install Node.js versions**:
   ```bash
   uvm install node 20.11.0
   uvm install node 18.20.0
   ```
4. **List versions**:
   ```bash
   uvm list node
   ```
5. **Switch between versions & test Node / NPM**:
   ```bash
   uvm use node 20.11.0
   node -v    # Output: v20.11.0
   npm -v

   uvm use node 18.20.0
   node -v    # Output: v18.20.0
   ```
6. **Remove version**:
   ```bash
   uvm remove node 18.20.0
   ```
7. **Run automated unit tests with coverage**:
   ```bash
   make test-coverage
   ./tests/test_install.sh
   ```

---

### 🪟 Testing on Windows (PowerShell / Command Prompt)

1. **Build `uvm.exe` and `installer.exe`**:
   ```powershell
   go build -o bin/uvm.exe main.go
   go build -o bin/installer.exe cmd/installer/main.go
   $env:Path = "$PWD\bin;$env:Path"
   ```
2. **Verify CLI**:
   ```powershell
   uvm --help
   uvm --version
   ```
3. **Install Node.js versions**:
   ```powershell
   uvm install node 20.11.0
   uvm install node 18.20.0
   ```
4. **List versions**:
   ```powershell
   uvm list node
   ```
5. **Switch active Node.js version**:
   ```powershell
   uvm use node 20.11.0
   node -v    # Output: v20.11.0
   npm -v

   uvm use node 18.20.0
   node -v    # Output: v18.20.0
   ```
6. **Test Visual Web GUI Installer**:
   ```powershell
   .\bin\installer.exe --web
   ```
7. **Run Unit Tests on Windows**:
   ```powershell
   go test -v ./...
   ```

---

## 📂 Project Structure

```
uvm/
├── .github/
│   └── workflows/
│       ├── ci.yml            # Multi-OS CI & test coverage validation
│       └── release.yml       # Automated cross-platform GitHub release pipeline
├── cmd/
│   └── installer/            # Visual / Web GUI installer application
│       ├── main.go
│       └── main_test.go
├── pkg/
│   ├── installer/            # Visual installation engine, Web UI & API
│   │   ├── installer.go
│   │   └── installer_test.go
│   └── node/                 # Node.js runtime manager & extraction engine
│       ├── manager.go
│       └── manager_test.go
├── packaging/
│   ├── homebrew/
│   │   └── uvm.rb            # Homebrew Formula
│   └── scoop/
│       └── uvm.json          # Windows Scoop Manifest
├── scripts/
│   └── build.sh              # Cross-platform build & packaging automation
├── tests/
│   └── test_install.sh       # Installer test suite
├── .gitignore                # Git ignore rules for Go, build artifacts, OS files
├── install.sh                # macOS & Linux installer script
├── install.ps1               # Windows PowerShell installer script
├── Makefile                  # Build, test, install, package commands
├── LICENSE                   # MIT Open Source License
├── README.md                 # Documentation and guide
├── go.mod                    # Go module definitions
├── go.sum                    # Dependency checksums
├── main.go                   # CLI root and command implementations
└── main_test.go              # CLI test suite
```

---

## 📄 License

Distributed under the MIT License. See [`LICENSE`](LICENSE) for more information.
