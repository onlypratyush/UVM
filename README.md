# 🚀 uvm (Universal Version Manager)

[![Release](https://img.shields.io/badge/release-v0.0.1-blue.svg)](https://github.com/onlypratyush/UVM-)
[![Go Version](https://img.shields.io/badge/go-1.22+-00ADD8.svg?style=flat&logo=go)](https://golang.org)
[![Coverage](https://img.shields.io/badge/coverage-100%25-brightgreen.svg)]()
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-brightgreen.svg)]()

**uvm** (Universal Version Manager) is a fast, cross-platform, and lightweight CLI tool designed to install, manage, and switch between multiple programming language runtimes (Node.js, Go, Python, Rust, etc.) seamlessly across **macOS**, **Linux**, and **Windows**.

---

## ⚡ Quick Install

### 🍎 macOS & 🐧 Linux (One-Liner)

Run the automated installer in your terminal:

```bash
curl -fsSL https://raw.githubusercontent.com/onlypratyush/UVM-/main/install.sh | bash
```

### 🪟 Windows (PowerShell One-Liner)

Run the installer in PowerShell (run as regular user, no Admin required):

```powershell
irm https://raw.githubusercontent.com/onlypratyush/UVM-/main/install.ps1 | iex
```

---

## 🌐 Cross-Platform Support

`uvm` is built with Go to deliver first-class, native performance across all major operating systems and architectures without external shell dependencies.

| Operating System | Architecture | Supported Shells | Installer |
| :--- | :--- | :--- | :--- |
| 🍎 **macOS** | ARM64 (Apple Silicon M1-M4), x86_64 (Intel) | Zsh, Bash, Fish | `install.sh` / Homebrew |
| 🐧 **Linux** | x86_64 (amd64), ARM64, ARMv7 | Bash, Zsh, Fish | `install.sh` |
| 🪟 **Windows** | x86_64 (amd64), ARM64 | PowerShell, CMD, Windows Terminal | `install.ps1` / Scoop |

---

## ✨ Features

- ⚡ **Lightweight & Fast**: Compiled native binary with zero runtime dependencies.
- 🌍 **True Multi-Platform**: Native installers and binaries on macOS, Linux, and Windows.
- 📦 **Runtime Installation**: Command interface to fetch and setup desired runtime versions.
- 🔄 **Quick Switching**: Switch active versions seamlessly across projects.
- 📋 **Version Listing**: View currently installed and managed language versions.
- 🗑️ **Clean Uninstallation**: Remove runtime versions cleanly when no longer needed.
- 🧪 **100% Test Coverage**: Fully verified CLI command suite with unit and integration tests.
- 🧩 **Extensible CLI**: Powered by [Cobra](https://github.com/spf13/cobra) with auto-generated help menus and shell completions.

---

## 📥 Detailed Installation Methods

### Method 1: Automated Script Installers (Recommended)

#### 🍎 macOS & 🐧 Linux

```bash
# Web one-liner
curl -fsSL https://raw.githubusercontent.com/onlypratyush/UVM-/main/install.sh | bash

# Or from cloned repository
git clone https://github.com/onlypratyush/UVM-.git
cd UVM-
./install.sh
```

The script automatically:
1. Detects your OS and CPU architecture (`darwin-arm64`, `darwin-amd64`, `linux-amd64`, `linux-arm64`).
2. Installs binary to `~/.uvm/bin/uvm`.
3. Adds `~/.uvm/bin` to your shell profile (`~/.zshrc`, `~/.bashrc`, or `~/.config/fish/config.fish`).

#### 🪟 Windows (PowerShell)

```powershell
# Web one-liner
irm https://raw.githubusercontent.com/onlypratyush/UVM-/main/install.ps1 | iex

# Or from cloned repository
git clone https://github.com/onlypratyush/UVM-.git
cd UVM-
powershell -ExecutionPolicy Bypass -File .\install.ps1
```

The script automatically:
1. Detects your Windows architecture (`AMD64` vs `ARM64`).
2. Installs `uvm.exe` to `$HOME\.uvm\bin\uvm.exe`.
3. Permanently configures your User `PATH` environment variable.

---

### Method 2: Package Managers

#### 🍎 Homebrew (macOS & Linux)

```bash
brew install onlypratyush/tap/uvm
# or install from formula
brew install packaging/homebrew/uvm.rb
```

#### 🪟 Scoop (Windows)

```powershell
scoop install packaging/scoop/uvm.json
```

---

### Method 3: Using `make` or `go install`

```bash
# Using make (macOS/Linux)
make install

# Using go install (Cross-platform)
go install github.com/onlypratyush/UVM-@latest
```

---

## 🧪 Testing & 100% Test Coverage

`uvm` features a comprehensive test suite covering 100% of CLI statements and edge cases.

```bash
# Run unit tests
make test

# Run tests and verify 100% statement coverage
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
Install a specified version of any supported language runtime:
```bash
uvm install node 20.11.0
uvm install go 1.22.0
uvm install python 3.12.2
```

#### 2. Switch Active Version
Switch your active environment to a specific installed runtime version:
```bash
uvm use node 20.11.0
uvm use go 1.22.0
```

#### 3. List Installed Runtimes & Versions
Inspect managed runtimes and versions:
```bash
# List all runtimes
uvm list
# or using alias
uvm ls

# List versions for a specific runtime
uvm list node
```

#### 4. Remove / Uninstall a Version
Cleanly remove an installed version:
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
│       ├── ci.yml            # Multi-OS CI & 100% coverage validation
│       └── release.yml       # Automated cross-platform GitHub release pipeline
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
└── main_test.go              # Test suite (100% statement coverage)
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

GitHub Actions will automatically compile binaries for macOS, Linux, and Windows, create `.tar.gz` and `.zip` archives with SHA256 checksums, and publish them to GitHub Releases.

---

## 🗺️ Roadmap

- [x] **v0.0.1**: Initial multi-platform CLI (`install`, `use`, `list`, `remove`), installers (`install.sh`, `install.ps1`), 100% test coverage, and release pipeline.
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
