#!/usr/bin/env bash
# ==============================================================================
# Cross-Platform Build & Packaging Script for uvm CLI & Visual Installer
# ==============================================================================

set -e

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${REPO_ROOT}/dist"
VERSION="${1:-0.0.1}"

# Target platforms and architectures: OS/ARCH/EXT
TARGETS=(
    "darwin/arm64/"
    "darwin/amd64/"
    "linux/amd64/"
    "linux/arm64/"
    "windows/amd64/.exe"
    "windows/arm64/.exe"
)

echo "==> Building uvm & Visual Installer v${VERSION}..."
rm -rf "${DIST_DIR}"
mkdir -p "${DIST_DIR}"

LDFLAGS="-s -w -X main.version=${VERSION}"

for target in "${TARGETS[@]}"; do
    IFS="/" read -r OS ARCH EXT <<< "${target}"
    
    # 1. Build CLI binary
    CLI_OUTPUT_NAME="uvm-${OS}-${ARCH}${EXT}"
    CLI_OUTPUT_PATH="${DIST_DIR}/${CLI_OUTPUT_NAME}"
    echo "  -> Compiling uvm CLI for ${OS}/${ARCH}..."
    GOOS="${OS}" GOARCH="${ARCH}" CGO_ENABLED=0 go build -ldflags="${LDFLAGS}" -o "${CLI_OUTPUT_PATH}" "${REPO_ROOT}/main.go"

    # 2. Build Visual Installer binary
    INSTALLER_OUTPUT_NAME="uvm-installer-${OS}-${ARCH}${EXT}"
    INSTALLER_OUTPUT_PATH="${DIST_DIR}/${INSTALLER_OUTPUT_NAME}"
    echo "  -> Compiling Visual Installer for ${OS}/${ARCH}..."
    GOOS="${OS}" GOARCH="${ARCH}" CGO_ENABLED=0 go build -ldflags="${LDFLAGS}" -o "${INSTALLER_OUTPUT_PATH}" "${REPO_ROOT}/cmd/installer/main.go"

    # Create release archives containing both CLI and Installer
    ARCHIVE_DIR="${DIST_DIR}/uvm_v${VERSION}_${OS}_${ARCH}"
    mkdir -p "${ARCHIVE_DIR}"
    cp "${REPO_ROOT}/README.md" "${ARCHIVE_DIR}/"
    cp "${REPO_ROOT}/LICENSE" "${ARCHIVE_DIR}/"

    if [ "${OS}" = "windows" ]; then
        cp "${CLI_OUTPUT_PATH}" "${ARCHIVE_DIR}/uvm.exe"
        cp "${INSTALLER_OUTPUT_PATH}" "${ARCHIVE_DIR}/installer.exe"
        (cd "${DIST_DIR}" && zip -q -r "uvm_v${VERSION}_${OS}_${ARCH}.zip" "uvm_v${VERSION}_${OS}_${ARCH}")
    else
        cp "${CLI_OUTPUT_PATH}" "${ARCHIVE_DIR}/uvm"
        cp "${INSTALLER_OUTPUT_PATH}" "${ARCHIVE_DIR}/installer"
        (cd "${DIST_DIR}" && tar -czf "uvm_v${VERSION}_${OS}_${ARCH}.tar.gz" "uvm_v${VERSION}_${OS}_${ARCH}")
    fi

    rm -rf "${ARCHIVE_DIR}"
done

# Create convenient root aliases in dist
if [ -f "${DIST_DIR}/uvm-windows-amd64.exe" ]; then
    cp "${DIST_DIR}/uvm-installer-windows-amd64.exe" "${DIST_DIR}/installer.exe"
fi

echo "==> Generating SHA256 Checksums..."
(
    cd "${DIST_DIR}"
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum uvm* installer.exe 2>/dev/null > checksums.txt || sha256sum uvm* > checksums.txt
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 uvm* installer.exe 2>/dev/null > checksums.txt || shasum -a 256 uvm* > checksums.txt
    fi
)

echo "==> Build complete! Output files in ${DIST_DIR}:"
ls -la "${DIST_DIR}"
