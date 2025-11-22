#!/bin/bash
set -e

# Hippo installer script
# Usage: curl -sSL https://raw.githubusercontent.com/oribarilan/hippo/main/install.sh | bash

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
REPO="oribarilan/hippo"
BINARY_NAME="hippo"

# Detect OS and architecture
detect_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)
    
    case "$os" in
        linux)
            OS="linux"
            # Linux: prefer ~/.local/bin (user), fallback to /usr/local/bin if root
            if [ -z "$INSTALL_DIR" ]; then
                if [ "$EUID" -eq 0 ]; then
                    INSTALL_DIR="/usr/local/bin"
                else
                    INSTALL_DIR="$HOME/.local/bin"
                fi
            fi
            ;;
        darwin)
            OS="darwin"
            # macOS: prefer /usr/local/bin (Homebrew standard)
            if [ -z "$INSTALL_DIR" ]; then
                INSTALL_DIR="/usr/local/bin"
            fi
            ;;
        mingw*|msys*|cygwin*)
            OS="windows"
            # Windows: use %USERPROFILE%\bin
            if [ -z "$INSTALL_DIR" ]; then
                INSTALL_DIR="$HOME/bin"
            fi
            ;;
        *)
            echo -e "${RED}Unsupported operating system: $os${NC}"
            exit 1
            ;;
    esac
    
    case "$arch" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        arm64|aarch64)
            ARCH="arm64"
            ;;
        *)
            echo -e "${RED}Unsupported architecture: $arch${NC}"
            exit 1
            ;;
    esac
    
    echo -e "${GREEN}Detected platform: ${OS}_${ARCH}${NC}"
    echo -e "${GREEN}Install directory: ${INSTALL_DIR}${NC}"
}

# Get latest release version
get_latest_version() {
    echo -e "${YELLOW}Fetching latest release...${NC}"
    VERSION=$(curl -sL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    
    if [ -z "$VERSION" ]; then
        echo -e "${RED}Failed to fetch latest version${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}Latest version: ${VERSION}${NC}"
}

# Download and install
install_binary() {
    # Remove 'v' prefix from version for archive name (v0.1.0 -> 0.1.0)
    local version_without_v="${VERSION#v}"
    local archive_name="${BINARY_NAME}_${version_without_v}_${OS}_${ARCH}"
    
    if [ "$OS" = "windows" ]; then
        archive_name="${archive_name}.zip"
    else
        archive_name="${archive_name}.tar.gz"
    fi
    
    local download_url="https://github.com/${REPO}/releases/download/${VERSION}/${archive_name}"
    local tmp_dir=$(mktemp -d)
    
    echo -e "${YELLOW}Downloading ${archive_name}...${NC}"
    
    if ! curl -sSL -o "${tmp_dir}/${archive_name}" "$download_url"; then
        echo -e "${RED}Download failed${NC}"
        rm -rf "$tmp_dir"
        exit 1
    fi
    
    echo -e "${YELLOW}Extracting...${NC}"
    cd "$tmp_dir"
    
    if [ "$OS" = "windows" ]; then
        unzip -q "$archive_name"
    else
        tar -xzf "$archive_name"
    fi
    
    # Create install directory if it doesn't exist
    if ! mkdir -p "$INSTALL_DIR" 2>/dev/null; then
        echo -e "${RED}Failed to create directory: ${INSTALL_DIR}${NC}"
        echo -e "${YELLOW}Try running with sudo or set INSTALL_DIR to a writable location:${NC}"
        echo -e "  curl -sSL https://raw.githubusercontent.com/${REPO}/main/install.sh | INSTALL_DIR=\$HOME/.local/bin bash"
        rm -rf "$tmp_dir"
        exit 1
    fi
    
    echo -e "${YELLOW}Installing to ${INSTALL_DIR}...${NC}"
    
    if [ -f "${INSTALL_DIR}/${BINARY_NAME}" ]; then
        echo -e "${YELLOW}Existing installation found, replacing...${NC}"
        if ! rm -f "${INSTALL_DIR}/${BINARY_NAME}" 2>/dev/null; then
            echo -e "${RED}Failed to remove existing installation. You may need sudo.${NC}"
            rm -rf "$tmp_dir"
            exit 1
        fi
    fi
    
    if ! mv "$BINARY_NAME" "$INSTALL_DIR/" 2>/dev/null; then
        echo -e "${RED}Failed to install binary to ${INSTALL_DIR}${NC}"
        echo -e "${YELLOW}Try running with sudo or set INSTALL_DIR to a writable location:${NC}"
        echo -e "  curl -sSL https://raw.githubusercontent.com/${REPO}/main/install.sh | INSTALL_DIR=\$HOME/.local/bin bash"
        rm -rf "$tmp_dir"
        exit 1
    fi
    
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    
    # Cleanup
    rm -rf "$tmp_dir"
    
    echo -e "${GREEN}✓ Hippo ${VERSION} installed successfully!${NC}"
}

# Check if install dir is in PATH
check_path() {
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        echo -e "${YELLOW}⚠ Warning: ${INSTALL_DIR} is not in your PATH${NC}"
        echo -e "Add this line to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
        echo -e "  export PATH=\"\$PATH:${INSTALL_DIR}\""
    fi
}

# Main
main() {
    echo -e "${GREEN}=== Hippo Installer ===${NC}\n"
    
    detect_platform
    get_latest_version
    install_binary
    check_path
    
    echo -e "\n${GREEN}Run 'hippo --init' to configure, then 'hippo' to start!${NC}"
}

main
