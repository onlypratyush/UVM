#!/usr/bin/env bash
# ==============================================================================
# Installer Test Suite for install.sh
# ==============================================================================

set -e

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INSTALLER="${REPO_ROOT}/install.sh"
TEMP_DIR="$(mktemp -d 2>/dev/null || mktemp -d -t 'uvm_test')"

cleanup() {
    if [ -d "${TEMP_DIR}" ]; then
        chmod -R u+w "${TEMP_DIR}" 2>/dev/null || true
        rm -rf "${TEMP_DIR}" 2>/dev/null || true
    fi
}
trap cleanup EXIT

PASSED_TESTS=0
FAILED_TESTS=0

run_test() {
    local test_name="$1"
    shift
    echo -n "  Testing: ${test_name}... "
    if "$@"; then
        echo -e "\033[0;32mPASSED\033[0m"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "\033[0;31mFAILED\033[0m"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
}

# Test 1: Help menu
test_help_flag() {
    local output
    output="$("${INSTALLER}" --help)"
    echo "${output}" | grep -q "uvm installer"
}

# Test 2: Local install into isolated environment
test_local_install() {
    local test_home="${TEMP_DIR}/home1"
    mkdir -p "${test_home}"
    local test_install_dir="${test_home}/.uvm/bin"

    HOME="${test_home}" GOPATH="${GOPATH:-$HOME/go}" UVM_INSTALL_DIR="${test_install_dir}" "${INSTALLER}" > /dev/null

    # Assert binary exists and is executable
    [ -x "${test_install_dir}/uvm" ] || return 1

    # Assert binary runs
    local ver_out
    ver_out="$("${test_install_dir}/uvm" --version)"
    echo "${ver_out}" | grep -q "uvm version" || return 1

    # Assert shell config created and contains uvm
    grep -q "uvm" "${test_home}/.zshrc" || grep -q "uvm" "${test_home}/.bashrc" || grep -q "uvm" "${test_home}/.bash_profile" || return 1
}

# Test 3: Idempotent PATH modification
test_idempotent_shell_config() {
    local test_home="${TEMP_DIR}/home2"
    mkdir -p "${test_home}"
    local test_install_dir="${test_home}/.uvm/bin"
    touch "${test_home}/.zshrc"

    HOME="${test_home}" GOPATH="${GOPATH:-$HOME/go}" SHELL="/bin/zsh" UVM_INSTALL_DIR="${test_install_dir}" "${INSTALLER}" > /dev/null
    local count1
    count1="$(grep -c "export UVM_INSTALL" "${test_home}/.zshrc")"

    # Run second time
    HOME="${test_home}" GOPATH="${GOPATH:-$HOME/go}" SHELL="/bin/zsh" UVM_INSTALL_DIR="${test_install_dir}" "${INSTALLER}" > /dev/null
    local count2
    count2="$(grep -c "export UVM_INSTALL" "${test_home}/.zshrc")"

    [ "${count1}" -eq 1 ] && [ "${count2}" -eq 1 ]
}

# Test 4: Skip PATH modification when requested
test_skip_path_modify() {
    local test_home="${TEMP_DIR}/home3"
    mkdir -p "${test_home}"
    local test_install_dir="${test_home}/.uvm/bin"

    HOME="${test_home}" GOPATH="${GOPATH:-$HOME/go}" UVM_INSTALL_DIR="${test_install_dir}" UVM_NO_MODIFY_PATH=1 "${INSTALLER}" > /dev/null
    [ -x "${test_install_dir}/uvm" ] || return 1
    if [ -f "${test_home}/.zshrc" ]; then
        ! grep -q "uvm" "${test_home}/.zshrc" || return 1
    fi
}

# Test 5: Uninstallation
test_uninstall() {
    local test_home="${TEMP_DIR}/home4"
    mkdir -p "${test_home}"
    local test_install_dir="${test_home}/.uvm/bin"

    HOME="${test_home}" GOPATH="${GOPATH:-$HOME/go}" UVM_INSTALL_DIR="${test_install_dir}" "${INSTALLER}" > /dev/null
    [ -f "${test_install_dir}/uvm" ] || return 1

    # Run uninstall
    HOME="${test_home}" GOPATH="${GOPATH:-$HOME/go}" UVM_INSTALL_DIR="${test_install_dir}" "${INSTALLER}" --uninstall > /dev/null

    # Assert binary was removed
    [ ! -f "${test_install_dir}/uvm" ]
}

# Test 6: Node Action Keep
test_node_action_keep() {
    local test_home="${TEMP_DIR}/home5"
    mkdir -p "${test_home}"
    local test_install_dir="${test_home}/.uvm/bin"

    HOME="${test_home}" GOPATH="${GOPATH:-$HOME/go}" UVM_INSTALL_DIR="${test_install_dir}" "${INSTALLER}" --node-action keep > /dev/null
    [ -x "${test_install_dir}/uvm" ] || return 1
}

echo "=================================================="
echo "Running install.sh Test Suite"
echo "=================================================="
run_test "Help Flag" test_help_flag
run_test "Local Installation" test_local_install
run_test "Idempotent Shell Config" test_idempotent_shell_config
run_test "Skip PATH Modification" test_skip_path_modify
run_test "Uninstallation" test_uninstall
run_test "Node Action Keep" test_node_action_keep

echo "=================================================="
echo "Test Summary: ${PASSED_TESTS} passed, ${FAILED_TESTS} failed"
echo "=================================================="

if [ "${FAILED_TESTS}" -gt 0 ]; then
    exit 1
fi
