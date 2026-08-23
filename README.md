# 🚀 uvm (Universal Version Manager)

[![Release](https://img.shields.io/badge/release-v0.0.1-blue.svg)](https://github.com/)
[![Go Version](https://img.shields.io/badge/go-1.22+-00ADD8.svg?style=flat&logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey.svg)]()

**uvm** is a fast, lightweight, and extensible CLI tool designed to install, manage, and switch between multiple programming language runtimes (Node.js, Go, Python, Rust, etc.) effortlessly from a single command line interface.

---

## ✨ Features (v0.0.1)

- ⚡ **Lightweight & Fast**: Written in Go with minimal dependencies.
- 📦 **Runtime Installation**: Command interface to fetch and setup desired runtime versions.
- 🔄 **Quick Switching**: Switch active versions seamlessly across projects.
- 📋 **Version Listing**: View currently installed and managed language versions.
- 🗑️ **Clean Uninstallation**: Remove runtime versions cleanly when no longer needed.
- 🧩 **Extensible CLI**: Powered by [Cobra](https://github.com/spf13/cobra) for rich CLI parsing and help menus.

---

## 📥 Installation

### Prerequisites
- [Go](https://go.dev/dl/) 1.22 or higher

### Build from Source

```bash
# Clone the repository
git clone https://github.com/<your-username>/uvm.git
cd uvm

# Download dependencies
go mod download

# Build the binary
go build -o uvm main.go

# (Optional) Move to your system PATH
sudo mv uvm /usr/local/bin/
```

### Install with `go install`

```bash
go install uvm@latest
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
Switch your environment to use a specific installed runtime version:
```bash
uvm use node 20.11.0
uvm use go 1.22.0
```

#### 3. List Installed Runtimes & Versions
Inspect your installed runtimes and available versions:
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

#### 5. Check Version
```bash
uvm --version
# Output: uvm version 0.0.1
```

---

## 📂 Project Structure

```
uvm/
├── .gitignore        # Git ignore rules for Go and build binaries
├── LICENSE           # MIT Open Source License
├── README.md         # Documentation and guide
├── go.mod            # Go module definitions and dependencies
├── go.sum            # Dependency checksums
└── main.go           # CLI root and command implementations
```

---

## 🏷️ Releasing v0.0.1

To tag and push the `v0.0.1` release to your Git remote:

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

- [x] v0.0.1: Initial CLI structure, commands (`install`, `use`, `list`, `remove`), flags, and versioning.
- [ ] v0.1.0: Shim & symlink architecture for PATH resolution.
- [ ] v0.2.0: Official download providers for Node.js, Go, Python, and Rust binaries.
- [ ] v0.3.0: Project-level `.uvmrc` or `.tool-versions` auto-switching.

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
