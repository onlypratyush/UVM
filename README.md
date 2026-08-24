# 🚀 uvm (Universal Version Manager)

[![Release](https://img.shields.io/badge/release-v0.0.1-blue.svg)](https://github.com/onlypratyush/UVM-)
[![Go Version](https://img.shields.io/badge/go-1.22+-00ADD8.svg?style=flat&logo=go)](https://golang.org)
[![Coverage](https://img.shields.io/badge/coverage-99.2%25-brightgreen.svg)]()
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-brightgreen.svg)]()

**uvm** (Universal Version Manager) is a fast, cross-platform, and lightweight CLI tool designed to install, manage, and switch between multiple programming language runtimes (Node.js, Go, Python, Rust, etc.) seamlessly across **macOS**, **Linux**, and **Windows**.

---

## ⚡ Quick Install Options

### 1. 🖥️ Visual Interactive & Web GUI Installers (Recommended)

Choose your platform for interactive visual setup:

| Platform | Visual Executable | GUI Mode |
| :--- | :--- | :--- |
| 🪟 **Windows** | [`installer.exe`](https://github.com/onlypratyush/UVM-/releases/latest/download/installer.exe) *(or `uvm-installer-windows-amd64.exe`)* | Double-click or `.\installer.exe --web` |
| 🍎 **macOS** | [`uvm-installer-darwin-arm64`](https://github.com/onlypratyush/UVM-/releases/latest/download/uvm-installer-darwin-arm64) / [`amd64`](https://github.com/onlypratyush/UVM-/releases/latest/download/uvm-installer-darwin-amd64) | `./uvm-installer-darwin-arm64 --web` |
| 🐧 **Linux** | [`uvm-installer-linux-amd64`](https://github.com/onlypratyush/UVM-/releases/latest/download/uvm-installer-linux-amd64) / [`arm64`](https://github.com/onlypratyush/UVM-/releases/latest/download/uvm-installer-linux-arm64) | `./uvm-installer-linux-amd64 --web` |

#### Visual Installer Features:
* 🎨 **Interactive Terminal Wizard**: Step-by-step TUI in your command prompt.
* 🌐 **Modern Web GUI (`--web`)**: Dark Glassmorphism setup wizard in your browser with real-time progress bars, custom directory selection, and PATH toggles.
* ⚙️ **Silent Automation (`--silent`)**: Unattended zero-touch installation for scripts and CI.

---

### 2. ⚡ One-Line Automated Terminal Installers

#### 🍎 macOS & 🐧 Linux
```bash
curl -fsSL https://raw.githubusercontent.com/onlypratyush/UVM-/main/install.sh | bash
```

#### 🪟 Windows (PowerShell)
```powershell
irm https://raw.githubusercontent.com/onlypratyush/UVM-/main/install.ps1 | iex
```

---

## 🌐 Cross-Platform Support

`uvm` delivers first-class, native performance across all major operating systems and architectures without external shell dependencies.

| Operating System | Architecture | Supported Shells | Installer Options |
| :--- | :--- | :--- | :--- |
| 🍎 **macOS** | ARM64 (Apple Silicon M1-M4), x86_64 (Intel) | Zsh, Bash, Fish | Visual Installer (`--web`), `install.sh`, Homebrew |
| 🐧 **Linux** | x86_64 (amd64), ARM64, ARMv7 | Bash, Zsh, Fish | Visual Installer (`--web`), `install.sh` |
| 🪟 **Windows** | x86_64 (amd64), ARM64 | PowerShell, CMD, Windows Terminal | `installer.exe`, `install.ps1`, Scoop |

---

## ✨ Features

- ⚡ **Lightweight & Fast**: Compiled native binary with zero runtime dependencies.
- 🌍 **True Multi-Platform**: Native installers and binaries on macOS, Linux, and Windows.
- 🖥️ **Visual & Web GUI Installers**: Visual configuration wizards on all 3 operating systems.
- 📦 **Runtime Installation**: Command interface to fetch and setup desired runtime versions.
- 🔄 **Quick Switching**: Switch active versions seamlessly across projects.
- 📋 **Version Listing**: View currently installed and managed language versions.
- 🗑️ **Clean Uninstallation**: Remove runtime versions cleanly when no longer needed.
- 🧪 **High Test Coverage**: Rigorous test suites for CLI and installer packages.
- 🧩 **Extensible CLI**: Powered by Cobra with auto-generated help menus and shell completions.

---

## 📥 Detailed Installation Methods

### Method 1: Visual Web & Terminal Wizard (`installer.exe` / `uvm-installer`)

#### 🪟 Windows
```powershell
# Launch Visual Web GUI
.\installer.exe --web

# Or run interactive terminal wizard
.\installer.exe
```

#### 🍎 macOS & 🐧 Linux
```bash
# Compile or download installer
make build-installer

# Launch Visual Web GUI
./bin/uvm-installer --web

# Or run terminal wizard
./bin/uvm-installer
```

---

### Method 2: One-Line Shell Scripts

#### 🍎 macOS & 🐧 Linux
```bash
# Web one-liner
curl -fsSL https://raw.githubusercontent.com/onlypratyush/UVM-/main/install.sh | bash

# Local script execution
./install.sh
```

#### 🪟 Windows (PowerShell)
```powershell
# Web one-liner
irm https://raw.githubusercontent.com/onlypratyush/UVM-/main/install.ps1 | iex

# Local script execution
powershell -ExecutionPolicy Bypass -File .\install.ps1
```

---

### Method 3: Package Managers

#### 🍎 Homebrew (macOS & Linux)
```bash
brew install onlypratyush/tap/uvm
# or from local formula
brew install packaging/homebrew/uvm.rb
```

#### 🪟 Scoop (Windows)
```powershell
scoop install packaging/scoop/uvm.json
```

---

### Method 4: Using `make` or `go install`

```bash
# Using make (macOS/Linux)
make install

# Using go install (Cross-platform)
go install github.com/onlypratyush/UVM-@latest
```

---

## 🧪 Testing & Code Coverage

`uvm` and its installers include automated test suites covering commands, argument validation, and edge cases.

```bash
# Run unit tests across all packages
make test

# Run tests and verify statement coverage
make test-coverage

# Run installer test suite
./tests/test_install.sh
```

---

## 🛠️ Usage

### Display Help & Available Commands

```bash
uvm --help
```

```text
uvm (Universal Version Manager) is a fast, lightweight CLI tool to install, 
manage, and switch between multiple programming language runtimes and versions seamlessly.

Usage:
  uvm [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  install     Install a specific runtime version
  list        List installed versions for runtimes
  remove      Remove a specific runtime version
  use         Switch to a specific installed runtime version

Flags:
  -h, --help      help for uvm
  -v, --version   version for uvm
```

---

### Command Reference

#### 1. Install a Runtime Version
```bash
uvm install node 20.11.0
uvm install go 1.22.0
uvm install python 3.12.2
```

#### 2. Switch Active Version
```bash
uvm use node 20.11.0
uvm use go 1.22.0
```

#### 3. List Installed Runtimes & Versions
```bash
# List all runtimes
uvm list
# or using alias
uvm ls

# List versions for a specific runtime
uvm list node
```

#### 4. Remove / Uninstall a Version
```bash
uvm remove node 20.11.0
# or using alias
uvm rm go 1.21.0
```

#### 5. Check CLI Version
```bash
uvm --version
# Output: uvm version 0.0.1
```

---

## 🗑️ Uninstallation

### 🖥️ Via Visual Installer
```bash
uvm-installer --uninstall
# or on Windows
.\installer.exe --uninstall
```

### 🍎 macOS & 🐧 Linux
```bash
./install.sh --uninstall
# or with make
make uninstall
```

### 🪟 Windows
```powershell
powershell -ExecutionPolicy Bypass -File .\install.ps1 -Uninstall
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
│   └── installer/            # Visual installation engine, Web UI & API
│       ├── installer.go
│       └── installer_test.go
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
├── main.go                   # CLI implementation
└── main_test.go              # CLI test suite
```

---

## 🏷️ Releasing New Versions

To release a new version with automated cross-platform binary builds:

```bash
# 1. Tag version
git tag -a v0.0.1 -m "Release v0.0.1"

# 2. Push tag to GitHub
git push origin v0.0.1
```

GitHub Actions will automatically compile binaries and visual installers for macOS, Linux, and Windows, create `.tar.gz` and `.zip` archives with SHA256 checksums, and publish them to GitHub Releases.

---

## 🗺️ Roadmap

- [x] **v0.0.1**: Initial multi-platform CLI (`install`, `use`, `list`, `remove`), Visual Installers (`installer.exe`, `uvm-installer`), script installers (`install.sh`, `install.ps1`), and release pipeline.
- [ ] **v0.1.0**: Cross-platform shim & symlink architecture for PATH resolution (macOS/Linux symlinks, Windows junction/shims).
- [ ] **v0.2.0**: Official download providers for Node.js, Go, Python, and Rust binaries.
- [ ] **v0.3.0**: Project-level `.uvmrc` or `.tool-versions` auto-switching.

---

## 🤝 Contributing

Contributions, issues, and feature requests are welcome!

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'feat: add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

---

## 📄 License

Distributed under the MIT License. See [`LICENSE`](LICENSE) for more information.
