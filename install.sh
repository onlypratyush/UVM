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
#   UVM_NODE_ACTION    Action for existing Node.js: move, delete, keep
#   --node-action      move, delete, keep
#   --confirm-delete   Confirm deletion of existing Node.js
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
NODE_ACTION="${UVM_NODE_ACTION:-}"
CONFIRM_DELETE="${UVM_CONFIRM_DELETE:-0}"

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

detect_existing_node() {
    local node_bin=""
    local candidates=(
        "$(command -v node 2>/dev/null || true)"
        "${NVM_BIN}/node"
        "/opt/homebrew/bin/node"
        "/usr/local/bin/node"
        "/usr/bin/node"
        "${HOME}/.nvm/current/bin/node"
        "${HOME}/.local/share/fnm/current/bin/node"
        "${HOME}/.n/bin/node"
        "${HOME}/.volta/bin/node"
        "${HOME}/.asdf/shims/node"
    )

    # Check NVM versions directories
    for nvm_node in "${HOME}"/.nvm/versions/node/*/bin/node; do
        if [ -x "${nvm_node}" ]; then
            candidates+=("${nvm_node}")
        fi
    done

    for c in "${candidates[@]}"; do
        if [ -n "${c}" ] && [ -x "${c}" ]; then
            # Skip if inside .uvm
            if [[ "${c}" == *".uvm"* ]]; then
                continue
            fi
            local v
            v="$("${c}" -v 2>/dev/null || true)"
            if [[ "${v}" == v* ]]; then
                node_bin="${c}"
                break
            fi
        fi
    done

    echo "${node_bin}"
}

migrate_node_to_uvm() {
    local node_exe="$1"
    local uvm_base="$2"
    local ver
    ver="$("${node_exe}" -v 2>/dev/null || true)"
    local node_dir
    node_dir="$(dirname "$(dirname "${node_exe}")")"
    if [ "$(basename "$(dirname "${node_exe}")")" != "bin" ]; then
        node_dir="$(dirname "${node_exe}")"
    fi

    local target_dir="${uvm_base}/versions/node/${ver}"
    log_info "[1/6] Copying existing Node.js (${ver}) to ${target_dir}..."
    mkdir -p "${target_dir}"

    cp -R "${node_dir}/"* "${target_dir}/" 2>/dev/null || cp -R "${node_dir}"/* "${target_dir}/"

    # Verify Step 2
    log_info "[2/6] Verifying copied node binary..."
    local copied_bin="${target_dir}/bin/node"
    if [ ! -f "${copied_bin}" ]; then
        copied_bin="${target_dir}/node"
    fi

    if [ ! -x "${copied_bin}" ] || [ "$("${copied_bin}" -v 2>/dev/null)" != "${ver}" ]; then
        rm -rf "${target_dir}"
        log_error "Copied Node.js verification failed. Rollback completed."
        return 1
    fi

    # Activate in UVM Step 4
    log_info "[4/6] Activating Node.js ${ver} in UVM..."
    mkdir -p "${uvm_base}/current"
    echo "${ver}" > "${uvm_base}/current/node.version"
    mkdir -p "${uvm_base}/bin"

    for b in node npm npx corepack; do
        if [ -f "${target_dir}/bin/${b}" ]; then
            ln -sf "${target_dir}/bin/${b}" "${uvm_base}/bin/${b}"
        elif [ -f "${target_dir}/${b}" ]; then
            ln -sf "${target_dir}/${b}" "${uvm_base}/bin/${b}"
        fi
    done

    # Verify Step 5
    log_info "[5/6] Verifying UVM active node binary..."
    if [ "$("${uvm_base}/bin/node" -v 2>/dev/null)" != "${ver}" ]; then
        rm -rf "${target_dir}"
        log_error "UVM active node verification failed."
        return 1
    fi

    log_success "Node.js ${ver} was successfully migrated to UVM!"
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
    local uvm_base="${INSTALL_DIR%/*}"

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
        local download_urls=(
            "https://github.com/${REPO}/releases/latest/download/${target_name}"
            "https://github.com/onlypratyush/UVM/releases/latest/download/${target_name}"
            "https://github.com/${REPO}/releases/download/${VERSION}/${target_name}"
        )

        local tmp_file
        tmp_file="$(mktemp 2>/dev/null || echo "/tmp/uvm_${os}_${arch}_$$")"

        for url in "${download_urls[@]}"; do
            log_info "Downloading ${target_name} from GitHub Releases (${url})..."
            if command -v curl >/dev/null 2>&1; then
                if curl -fsSL "${url}" -o "${tmp_file}" 2>/dev/null && [ -s "${tmp_file}" ]; then
                    mv "${tmp_file}" "${BIN_PATH}"
                    chmod +x "${BIN_PATH}"
                    installed=1
                    break
                fi
            elif command -v wget >/dev/null 2>&1; then
                if wget -qO "${tmp_file}" "${url}" 2>/dev/null && [ -s "${tmp_file}" ]; then
                    mv "${tmp_file}" "${BIN_PATH}"
                    chmod +x "${BIN_PATH}"
                    installed=1
                    break
                fi
            fi
        done

        rm -f "${tmp_file}"

        # Fallback to Go install / build if present
        if [ "${installed}" -eq 0 ] && command -v go >/dev/null 2>&1; then
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

    # 4. Check for existing Node.js
    local detected_node
    detected_node="$(detect_existing_node)"
    if [ -n "${detected_node}" ]; then
        local node_ver
        node_ver="$("${detected_node}" -v 2>/dev/null || true)"
        echo ""
        echo -e "${YELLOW}${BOLD}┌─────────────────────────────────────────────────────────────┐${NC}"
        echo -e "${YELLOW}${BOLD}│  Existing Node.js Installation Found                        │${NC}"
        printf "${YELLOW}${BOLD}│${NC}  Version:  ${GREEN}%-49s${NC}${YELLOW}${BOLD}│${NC}\n" "${node_ver}"
        printf "${YELLOW}${BOLD}│${NC}  Location: %-49s${YELLOW}${BOLD}│${NC}\n" "${detected_node}"
        echo -e "${YELLOW}${BOLD}└─────────────────────────────────────────────────────────────┘${NC}"
        echo ""

        local action="${NODE_ACTION}"
        local has_tty=0
        if [ -t 0 ]; then
            has_tty=1
        elif [ -r /dev/tty ] && [ -w /dev/tty ]; then
            has_tty=2
        fi

        if [ -z "${action}" ] && [ "${has_tty}" -gt 0 ]; then
            echo "How would you like UVM to handle it?"
            echo -e "  ${GREEN}[1] Move to UVM (Recommended)${NC} - Keep current Node.js and let UVM manage it"
            echo "  [2] Delete existing Node.js   - Remove existing installation"
            echo "  [3] Keep existing Node.js     - Leave existing installation unchanged"

            local choice="1"
            if [ "${has_tty}" -eq 1 ]; then
                read -r -p "Choice [1-3] (Default: 1): " choice || choice="1"
            else
                read -r -p "Choice [1-3] (Default: 1): " choice < /dev/tty 2>/dev/null || choice="1"
            fi

            case "${choice}" in
                2)
                    local confirm=""
                    if [ "${has_tty}" -eq 1 ]; then
                        read -r -p "Are you sure you want to delete existing Node.js? [y/N]: " confirm || confirm="n"
                    else
                        read -r -p "Are you sure you want to delete existing Node.js? [y/N]: " confirm < /dev/tty 2>/dev/null || confirm="n"
                    fi
                    if [[ "${confirm}" =~ ^[Yy] ]]; then
                        action="delete"
                        CONFIRM_DELETE="1"
                    else
                        action="move"
                    fi
                    ;;
                3) action="keep" ;;
                *) action="move" ;;
            esac
        elif [ -z "${action}" ]; then
            action="move"
        fi

        case "${action}" in
            move)
                migrate_node_to_uvm "${detected_node}" "${uvm_base}" || true
                ;;
            delete)
                if [ "${CONFIRM_DELETE}" = "1" ]; then
                    log_info "Removing existing Node.js..."
                    rm -f "${detected_node}" 2>/dev/null || true
                    log_success "Existing Node.js removed."
                else
                    log_warn "Deletion skipped (confirmation not provided)."
                fi
                ;;
            keep)
                log_info "Leaving existing Node.js installation unchanged."
                ;;
        esac
    fi

    configure_shell_path

    echo ""
    log_success "uvm was installed successfully to ${BIN_PATH}!"
    echo -e "\n${BOLD}Quick Start:${NC}"
    echo -e "  1. Restart your terminal or run: ${CYAN}export PATH=\"${INSTALL_DIR}:\$PATH\"${NC}"
    echo -e "  2. Verify installation:        ${CYAN}uvm --help${NC}"
    echo -e "  3. Manage runtimes:            ${CYAN}uvm list node${NC}"
    echo -e "  4. Install a runtime:          ${CYAN}uvm install node 20.11.0${NC}\n"
}

# Parse command-line flags
while [[ $# -gt 0 ]]; do
    case "$1" in
        --node-action)
            NODE_ACTION="$2"
            shift 2
            ;;
        --confirm-delete)
            CONFIRM_DELETE="1"
            shift
            ;;
        --uninstall|-u)
            uninstall_uvm
            ;;
        --help|-h)
            echo "uvm installer"
            echo "Usage: $0 [OPTIONS]"
            echo "Options:"
            echo "  --node-action <move|delete|keep>  Action for existing Node.js"
            echo "  --confirm-delete                  Confirm deletion of existing Node.js"
            echo "  -h, --help                        Show this help message"
            echo "  -u, --uninstall                   Uninstall uvm"
            exit 0
            ;;
        *)
            shift
            ;;
    esac
done

install_uvm
