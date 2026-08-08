#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# Crenox Interactive & Automated Installer Script
# ─────────────────────────────────────────────────────────────────────────────
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/crenoxhq/crenox/main/scripts/install.sh | bash
#
# Non-Interactive Flags:
#   curl ... | bash -s -- --global     # Install binary & set global git hook
#   curl ... | bash -s -- --local      # Install binary & set hook for current repo
#   curl ... | bash -s -- --no-hook    # Install binary only
# ─────────────────────────────────────────────────────────────────────────────

set -euo pipefail

REPO="crenoxhq/crenox"

# Terminal Colors & Aesthetics
COLOR_RESET="\033[0m"
COLOR_BOLD="\033[1m"
COLOR_CYAN="\033[36m"
COLOR_GREEN="\033[32m"
COLOR_RED="\033[31m"
COLOR_YELLOW="\033[33m"
COLOR_BLUE="\033[34m"

# Parse CLI flags passed to the installer
MODE_FLAG=""
CUSTOM_VERSION="${CRENOX_VERSION:-}"
CUSTOM_DIR="${CRENOX_DIR:-}"

for arg in "$@"; do
    case "$arg" in
        --global)  MODE_FLAG="global" ;;
        --local)   MODE_FLAG="local" ;;
        --no-hook) MODE_FLAG="none" ;;
        --version=*) CUSTOM_VERSION="${arg#*=}" ;;
        --dir=*)     CUSTOM_DIR="${arg#*=}" ;;
        *) ;;
    esac
done

print_banner() {
    echo -e "${COLOR_CYAN}${COLOR_BOLD}"
    cat << "EOF"
  ██████╗ ██████╗  ███████╗ ████╗  ██╗  ██████╗  ██╗  ██╗
 ██╔════╝ ██╔══██╗ ██╔════╝ ██╔██╗ ██║ ██╔═══██╗ ╚██╗██╔╝
 ██║      ██████╔╝ █████╗   ██║╚██╗██║ ██║   ██║  ╚███╔╝ 
 ██║      ██╔══██╗ ██╔══╝   ██║ ╚████║ ██║   ██║  ██╔██╗ 
 ╚██████╗ ██║  ██║ ███████╗ ██║  ╚███║ ╚██████╔╝ ██╔╝╚██╗
  ╚═════╝ ╚═╝  ╚═╝ ╚══════╝ ╚═╝   ╚══╝  ╚═════╝  ╚═╝  ╚═╝
EOF
    echo -e "${COLOR_RESET}"
    echo -e "${COLOR_BOLD}Statically Compiled Git Secret Scanner & Pre-Commit Hook${COLOR_RESET}\n"
}

info() {
    echo -e "${COLOR_CYAN}INFO${COLOR_RESET} $1"
}

success() {
    echo -e "${COLOR_GREEN}SUCCESS${COLOR_RESET} $1"
}

warn() {
    echo -e "${COLOR_YELLOW}WARN${COLOR_RESET} $1"
}

error() {
    echo -e "${COLOR_RED}ERROR${COLOR_RESET} $1" >&2
    exit 1
}

print_banner

# 1. Check Git Environment
info "Checking Git environment..."
if ! command -v git >/dev/null 2>&1; then
    warn "Git is not installed on this system!"
    echo -e "   Crenox requires Git to protect your repositories."
    echo -e "   Please install Git via your package manager (e.g. 'apt install git', 'brew install git', or 'pkg install git').\n"
fi

# 2. Detect OS & Architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "${OS}" in
    linux*)   OS="linux" ;;
    darwin*)  OS="darwin" ;;
    msys*|mingw*|cygwin*) OS="windows" ;;
    *) error "Unsupported operating system: ${OS}" ;;
esac

# Special check for Termux / Android environment
if [ -d "/data/data/com.termux" ] || [[ "${PREFIX:-}" == *"com.termux"* ]]; then
    OS="android"
fi

ARCH="$(uname -m)"
case "${ARCH}" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    armv7l|armv6l|arm) ARCH="arm" ;;
    *) error "Unsupported CPU architecture: ${ARCH}" ;;
esac

info "Detected System: ${COLOR_BOLD}${OS}/${ARCH}${COLOR_RESET}"

# 3. Determine Version & Download Binary
TARGET_TAG="${CUSTOM_VERSION}"

if [ -z "${TARGET_TAG}" ]; then
    info "Fetching latest Crenox version..."
    TARGET_TAG=$(curl -s --connect-timeout 5 "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' || true)
fi

if [ -z "${TARGET_TAG}" ]; then
    warn "Could not resolve latest release via API. Falling back to v2.1.5"
    TARGET_TAG="v2.1.5"
fi

info "Installing Crenox version: ${COLOR_BOLD}${TARGET_TAG}${COLOR_RESET}"

if [ "${OS}" = "windows" ]; then
    RELEASE_FILE="crenox-${TARGET_TAG}-${OS}-${ARCH}.exe"
    TARGET_NAME="crenox.exe"
else
    RELEASE_FILE="crenox-${TARGET_TAG}-${OS}-${ARCH}"
    TARGET_NAME="crenox"
fi

DOWNLOAD_URL="https://github.com/crenoxhq/crenox/releases/download/${TARGET_TAG}/${RELEASE_FILE}"

TEMP_DIR=$(mktemp -d)
trap 'rm -rf "${TEMP_DIR}"' EXIT

TEMP_BIN="${TEMP_DIR}/${TARGET_NAME}"

info "Downloading binary artifact from:"
echo -e "   ${COLOR_BLUE}${DOWNLOAD_URL}${COLOR_RESET}"

# Download with retries
DOWNLOAD_SUCCESS=false

if command -v curl >/dev/null 2>&1; then
    if curl -fL --retry 3 --retry-connrefused --connect-timeout 10 "${DOWNLOAD_URL}" -o "${TEMP_BIN}"; then
        DOWNLOAD_SUCCESS=true
    fi
fi

if [ "$DOWNLOAD_SUCCESS" = false ] && command -v wget >/dev/null 2>&1; then
    if wget -q --tries=3 -O "${TEMP_BIN}" "${DOWNLOAD_URL}"; then
        DOWNLOAD_SUCCESS=true
    fi
fi

if [ "$DOWNLOAD_SUCCESS" = false ]; then
    error "Download failed. Please check your network connection or download directly from:\n   ${DOWNLOAD_URL}"
fi

chmod +x "${TEMP_BIN}"

# Verify binary execution
if ! "${TEMP_BIN}" version >/dev/null 2>&1; then
    error "Downloaded binary failed self-test execution."
fi

# 4. Install Executable to PATH
INSTALL_DIR="${CUSTOM_DIR}"

if [ -z "${INSTALL_DIR}" ]; then
    if [ "${OS}" = "android" ] && [ -n "${PREFIX:-}" ]; then
        INSTALL_DIR="${PREFIX}/bin"
    elif [ -w "/usr/local/bin" ]; then
        INSTALL_DIR="/usr/local/bin"
    elif [ -d "$HOME/.local/bin" ] || mkdir -p "$HOME/.local/bin" 2>/dev/null; then
        INSTALL_DIR="$HOME/.local/bin"
    else
        INSTALL_DIR="/usr/local/bin"
    fi
fi

mkdir -p "${INSTALL_DIR}" 2>/dev/null || true

info "Installing executable to ${COLOR_BOLD}${INSTALL_DIR}/${TARGET_NAME}${COLOR_RESET}..."

if [ -w "${INSTALL_DIR}" ]; then
    mv "${TEMP_BIN}" "${INSTALL_DIR}/${TARGET_NAME}"
else
    info "Elevated permissions required to install into ${INSTALL_DIR}"
    sudo mv "${TEMP_BIN}" "${INSTALL_DIR}/${TARGET_NAME}"
fi

INSTALLED_PATH="${INSTALL_DIR}/${TARGET_NAME}"
success "Crenox binary installed successfully at ${INSTALLED_PATH}"

# Warn if directory is not in PATH
if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
    warn "${INSTALL_DIR} is not in your current PATH."
    echo -e "   Add it to your shell configuration file (~/.bashrc or ~/.zshrc):"
    echo -e "     ${COLOR_CYAN}export PATH=\"${INSTALL_DIR}:\$PATH\"${COLOR_RESET}\n"
fi

# 5. Interactive or Flag-Based Git Hook Configuration
echo
info "Configuring Git Hook protection..."

HOOK_CHOICE="${MODE_FLAG}"

# If running interactively and no flag was specified, ask the user
if [ -z "${HOOK_CHOICE}" ] && [ -t 0 ]; then
    echo -e "${COLOR_BOLD}Select how you would like to enable Crenox Git protection:${COLOR_RESET}"
    echo -e "  ${COLOR_CYAN}1)${COLOR_RESET} Protect current repository (${COLOR_BOLD}crenox install${COLOR_RESET})"
    echo -e "  ${COLOR_CYAN}2)${COLOR_RESET} Protect ALL Git repositories globally (${COLOR_BOLD}crenox install --global${COLOR_RESET})"
    echo -e "  ${COLOR_CYAN}3)${COLOR_RESET} Skip hook setup for now (Binary only)"
    read -rp "Enter choice [1-3] (default: 1): " USER_INPUT
    case "${USER_INPUT}" in
        2) HOOK_CHOICE="global" ;;
        3) HOOK_CHOICE="none" ;;
        *) HOOK_CHOICE="local" ;;
    esac
fi

# Default to local if still empty (e.g. piped script without flags)
if [ -z "${HOOK_CHOICE}" ]; then
    HOOK_CHOICE="local"
fi

case "${HOOK_CHOICE}" in
    global)
        info "Enabling Crenox Git hook globally for all repositories..."
        if "${INSTALLED_PATH}" install --global; then
            success "Crenox global Git hook enabled! Every git repository on this machine is now protected."
        else
            warn "Failed to set global Git hook automatically. You can run 'crenox install --global' manually."
        fi
        ;;

    local)
        info "Checking current directory for Git repository..."
        if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
            if "${INSTALLED_PATH}" install; then
                success "Crenox Git hook installed in current repository!"
            else
                warn "Could not install hook. You can run 'crenox install' manually inside your repo."
            fi
        else
            warn "Current directory is NOT a Git repository."
            echo -e "${COLOR_YELLOW}Guidance:${COLOR_RESET}"
            echo -e "  To protect a repository, navigate to your project folder and run:"
            echo -e "     ${COLOR_CYAN}cd /path/to/your/project${COLOR_RESET}"
            echo -e "     ${COLOR_CYAN}crenox install${COLOR_RESET}\n"
            echo -e "  Or if creating a new repository:"
            echo -e "     ${COLOR_CYAN}git init && crenox install${COLOR_RESET}\n"
        fi
        ;;

    none)
        info "Skipped hook configuration. You can enable it anytime later with 'crenox install'."
        ;;
esac

echo
echo -e "${COLOR_BOLD}Setup Complete!${COLOR_RESET}"
echo -e "  • Verify installation: ${COLOR_CYAN}crenox version${COLOR_RESET}"
echo -e "  • Scan any directory:  ${COLOR_CYAN}crenox scan --recursive .${COLOR_RESET}"
echo -e "  • Global protection:   ${COLOR_CYAN}crenox install --global${COLOR_RESET}"
echo
