#!/usr/bin/env bash
# ==============================================================================
# uvm (Universal Version Manager) - macOS & Linux Installer
# ==============================================================================
# Usage (Online):
#   curl -fsSL https://raw.githubusercontent.com/onlypratyush/UVM-/main/install.sh | bash
#
# Usage (Local):
#   ./install.sh
#
# Options:
#   UVM_INSTALL_DIR    Custom install directory (default: $HOME/.uvm/bin)
#   UVM_VERSION        Version to install (default: latest)
#   UVM_NO_MODIFY_PATH Set to 1 to skip modifying shell profile
#   --uninstall, -u    Uninstall uvm
# ==============================================================================

set -e

# Colors for terminal output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

REPO="onlypratyush/UVM-"
DEFAULT_INSTALL_DIR="${HOME}/.uvm/bin"
INSTALL_DIR="${UVM_INSTALL_DIR:-$DEFAULT_INSTALL_DIR}"
BIN_PATH="${INSTALL_DIR}/uvm"
VERSION="${UVM_VERSION:-latest}"

print_banner() {
    echo -e "${CYAN}${BOLD}"
    echo "  ██╗   ██╗██╗   ██╗███╗   ███╗"
    echo "  ██║   ██║██║   ██║████╗ ████║"
    echo "  ██║   ██║██║   ██║██╔████╔██║"
    echo "  ██║   ██║╚██╗ ██╔╝██║╚██╔╝██║"
    echo "  ╚██████╔╝ ╚████╔╝ ██║ ╚═╝ ██║"
    echo "   ╚═════╝   ╚═══╝  ╚═╝     ╚═╝"
    echo -e "  Universal Version Manager${NC}\n"
}

log_info() {
    echo -e "${BLUE}${BOLD}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}${BOLD}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}${BOLD}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}${BOLD}[ERROR]${NC} $1" >&2
}

# Detect operating system
detect_os() {
    local os
    os="$(uname -s)"
    case "${os}" in
        Darwin*)  echo "darwin" ;;
        Linux*)   echo "linux" ;;
        *)
            log_error "Unsupported Operating System: ${os}"
            exit 1
            ;;
    esac
}

# Detect CPU architecture
detect_arch() {
    local arch
    arch="$(uname -m)"
    case "${arch}" in
        x86_64|amd64) echo "amd64" ;;
        arm64|aarch64) echo "arm64" ;;
        armv7l|armv6l) echo "armv7" ;;
        *)
            log_error "Unsupported CPU architecture: ${arch}"
            exit 1
            ;;
    esac
}

# Add directory to shell config
configure_shell_path() {
    if [ "${UVM_NO_MODIFY_PATH:-0}" = "1" ]; then
        log_info "Skipping shell configuration (UVM_NO_MODIFY_PATH=1)."
        return
    fi

    local shell_configs=()
    local user_shell
    user_shell="$(basename "${SHELL:-bash}")"

    # Identify candidate profile files based on available files and current shell
    case "${user_shell}" in
        zsh)
            shell_configs=("${HOME}/.zshrc")
            ;;
        bash)
            if [ -f "${HOME}/.bashrc" ]; then
                shell_configs+=("${HOME}/.bashrc")
            fi
            if [ -f "${HOME}/.bash_profile" ]; then
                shell_configs+=("${HOME}/.bash_profile")
            elif [ ${#shell_configs[@]} -eq 0 ]; then
                shell_configs=("${HOME}/.bash_profile")
            fi
            ;;
        fish)
            shell_configs=("${HOME}/.config/fish/config.fish")
            ;;
        *)
            shell_configs=("${HOME}/.profile" "${HOME}/.bashrc")
            ;;
    esac

    local path_line="export PATH=\"${INSTALL_DIR}:\$PATH\""
    local config_block="
# uvm (Universal Version Manager)
export UVM_INSTALL=\"${INSTALL_DIR%/*}\"
export PATH=\"${INSTALL_DIR}:\$PATH\""

    for config in "${shell_configs[@]}"; do
        local config_dir
        config_dir="$(dirname "${config}")"
        mkdir -p "${config_dir}"
        touch "${config}"

        if grep -q "uvm" "${config}" 2>/dev/null; then
            log_info "uvm PATH configuration already present in ${config}"
        else
            if [[ "${config}" == *fish* ]]; then
                echo -e "\n# uvm (Universal Version Manager)\nfish_add_path ${INSTALL_DIR}" >> "${config}"
            else
                echo "${config_block}" >> "${config}"
            fi
            log_success "Added uvm to PATH in ${config}"
        fi
    done
}

uninstall_uvm() {
    print_banner
    log_info "Uninstalling uvm..."

    if [ -f "${BIN_PATH}" ]; then
        rm -f "${BIN_PATH}"
        log_success "Removed binary: ${BIN_PATH}"
    else
        log_warn "Binary not found at ${BIN_PATH}"
    fi

    local parent_dir="${INSTALL_DIR%/*}"
    if [ -d "${parent_dir}" ] && [ "$(ls -A "${parent_dir}/bin" 2>/dev/null)" = "" ]; then
        rm -rf "${parent_dir}"
        log_success "Cleaned up directory: ${parent_dir}"
    fi

    echo -e "\n${YELLOW}Note: If you want to remove uvm from your shell profile, remove the uvm lines from ~/.zshrc, ~/.bashrc, or ~/.config/fish/config.fish${NC}\n"
    log_success "uvm has been successfully uninstalled!"
    exit 0
}

install_uvm() {
    print_banner
    local os
    local arch
    os="$(detect_os)"
    arch="$(detect_arch)"

    log_info "Target Platform: ${BOLD}${os}/${arch}${NC}"
    log_info "Installation Directory: ${BOLD}${INSTALL_DIR}${NC}"

    mkdir -p "${INSTALL_DIR}"

    local script_dir
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" 2>/dev/null && pwd || echo "")"
    local local_binary="${script_dir}/uvm"
    local installed=0

    # 1. Check if uvm binary already exists locally in script directory
    if [ -n "${script_dir}" ] && [ -f "${local_binary}" ]; then
        log_info "Found local uvm binary at ${local_binary}, installing..."
        cp -f "${local_binary}" "${BIN_PATH}"
        chmod +x "${BIN_PATH}"
        installed=1
    # 2. Check if running inside cloned repo with Go available
    elif [ -n "${script_dir}" ] && [ -f "${script_dir}/main.go" ] && command -v go >/dev/null 2>&1; then
        log_info "Building uvm from source with Go..."
        (cd "${script_dir}" && go build -ldflags="-s -w" -o "${BIN_PATH}" main.go)
        chmod +x "${BIN_PATH}"
        installed=1
    fi

    # 3. If not built locally, download pre-built release binary from GitHub Releases
    if [ "${installed}" -eq 0 ]; then
        local target_name="uvm-${os}-${arch}"
        local download_url
        if [ "${VERSION}" = "latest" ]; then
            download_url="https://github.com/${REPO}/releases/latest/download/${target_name}"
        else
            download_url="https://github.com/${REPO}/releases/download/${VERSION}/${target_name}"
        fi

        log_info "Downloading ${target_name} from GitHub Releases..."
        local tmp_file
        tmp_file="$(mktemp 2>/dev/null || echo "/tmp/uvm_${os}_${arch}_$$")"

        local download_success=0
        if command -v curl >/dev/null 2>&1; then
            if curl -fsSL "${download_url}" -o "${tmp_file}"; then
                download_success=1
            fi
        elif command -v wget >/dev/null 2>&1; then
            if wget -qO "${tmp_file}" "${download_url}"; then
                download_success=1
            fi
        fi

        if [ "${download_success}" -eq 1 ] && [ -s "${tmp_file}" ]; then
            mv "${tmp_file}" "${BIN_PATH}"
            chmod +x "${BIN_PATH}"
            installed=1
        else
            rm -f "${tmp_file}"
            # If download fails (e.g. repo release not yet created) and Go is installed, fallback to go install / build
            if command -v go >/dev/null 2>&1; then
                log_warn "Release binary not found online. Falling back to compiling with local Go..."
                GOBIN="${INSTALL_DIR}" go install "github.com/${REPO}@latest" 2>/dev/null || \
                GOBIN="${INSTALL_DIR}" go install "uvm@latest" 2>/dev/null || true
                if [ -f "${BIN_PATH}" ]; then
                    installed=1
                fi
            fi

            if [ "${installed}" -eq 0 ]; then
                log_error "Could not download or build uvm binary for ${os}/${arch}."
                log_error "Please make sure you have internet access, or install Go (https://go.dev) to build from source."
                exit 1
            fi
        fi
    fi

    configure_shell_path

    echo ""
    log_success "uvm was installed successfully to ${BIN_PATH}!"
    echo -e "\n${BOLD}Quick Start:${NC}"
    echo -e "  1. Restart your terminal or run: ${CYAN}export PATH=\"${INSTALL_DIR}:\$PATH\"${NC}"
    echo -e "  2. Verify installation:        ${CYAN}uvm --help${NC}"
    echo -e "  3. Install a runtime:          ${CYAN}uvm install node 20.11.0${NC}\n"
}

# Parse command-line flags
case "${1:-}" in
    --uninstall|-u)
        uninstall_uvm
        ;;
    --help|-h)
        echo "uvm installer"
        echo "Usage: $0 [OPTIONS]"
        echo "Options:"
        echo "  -h, --help       Show this help message"
        echo "  -u, --uninstall  Uninstall uvm"
        exit 0
        ;;
    *)
        install_uvm
        ;;
esac
