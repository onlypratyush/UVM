# 🚀 uvm (Universal Version Manager)

[![Release](https://img.shields.io/badge/release-v0.0.1-blue.svg)](https://github.com/)
[![Go Version](https://img.shields.io/badge/go-1.22+-00ADD8.svg?style=flat&logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-brightgreen.svg)]()

**uvm** (Universal Version Manager) is a fast, cross-platform, and lightweight CLI tool designed to install, manage, and switch between multiple programming language runtimes (Node.js, Go, Python, Rust, etc.) seamlessly across **macOS**, **Linux**, and **Windows**.

---

## 🌐 Cross-Platform Support

`uvm` is built with Go to deliver first-class, native performance across all major operating systems and architectures without external shell dependencies.

| Operating System | Architecture | Supported Shells |
| :--- | :--- | :--- |
| 🍎 **macOS** | ARM64 (Apple Silicon M1/M2/M3/M4), x86_64 (Intel) | Zsh, Bash, Fish |
| 🐧 **Linux** | x86_64 (amd64), ARM64, ARMv7 | Bash, Zsh, Fish |
| 🪟 **Windows** | x86_64 (amd64), ARM64 | PowerShell, CMD, Git Bash, WSL |

---

## ✨ Features (v0.0.1)

- ⚡ **Lightweight & Fast**: Compiled native binary with zero runtime dependencies.
- 🌍 **True Multi-Platform**: Native support on macOS, Linux, and Windows.
- 📦 **Runtime Installation**: Command interface to fetch and setup desired runtime versions.
- 🔄 **Quick Switching**: Switch active versions seamlessly across projects.
- 📋 **Version Listing**: View currently installed and managed language versions.
- 🗑️ **Clean Uninstallation**: Remove runtime versions cleanly when no longer needed.
- 🧩 **Extensible CLI**: Powered by [Cobra](https://github.com/spf13/cobra) with auto-generated help menus and shell completions.

---

## 📥 Installation

### Prerequisites
- [Go](https://go.dev/dl/) 1.22 or higher

---

### Method 1: Using `go install` (Recommended)

Works on macOS, Linux, and Windows:

```bash
go install uvm@latest
```

---

### Method 2: Build from Source

#### 🍎 macOS & 🐧 Linux

```bash
# Clone the repository
git clone https://github.com/<your-username>/uvm.git
cd uvm

# Download dependencies
go mod download

# Build binary
go build -o uvm main.go

# (Optional) Move to system PATH
sudo mv uvm /usr/local/bin/
```

#### 🪟 Windows (PowerShell / Command Prompt)

```powershell
# Clone the repository
git clone https://github.com/<your-username>/uvm.git
cd uvm

# Download dependencies
go mod download

# Build binary
go build -o uvm.exe main.go

# (Optional) Move to a folder in your PATH (e.g., C:\tools\uvm or user bin)
New-Item -ItemType Directory -Force -Path "$HOME\bin"
Move-Item .\uvm.exe "$HOME\bin\uvm.exe"
```

---

### 📦 Cross-Compilation

To build binaries for all supported platforms from any operating system:

```bash
# macOS (Apple Silicon & Intel)
GOOS=darwin GOARCH=arm64 go build -o dist/uvm-darwin-arm64 main.go
GOOS=darwin GOARCH=amd64 go build -o dist/uvm-darwin-amd64 main.go

# Linux (amd64 & arm64)
GOOS=linux GOARCH=amd64 go build -o dist/uvm-linux-amd64 main.go
GOOS=linux GOARCH=arm64 go build -o dist/uvm-linux-arm64 main.go

# Windows (64-bit & ARM64)
GOOS=windows GOARCH=amd64 go build -o dist/uvm-windows-amd64.exe main.go
GOOS=windows GOARCH=arm64 go build -o dist/uvm-windows-arm64.exe main.go
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

## 📂 Project Structure

```
uvm/
├── .gitignore        # Git ignore rules for Go, build artifacts, and OS files
├── LICENSE           # MIT Open Source License
├── README.md         # Documentation and cross-platform guide
├── go.mod            # Go module definitions and dependencies
├── go.sum            # Dependency checksums
└── main.go           # CLI root and command implementations
```

---

## 🏷️ Releasing v0.0.1

To tag and push the `v0.0.1` release:

```bash
# 1. Stage and commit all changes
git add .
git commit -m "chore(release): prepare v0.0.1"

# 2. Create git tag
git tag -a v0.0.1 -m "Release v0.0.1"

# 3. Push commits and tags to remote
git push origin main
git push origin v0.0.1
```

---

## 🗺️ Roadmap

- [x] **v0.0.1**: Initial multi-platform CLI (`install`, `use`, `list`, `remove`), flags, and versioning.
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
