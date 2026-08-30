# 🚀 uvm (Universal Version Manager)

[![Release](https://img.shields.io/badge/release-v0.0.4-blue.svg)](https://github.com/onlypratyush/UVM-)
[![Go Version](https://img.shields.io/badge/go-1.22+-00ADD8.svg?style=flat&logo=go)](https://golang.org)
[![Coverage](https://img.shields.io/badge/coverage-94.6%25-brightgreen.svg)]()
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-brightgreen.svg)]()

**uvm** (Universal Version Manager) is a fast, cross-platform, and lightweight CLI tool designed to install, manage, and switch between multiple programming language runtimes (**Node.js**, **Go**, **Python**, etc.) seamlessly across **macOS**, **Linux**, and **Windows**.

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
brew trust onlypratyush/uvm
brew install uvm
```

### 🍨 Scoop (Windows)
```powershell
scoop bucket add uvm https://github.com/onlypratyush/UVM-
scoop install uvm
```

---

## 📦 Multi-Language Runtime Management

`uvm` natively downloads, unpacks, shims, and manages official releases for multiple runtimes with **partial version auto-resolution** (e.g. typing `24` resolves automatically to `v24.x.x`):

### 🟢 1. Node.js (`node` / `nodejs`)
```bash
# Install with major prefix, exact version, LTS, or latest
uvm install node 24          # Resolves to latest 24.x.x
uvm install node 20.11.0
uvm install node lts
uvm install node latest

# Switch active version (supports partial prefix e.g. 24 -> v24.20.0)
uvm use node 24
uvm use node 20.11.0

# List installed versions
uvm list node

# List available remote versions from nodejs.org
uvm list-remote node
# or
uvm list --remote node

# Show current version
uvm current node

# Remove version (supports partial prefix)
uvm remove node 24
```

### 🔵 2. Go (`go` / `golang`)
```bash
# Install with prefix (e.g. 1.22 -> go1.22.x) or latest stable
uvm install go 1.22
uvm install go 1.22.0
uvm install go latest

# Switch active version (configures go, gofmt; supports prefix matching)
uvm use go 1.22
uvm use go 1.22.0

# List installed Go versions
uvm list go

# List available remote Go releases from go.dev
uvm list-remote go
# or
uvm list --remote go

# Show current active Go version
uvm current go

# Remove version
uvm remove go 1.22
```

### 🐍 3. Python (`python` / `py` / `python3`)
```bash
# Install with prefix (e.g. 3.12 -> 3.12.9) or latest
uvm install python 3.12
uvm install python 3.12.2
uvm install python latest

# Switch active version (configures python, python3, pip, pip3)
uvm use python 3.12
uvm use python 3.12.2

# List installed Python versions
uvm list python

# List available remote Python releases
uvm list-remote python
# or
uvm list --remote python

# Show current active Python version
uvm current python

# Remove version
uvm remove python 3.12
```

### 🌐 Cross-Runtime Overview
```bash
# List installed versions across ALL managed runtimes
uvm list
# or alias
uvm ls

# List available remote versions across ALL managed runtimes
uvm list-remote
# or alias
uvm ls-remote
uvm list --remote

# Show active versions across ALL managed runtimes
uvm current
```

---

## ⚡ Shell Autocompletion

`uvm` provides dynamic shell autocompletion for commands, runtime names (`node`, `go`, `python`), and version strings:

### 🐚 Enable in Zsh
```bash
echo 'source <(uvm completion zsh)' >> ~/.zshrc
source ~/.zshrc
```

### 🐚 Enable in Bash
```bash
echo 'source <(uvm completion bash)' >> ~/.bashrc
source ~/.bashrc
```

### 🐚 Enable in Fish
```fish
uvm completion fish | source
```

### 🪟 Enable in PowerShell
```powershell
uvm completion powershell | Out-String | Invoke-Expression
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
3. **List remote available versions**:
   ```bash
   uvm list-remote
   uvm list-remote node
   uvm list-remote go
   uvm list-remote python
   ```
4. **Install runtimes using partial version auto-resolution**:
   ```bash
   uvm install node 20
   uvm install go 1.22
   uvm install python 3.12
   ```
5. **Switch and verify binaries with prefix matching**:
   ```bash
   uvm use node 20
   node -v
   npm -v

   uvm use go 1.22
   go version

   uvm use python 3.12
   python3 --version
   pip3 --version
   ```
6. **Run automated unit tests with coverage**:
   ```bash
   make test-coverage
   ./tests/test_install.sh
   ```

---

## 🛠️ Troubleshooting & Common Issues

### 1. `zsh: command not found` or `uvm: command not found`
* **Cause**: Your shell session hasn't loaded `~/.uvm/bin` into its `$PATH`.
* **Fix for macOS / Linux**:
  1. Ensure the following lines are in your `~/.zshrc` or `~/.bashrc`:
     ```bash
     export UVM_INSTALL="$HOME/.uvm"
     export PATH="$HOME/.uvm/bin:$PATH"
     ```
  2. Apply the changes to your current terminal:
     ```bash
     source ~/.zshrc   # or source ~/.bashrc
     ```
  3. Every new terminal tab or window will automatically load `uvm` and active runtime binaries (`node`, `go`, `python`, `npm`, `pip`).

* **Fix for Windows (PowerShell)**:
  1. Restart PowerShell, or apply it to your current session immediately:
     ```powershell
     $env:Path = "$HOME\.uvm\bin;" + $env:Path
     ```

---

### 2. Homebrew: `Refusing to load formula from untrusted tap` or `Repository not found`
* **Cause**: Modern Homebrew (4.0+) requires explicit tap URLs and trust confirmation for third-party taps.
* **Fix**:
  ```bash
  # 1. Tap with the full repository URL
  brew tap onlypratyush/uvm https://github.com/onlypratyush/UVM-

  # 2. Trust the tap
  brew trust onlypratyush/uvm

  # 3. Install uvm
  brew install uvm
  ```

---

### 3. "Existing Runtime Installation Found" During Installation
* **Cause**: UVM detected a pre-existing Node.js, Go, or Python installation.
* **Options Explained**:
  * **`[1] Move to UVM (Recommended)`**: Copies your currently installed runtime into UVM (`~/.uvm/versions/<runtime>/<version>`), activates it, and cleans up legacy PATH entries.
  * **`[2] Delete existing Runtime`**: Safely removes the previous standalone installation and configures UVM as your sole runtime manager.
  * **`[3] Keep existing Runtime`**: Leaves your existing installation untouched and installs UVM alongside it.

---

## 📂 Project Structure

```
uvm/
├── .github/
│   └── workflows/
│       ├── ci.yml            # Multi-OS CI & test coverage validation
│       └── release.yml       # Automated cross-platform GitHub release pipeline
├── Formula/
│   └── uvm.rb                # Direct Homebrew Formula for repository tapping
├── packaging/
│   ├── homebrew/
│   │   └── uvm.rb            # Homebrew Formula
│   └── scoop/
│       └── uvm.json          # Windows Scoop Manifest
├── pkg/
│   ├── golang/               # Go runtime manager & extraction engine
│   │   ├── manager.go
│   │   └── manager_test.go
│   ├── installer/            # Cross-platform runtime detector & safe migration engine
│   │   ├── installer.go
│   │   ├── installer_test.go
│   │   ├── runtime_detector.go
│   │   ├── runtime_migrator.go
│   │   ├── windows_env.go
│   │   ├── windows_env_windows.go
│   │   └── windows_env_other.go
│   ├── node/                 # Node.js runtime manager & extraction engine
│   │   ├── manager.go
│   │   └── manager_test.go
│   └── python/               # Python runtime manager & extraction engine
│       ├── manager.go
│       └── manager_test.go
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
