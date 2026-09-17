#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# Crenox Secure & Automated Installer
# High-Performance Secret Detection & Pre-Commit Protection Engine
# ─────────────────────────────────────────────────────────────────────────────
# Usage:
#   curl -fsSL https://crenoxhq.github.io/crenox/install.sh | bash
#
# Non-Interactive / Automation Flags:
#   curl ... | bash -s -- --global          # Install binary & set global Git hook
#   curl ... | bash -s -- --local           # Install binary & protect current repo
#   curl ... | bash -s -- --no-hook         # Install binary only
#   curl ... | bash -s -- --version=v2.1.9  # Pin to a specific version
#   curl ... | bash -s -- --dir=/custom/bin # Custom install directory
#   curl ... | bash -s -- --skip-verify     # Bypass SHA-256 verification
# ─────────────────────────────────────────────────────────────────────────────

{
# Enforce strict error handling
set -euo pipefail

REPO="crenoxhq/crenox"
INSTALLER_VERSION="2.2.0"

# Configuration variables
MODE_FLAG=""
CUSTOM_VERSION="${CRENOX_VERSION:-}"
CUSTOM_DIR="${CRENOX_DIR:-}"
SKIP_VERIFY=false
QUIET=false
USE_SUDO=false
HOOK_STATUS="Not configured"

# Setup terminal styling and color palette
setup_colors() {
    if [ -t 1 ] && [ -z "${NO_COLOR:-}" ] && [ "${TERM:-}" != "dumb" ]; then
        COLOR_RESET="\033[0m"
        COLOR_BOLD="\033[1m"
        COLOR_DIM="\033[2m"
        COLOR_CYAN="\033[1;36m"
        COLOR_GREEN="\033[1;32m"
        COLOR_RED="\033[1;31m"
        COLOR_YELLOW="\033[1;33m"
        COLOR_BLUE="\033[1;34m"
        COLOR_MAGENTA="\033[1;35m"
        COLOR_GRAY="\033[0;90m"
    else
        COLOR_RESET=""
        COLOR_BOLD=""
        COLOR_DIM=""
        COLOR_CYAN=""
        COLOR_GREEN=""
        COLOR_RED=""
        COLOR_YELLOW=""
        COLOR_BLUE=""
        COLOR_MAGENTA=""
        COLOR_GRAY=""
    fi

    if [[ "${LANG:-}" =~ [Uu][Tt][Ff]-?8 ]] || [[ "${LC_ALL:-}" =~ [Uu][Tt][Ff]-?8 ]] || [[ "${LC_CTYPE:-}" =~ [Uu][Tt][Ff]-?8 ]]; then
        ICON_CHECK="✔"
        ICON_INFO="ℹ"
        ICON_WARN="⚠"
        ICON_ERROR="✖"
        ICON_ARROW="➜"
    else
        ICON_CHECK="[OK]"
        ICON_INFO="[i]"
        ICON_WARN="[!]"
        ICON_ERROR="[X]"
        ICON_ARROW=">"
    fi
}

info() {
    if [ "${QUIET}" = false ]; then
        echo -e "${COLOR_CYAN}${ICON_INFO}${COLOR_RESET} $*"
    fi
}

success() {
    if [ "${QUIET}" = false ]; then
        echo -e "${COLOR_GREEN}${ICON_CHECK}${COLOR_RESET} $*"
    fi
}

warn() {
    echo -e "${COLOR_YELLOW}${ICON_WARN}${COLOR_RESET} $*" >&2
}

error() {
    echo -e "${COLOR_RED}${ICON_ERROR} Error:${COLOR_RESET} $*" >&2
    exit 1
}

step() {
    if [ "${QUIET}" = false ]; then
        echo -e "\n${COLOR_BLUE}${COLOR_BOLD}[$1/$2]${COLOR_RESET} ${COLOR_BOLD}$3${COLOR_RESET}"
    fi
}

print_banner() {
    if [ "${QUIET}" = true ]; then
        return
    fi
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
    echo -e "${COLOR_BOLD}Crenox — Statically Compiled Git Secret Scanner & Hook Engine${COLOR_RESET}\n"
}

print_help() {
    cat << EOF
Crenox Installer

Usage:
  curl -fsSL https://crenoxhq.github.io/crenox/install.sh | bash
  curl -fsSL https://crenoxhq.github.io/crenox/install.sh | bash -s -- [OPTIONS]

Options:
  --global             Install binary & configure global Git hook for all repos
  --local              Install binary & configure hook for current repository
  --no-hook            Install binary only (skip Git hook configuration)
  --version=<tag>      Install specific version (e.g. --version=v2.1.7)
  --dir=<path>         Install executable into custom directory
  --skip-verify        Skip cryptographic SHA-256 checksum verification
  -q, --quiet          Minimal output (suppress banner and non-essential logs)
  -h, --help           Show this help message

Environment Variables:
  CRENOX_VERSION       Version to install (overridden by --version)
  CRENOX_DIR           Installation directory (overridden by --dir)
  NO_COLOR             Disable colored output when set

EOF
}

# Parse command line flags
parse_args() {
    for arg in "$@"; do
        case "$arg" in
            --global)
                MODE_FLAG="global"
                ;;
            --local)
                MODE_FLAG="local"
                ;;
            --no-hook)
                MODE_FLAG="none"
                ;;
            --skip-verify)
                SKIP_VERIFY=true
                ;;
            -q|--quiet)
                QUIET=true
                ;;
            --version=*)
                CUSTOM_VERSION="${arg#*=}"
                ;;
            --dir=*)
                CUSTOM_DIR="${arg#*=}"
                ;;
            -h|--help)
                print_help
                exit 0
                ;;
            *)
                warn "Unknown flag ignored: $arg"
                ;;
        esac
    done
}

# Check for required HTTP client
detect_http_tool() {
    if command -v curl >/dev/null 2>&1; then
        HTTP_CLIENT="curl"
    elif command -v wget >/dev/null 2>&1; then
        HTTP_CLIENT="wget"
    else
        error "Neither 'curl' nor 'wget' was found. Please install curl or wget to continue."
    fi
}

http_download() {
    local url="$1"
    local dest="$2"

    if [ "${HTTP_CLIENT}" = "curl" ]; then
        curl --proto '=https' --tlsv1.2 -sSfL \
            --retry 3 --retry-delay 1 --connect-timeout 10 \
            "${url}" -o "${dest}"
        return $?
    elif [ "${HTTP_CLIENT}" = "wget" ]; then
        wget --https-only --secure-protocol=TLSv1_2 -q \
            --tries=3 --timeout=10 \
            -O "${dest}" "${url}"
        return $?
    fi
    return 1
}

download_to_stdout() {
    local url="$1"
    if [ "${HTTP_CLIENT}" = "curl" ]; then
        curl --proto '=https' --tlsv1.2 -sSfL --connect-timeout 8 "${url}" 2>/dev/null || true
    elif [ "${HTTP_CLIENT}" = "wget" ]; then
        wget --https-only --secure-protocol=TLSv1_2 -qO- --timeout=8 "${url}" 2>/dev/null || true
    fi
}

# Detect Platform OS & Architecture
detect_platform() {
    OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
    case "${OS}" in
        linux*)   OS="linux" ;;
        darwin*)  OS="darwin" ;;
        msys*|mingw*|cygwin*) OS="windows" ;;
        freebsd*) OS="freebsd" ;;
        *) error "Unsupported operating system: ${OS}" ;;
    esac

    # Termux / Android environment detection
    if [ -d "/data/data/com.termux" ] || [[ "${PREFIX:-}" == *"com.termux"* ]] || [ -n "${ANDROID_ROOT:-}" ]; then
        OS="android"
    fi

    ARCH="$(uname -m)"
    case "${ARCH}" in
        x86_64|amd64)           ARCH="amd64" ;;
        aarch64|arm64|armv8*)   ARCH="arm64" ;;
        armv7*|armv6*|armhf|arm) ARCH="arm" ;;
        *) error "Unsupported CPU architecture: ${ARCH}. Crenox prebuilt binaries support amd64, arm64, and arm." ;;
    esac

    info "Detected platform: ${COLOR_BOLD}${OS}/${ARCH}${COLOR_RESET}"
}

# Resolve release version
resolve_version() {
    if [ -n "${CUSTOM_VERSION}" ]; then
        if [[ "${CUSTOM_VERSION}" != v* ]]; then
            TARGET_TAG="v${CUSTOM_VERSION}"
        else
            TARGET_TAG="${CUSTOM_VERSION}"
        fi
        # Validate format
        if [[ ! "${TARGET_TAG}" =~ ^v[0-9]+\.[0-9]+(\.[0-9]+)?(-[a-zA-Z0-9.]+)?$ ]]; then
            error "Invalid version format: '${CUSTOM_VERSION}'. Expected format: vX.Y.Z (e.g. v2.1.7)"
        fi
        info "Target version specified: ${COLOR_BOLD}${TARGET_TAG}${COLOR_RESET}"
        return
    fi

    info "Resolving latest stable release..."
    TARGET_TAG=$(download_to_stdout "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep -m1 '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' || true)

    # Fallback to redirect location header if API is rate-limited
    if [ -z "${TARGET_TAG}" ]; then
        if [ "${HTTP_CLIENT}" = "curl" ]; then
            TARGET_TAG=$(curl --proto '=https' --tlsv1.2 -sI --connect-timeout 8 "https://github.com/${REPO}/releases/latest" 2>/dev/null \
                | awk -F'/' '/[Ll]ocation:/ {print $NF}' | tr -d '\r\n' || true)
        fi
    fi

    if [ -z "${TARGET_TAG}" ]; then
        error "Could not resolve latest release version from GitHub.\nPlease check your connection or specify explicitly: curl ... | bash -s -- --version=v2.1.7"
    fi

    info "Resolved latest version: ${COLOR_BOLD}${TARGET_TAG}${COLOR_RESET}"
}

# Compute SHA-256 hash
compute_sha256() {
    local target_file="$1"
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "${target_file}" | awk '{print $1}'
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "${target_file}" | awk '{print $1}'
    elif command -v openssl >/dev/null 2>&1; then
        openssl dgst -sha256 "${target_file}" | awk '{print $NF}'
    else
        return 1
    fi
}

# Verify cryptographic checksum
verify_checksum() {
    local bin_path="$1"
    local release_filename="$2"

    if [ "${SKIP_VERIFY}" = true ]; then
        warn "Skipping checksum verification as requested (--skip-verify)."
        return 0
    fi

    local checksums_url="https://github.com/${REPO}/releases/download/${TARGET_TAG}/checksums.txt"
    local single_sha_url="https://github.com/${REPO}/releases/download/${TARGET_TAG}/${release_filename}.sha256"
    local checksums_file="${TEMP_DIR}/checksums.txt"
    local single_sha_file="${TEMP_DIR}/${release_filename}.sha256"
    local expected_hash=""

    if http_download "${checksums_url}" "${checksums_file}" 2>/dev/null; then
        expected_hash=$(grep -E "[[:space:]]${release_filename}\$" "${checksums_file}" 2>/dev/null | awk '{print $1}' | tr '[:upper:]' '[:lower:]' || true)
    elif http_download "${single_sha_url}" "${single_sha_file}" 2>/dev/null; then
        expected_hash=$(awk '{print $1}' "${single_sha_file}" 2>/dev/null | tr '[:upper:]' '[:lower:]' || true)
    fi

    if [ -n "${expected_hash}" ]; then
        info "Verifying SHA-256 cryptographic integrity..."
        local actual_hash=""
        if ! actual_hash=$(compute_sha256 "${bin_path}"); then
            warn "No SHA-256 tool available to compute hash. Skipping checksum verification."
            return 0
        fi
        actual_hash=$(echo "${actual_hash}" | tr '[:upper:]' '[:lower:]')

        if [ "${actual_hash}" != "${expected_hash}" ]; then
            error "SHA-256 checksum mismatch!\n  Expected: ${expected_hash}\n  Actual:   ${actual_hash}\nThe download may be incomplete or corrupted."
        fi
        success "SHA-256 checksum verified: ${COLOR_DIM}${actual_hash:0:16}...${COLOR_RESET}"
    else
        info "Checksum file not available for this release; relying on binary self-test."
    fi
}

# Select target directory for installation
resolve_install_dir() {
    if [ -n "${CUSTOM_DIR}" ]; then
        INSTALL_DIR="${CUSTOM_DIR}"
        return
    fi

    if [ "${OS}" = "android" ] && [ -n "${PREFIX:-}" ]; then
        INSTALL_DIR="${PREFIX}/bin"
        return
    fi

    if [ -w "/usr/local/bin" ]; then
        INSTALL_DIR="/usr/local/bin"
        return
    fi

    if command -v sudo >/dev/null 2>&1; then
        INSTALL_DIR="/usr/local/bin"
        USE_SUDO=true
        return
    fi

    # Fallback for environments without root/sudo (e.g. rootless containers)
    INSTALL_DIR="${HOME}/.local/bin"
}

# Install executable safely
install_binary() {
    local src="$1"
    local dst_dir="$2"
    local bin_name="$3"
    local dst_path="${dst_dir}/${bin_name}"

    mkdir -p "${dst_dir}" 2>/dev/null || true

    if [ -w "${dst_dir}" ]; then
        mv -f "${src}" "${dst_path}"
        chmod 755 "${dst_path}"
    elif [ "${USE_SUDO}" = true ] && command -v sudo >/dev/null 2>&1; then
        info "Elevated permissions required to write to ${dst_dir}..."
        if sudo mv -f "${src}" "${dst_path}" && sudo chmod 755 "${dst_path}"; then
            :
        else
            warn "Sudo installation failed. Falling back to ${HOME}/.local/bin..."
            mkdir -p "${HOME}/.local/bin"
            mv -f "${src}" "${HOME}/.local/bin/${bin_name}"
            chmod 755 "${HOME}/.local/bin/${bin_name}"
            INSTALL_DIR="${HOME}/.local/bin"
            dst_path="${INSTALL_DIR}/${bin_name}"
        fi
    else
        local fallback_dir="${HOME}/.local/bin"
        info "${dst_dir} is not writable. Installing to ${fallback_dir}..."
        mkdir -p "${fallback_dir}"
        mv -f "${src}" "${fallback_dir}/${bin_name}"
        chmod 755 "${fallback_dir}/${bin_name}"
        INSTALL_DIR="${fallback_dir}"
        dst_path="${INSTALL_DIR}/${bin_name}"
    fi

    INSTALLED_PATH="${dst_path}"
}

# Interactive / Non-Interactive Git Hook Setup
configure_git_protection() {
    local choice="${MODE_FLAG}"

    # If piped into bash (e.g. curl | bash), stdin is the pipe.
    # Check if /dev/tty is available for interactive prompts.
    if [ -z "${choice}" ]; then
        if [ -r /dev/tty ] && [ -w /dev/tty ]; then
            echo -e "\n${COLOR_BOLD}Choose Crenox Git protection mode:${COLOR_RESET}" >/dev/tty
            echo -e "  ${COLOR_CYAN}1)${COLOR_RESET} Protect current repository (${COLOR_BOLD}crenox install${COLOR_RESET})" >/dev/tty
            echo -e "  ${COLOR_CYAN}2)${COLOR_RESET} Protect ALL Git repositories globally (${COLOR_BOLD}crenox install --global${COLOR_RESET})" >/dev/tty
            echo -e "  ${COLOR_CYAN}3)${COLOR_RESET} Skip hook setup for now (Binary only)" >/dev/tty
            local user_input=""
            read -rp "Enter choice [1-3] (default: 1): " user_input </dev/tty || true
            case "${user_input}" in
                2) choice="global" ;;
                3) choice="none" ;;
                *) choice="local" ;;
            esac
        else
            choice="local"
        fi
    fi

    case "${choice}" in
        global)
            info "Enabling Crenox Git pre-commit hook globally..."
            if "${INSTALLED_PATH}" install --global; then
                success "Global Git hook active. Every repository on this machine is protected!"
                HOOK_STATUS="Active (Global)"
            else
                warn "Failed to configure global Git hook automatically. You can run 'crenox install --global' manually."
                HOOK_STATUS="Manual setup needed"
            fi
            ;;
        local)
            if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
                info "Git repository detected in current directory."
                if "${INSTALLED_PATH}" install; then
                    success "Pre-commit hook installed in current repository!"
                    HOOK_STATUS="Active (Local Repo)"
                else
                    warn "Could not install hook. You can run 'crenox install' manually."
                    HOOK_STATUS="Manual setup needed"
                fi
            else
                info "Current directory is not a Git repository."
                HOOK_STATUS="Ready (Run 'crenox install' in your repo)"
            fi
            ;;
        none)
            info "Hook setup skipped."
            HOOK_STATUS="Skipped (Binary only)"
            ;;
    esac
}

check_path_env() {
    if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
        echo
        warn "${INSTALL_DIR} is not in your current \$PATH."
        echo -e "   ${COLOR_BOLD}To use 'crenox' anywhere, add this directory to your shell configuration:${COLOR_RESET}"
        local user_shell
        user_shell=$(basename "${SHELL:-bash}")
        case "${user_shell}" in
            zsh)
                echo -e "     ${COLOR_CYAN}echo 'export PATH=\"${INSTALL_DIR}:\$PATH\"' >> ~/.zshrc && source ~/.zshrc${COLOR_RESET}\n"
                ;;
            fish)
                echo -e "     ${COLOR_CYAN}fish_add_path ${INSTALL_DIR}${COLOR_RESET}\n"
                ;;
            *)
                echo -e "     ${COLOR_CYAN}echo 'export PATH=\"${INSTALL_DIR}:\$PATH\"' >> ~/.bashrc && source ~/.bashrc${COLOR_RESET}\n"
                ;;
        esac
    fi
}

print_summary() {
    if [ "${QUIET}" = true ]; then
        return
    fi
    echo
    echo -e "${COLOR_GREEN}${COLOR_BOLD}${ICON_CHECK} Crenox ${TARGET_TAG} installed successfully!${COLOR_RESET}"
    echo
    echo -e "  ${COLOR_BOLD}Details:${COLOR_RESET}"
    echo -e "    • Binary:       ${COLOR_CYAN}${INSTALLED_PATH}${COLOR_RESET}"
    echo -e "    • Platform:     ${COLOR_CYAN}${OS}/${ARCH}${COLOR_RESET}"
    echo -e "    • Git Hook:     ${COLOR_GREEN}${HOOK_STATUS}${COLOR_RESET}"
    echo
    echo -e "  ${COLOR_BOLD}Quickstart:${COLOR_RESET}"
    echo -e "    ${COLOR_CYAN}crenox scan --recursive .${COLOR_RESET}   ${COLOR_GRAY}# Scan current directory for secrets${COLOR_RESET}"
    echo -e "    ${COLOR_CYAN}crenox run${COLOR_RESET}                   ${COLOR_GRAY}# Check staged files before commit${COLOR_RESET}"
    echo -e "    ${COLOR_CYAN}crenox install --global${COLOR_RESET}   ${COLOR_GRAY}# Protect all Git repos on this machine${COLOR_RESET}"
    echo -e "    ${COLOR_CYAN}crenox update${COLOR_RESET}                ${COLOR_GRAY}# Check and upgrade to latest release${COLOR_RESET}"
    echo
}

# Main Execution Flow
main() {
    setup_colors
    parse_args "$@"
    print_banner

    # Temporary directory with restricted permissions (0700)
    TEMP_DIR=$(mktemp -d 2>/dev/null || mktemp -d -t 'crenox-install')
    chmod 700 "${TEMP_DIR}"
    trap 'rm -rf "${TEMP_DIR}"' EXIT INT TERM

    # Step 1: Pre-flight checks
    step 1 4 "System and Environment Discovery"
    detect_http_tool
    detect_platform

    if ! command -v git >/dev/null 2>&1; then
        warn "Git was not found on your system."
        warn "Crenox requires Git for pre-commit protection. Please install Git via your package manager."
    fi

    # Step 2: Version and Artifact Resolution
    step 2 4 "Release and Cryptographic Verification"
    resolve_version

    if [ "${OS}" = "windows" ]; then
        RELEASE_FILE="crenox-${TARGET_TAG}-${OS}-${ARCH}.exe"
        TARGET_BIN="crenox.exe"
    else
        RELEASE_FILE="crenox-${TARGET_TAG}-${OS}-${ARCH}"
        TARGET_BIN="crenox"
    fi

    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TARGET_TAG}/${RELEASE_FILE}"
    TEMP_EXEC="${TEMP_DIR}/${TARGET_BIN}"

    info "Downloading ${COLOR_BOLD}${RELEASE_FILE}${COLOR_RESET}..."
    if ! http_download "${DOWNLOAD_URL}" "${TEMP_EXEC}"; then
        error "Failed to download binary from:\n  ${DOWNLOAD_URL}\nPlease verify your internet connection or check the release tag."
    fi

    # Integrity verification
    verify_checksum "${TEMP_EXEC}" "${RELEASE_FILE}"

    chmod +x "${TEMP_EXEC}"

    # Self-test pre-flight execution
    if ! "${TEMP_EXEC}" version >/dev/null 2>&1; then
        error "Self-test failed: the downloaded binary could not be executed."
    fi

    # Step 3: Installation to PATH
    step 3 4 "Binary Placement"
    resolve_install_dir
    install_binary "${TEMP_EXEC}" "${INSTALL_DIR}" "${TARGET_BIN}"
    success "Executable installed to ${COLOR_BOLD}${INSTALLED_PATH}${COLOR_RESET}"

    # Step 4: Security Hook Configuration
    step 4 4 "Git Protection Setup"
    configure_git_protection

    # Check PATH and print finish
    check_path_env
    print_summary
}

main "$@"
}
