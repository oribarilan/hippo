# Continuous Deployment Implementation Plan

**Status:** Ready for Implementation  
**Created:** 2025-11-20  
**Goal:** Set up automated releases with GitHub Actions + GoReleaser, supporting curl-based installation and future in-app updates

---

## Overview

Implement industry-standard continuous deployment for Hippo using git tags, GoReleaser, and GitHub Actions. Users create a version tag, and the system automatically builds multi-platform binaries and creates a GitHub Release.

## Key Design Decisions

1. **Version management:** Git tags (e.g., `v0.3.0`) trigger releases
2. **Build tool:** GoReleaser (industry standard for Go projects)
3. **Installation method:** curl/wget script for initial setup
4. **Update mechanism:** In-app self-update (future enhancement)
5. **Platforms:** Linux, macOS, Windows × amd64, arm64
6. **Version embedding:** Use `-ldflags` to inject version at build time

## Release Workflow

```bash
# Developer workflow
git tag v0.3.0
git push origin v0.3.0

# GitHub Actions automatically:
# 1. Runs tests on all platforms
# 2. Builds binaries (Linux, macOS, Windows × amd64, arm64)
# 3. Generates checksums (SHA256)
# 4. Creates GitHub Release with all artifacts
# 5. Auto-generates release notes from commits
```

## Implementation Plan

### Phase 1: Version Management

#### 1. Fix Version Inconsistencies

**Current state:**
- `app/constants.go:4` - `const version = "v0.1.0"`
- `app/main.go:23` - Hardcoded `"Hippo v0.2.0"`

**Changes needed:**

**File: `app/constants.go`**
```go
package main

// Version is set via ldflags during build: -X main.Version=v0.3.0
var Version = "dev"

// defaultLoadLimit is the number of items to load per request
const defaultLoadLimit = 40
```

**File: `app/main.go`** (Line 23)
```go
// OLD:
fmt.Println("Hippo v0.2.0")

// NEW:
fmt.Printf("Hippo %s\n", Version)
```

**Rationale:**
- Single source of truth for version
- `var` instead of `const` allows build-time injection
- Defaults to "dev" for local builds
- Build command: `go build -ldflags="-X main.Version=v0.3.0"`

### Phase 2: GoReleaser Configuration

#### 2. Create `.goreleaser.yml`

**File: `.goreleaser.yml`** (root directory)
```yaml
# GoReleaser configuration for Hippo
# Docs: https://goreleaser.com

version: 2

before:
  hooks:
    # Format and vet code
    - go fmt ./app/...
    - go vet ./app/...
    # Run tests
    - go test -v ./app/...

builds:
  - id: hippo
    # Source directory
    dir: ./app
    # Binary name
    binary: hippo
    # Version injection
    ldflags:
      - -s -w
      - -X main.Version={{.Version}}
    # Build targets
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    # Exclude problematic combinations
    ignore:
      - goos: windows
        goarch: arm64

archives:
  - id: hippo-archive
    # Archive name format: hippo_v0.3.0_linux_amd64.tar.gz
    name_template: >-
      {{ .ProjectName }}_
      {{- .Version }}_
      {{- .Os }}_
      {{- .Arch }}
    format_overrides:
      - goos: windows
        format: zip
    files:
      - README.md
      - LICENSE.md

checksum:
  name_template: 'checksums.txt'
  algorithm: sha256

changelog:
  sort: asc
  use: github
  filters:
    exclude:
      - '^docs:'
      - '^test:'
      - '^ci:'
      - '^chore:'
      - Merge pull request
      - Merge branch
  groups:
    - title: Features
      regexp: '^.*?feat(\([[:word:]]+\))??!?:.+$'
      order: 0
    - title: Bug Fixes
      regexp: '^.*?fix(\([[:word:]]+\))??!?:.+$'
      order: 1
    - title: Enhancements
      regexp: '^.*?enhance(\([[:word:]]+\))??!?:.+$'
      order: 2
    - title: Other Changes
      order: 999

release:
  github:
    owner: orbarila
    name: hippo
  draft: false
  prerelease: auto
  mode: replace
  header: |
    ## Hippo {{.Version}}
    
    Install or upgrade:
    ```bash
    curl -sSL https://raw.githubusercontent.com/orbarila/hippo/main/install.sh | bash
    ```
  footer: |
    ## Checksums
    
    All binaries are checksummed for security. Verify downloads with:
    ```bash
    sha256sum -c checksums.txt
    ```
```

**Key Features:**
- Runs tests before building
- Builds for 5 platforms (Linux, macOS, Windows × amd64, arm64)
- Injects version at build time
- Creates archives (`.tar.gz` for Unix, `.zip` for Windows)
- Generates SHA256 checksums
- Auto-generates changelog from commit messages
- Includes installation instructions in release notes

### Phase 3: GitHub Actions Workflow

#### 3. Create `.github/workflows/release.yml`

**File: `.github/workflows/release.yml`**
```yaml
name: Release

on:
  push:
    tags:
      - 'v*.*.*'

permissions:
  contents: write

jobs:
  test:
    name: Test Before Release
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'
          cache: true
          cache-dependency-path: app/go.sum

      - name: Run tests
        working-directory: ./app
        run: go test -v -race ./...

  goreleaser:
    name: Build and Release
    needs: test
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Full history for changelog

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'
          cache: true
          cache-dependency-path: app/go.sum

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v5
        with:
          distribution: goreleaser
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

**Workflow:**
1. Triggered by pushing tags matching `v*.*.*` (e.g., `v0.3.0`)
2. First runs tests to ensure code quality
3. Only proceeds to release if tests pass
4. Uses GoReleaser to build and publish
5. Creates GitHub Release automatically

### Phase 4: Installation Script

#### 4. Create `install.sh`

**File: `install.sh`** (root directory)
```bash
#!/bin/bash
set -e

# Hippo installer script
# Usage: curl -sSL https://raw.githubusercontent.com/orbarila/hippo/main/install.sh | bash

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
REPO="orbarila/hippo"
BINARY_NAME="hippo"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

# Detect OS and architecture
detect_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)
    
    case "$os" in
        linux)
            OS="linux"
            ;;
        darwin)
            OS="darwin"
            ;;
        mingw*|msys*|cygwin*)
            OS="windows"
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
    local archive_name="${BINARY_NAME}_${VERSION}_${OS}_${ARCH}"
    
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
    mkdir -p "$INSTALL_DIR"
    
    echo -e "${YELLOW}Installing to ${INSTALL_DIR}...${NC}"
    
    if [ -f "${INSTALL_DIR}/${BINARY_NAME}" ]; then
        echo -e "${YELLOW}Existing installation found, replacing...${NC}"
        rm -f "${INSTALL_DIR}/${BINARY_NAME}"
    fi
    
    mv "$BINARY_NAME" "$INSTALL_DIR/"
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
```

**Features:**
- Auto-detects OS and architecture
- Downloads latest release from GitHub
- Installs to `~/.local/bin` (configurable via `INSTALL_DIR`)
- Checks if install directory is in PATH
- Works on Linux, macOS, and Windows (with bash)
- Colorful output for better UX

**Usage:**
```bash
# Standard install
curl -sSL https://raw.githubusercontent.com/orbarila/hippo/main/install.sh | bash

# Custom install location
curl -sSL https://raw.githubusercontent.com/orbarila/hippo/main/install.sh | INSTALL_DIR=/usr/local/bin bash
```

### Phase 5: Documentation Updates

#### 5. Update `README.md`

**Add Installation section:**
```markdown
## Installation

### Quick Install (Linux/macOS)

```bash
curl -sSL https://raw.githubusercontent.com/oribarilan/hippo/main/install.sh | bash
```

The installer will:
- Download the latest release for your platform
- Install to `~/.local/bin/hippo`
- Make the binary executable

After installation:
```bash
hippo --init  # Configure
hippo         # Start
```

### Manual Installation

Download the latest release for your platform:
https://github.com/oribarilan/hippo/releases/latest

Extract and move to a directory in your PATH:
```bash
# Linux/macOS
tar -xzf hippo_*_linux_amd64.tar.gz
sudo mv hippo /usr/local/bin/

# Windows
# Extract hippo_*_windows_amd64.zip
# Move hippo.exe to a directory in your PATH
```

### Verify Installation

```bash
hippo --version
```
```

**Update Development section:**
```markdown
## Development

### Building from Source

```bash
cd app
go build -o hippo .
./hippo --version  # Should show "dev"
```

### Building with Version

```bash
cd app
go build -ldflags="-X main.Version=v0.3.0-custom" -o hippo .
./hippo --version  # Should show "v0.3.0-custom"
```
```

#### 6. Update `AGENTS.md`

**Add Release section:**
```markdown
## Releasing

Hippo uses GoReleaser for automated releases triggered by git tags.

**Release Process:**

1. Ensure all changes are committed and pushed to main
2. Create and push a version tag:
   ```bash
   git tag v0.3.0
   git push origin v0.3.0
   ```
3. GitHub Actions automatically:
   - Runs tests
   - Builds binaries for all platforms
   - Creates GitHub Release with artifacts
   - Generates changelog

**Version Format:** Semantic versioning (v0.3.0, v1.0.0, etc.)

**Version Embedding:**
- Version is injected at build time via `-ldflags`
- Variable: `main.Version` in `app/constants.go`
- Local builds show "dev"
- Released builds show actual version (e.g., "v0.3.0")

**Key Files:**
- `.goreleaser.yml` - Build configuration
- `.github/workflows/release.yml` - CI/CD workflow
- `install.sh` - Installation script
- `app/constants.go` - Version variable
```

### Phase 6: Testing Plan

#### Manual Testing Checklist

**Pre-Release:**
- [ ] Version variable exists in `constants.go` as `var Version = "dev"`
- [ ] `main.go` uses `Version` variable for `--version` flag
- [ ] Local build shows "dev" version: `go build && ./hippo --version`
- [ ] Build with version flag works: `go build -ldflags="-X main.Version=v0.2.9-test" && ./hippo --version`
- [ ] All existing tests pass: `go test -v ./...`
- [ ] CI workflow passes on main branch

**Release Testing:**
- [ ] Create test tag locally: `git tag v0.2.9-test`
- [ ] Push test tag: `git push origin v0.2.9-test`
- [ ] Verify GitHub Action runs
- [ ] Verify tests pass before build
- [ ] Verify binaries are created for all platforms
- [ ] Verify checksums are generated
- [ ] Verify GitHub Release is created
- [ ] Download each binary and verify it runs
- [ ] Verify `--version` shows correct version in released binaries
- [ ] Delete test tag: `git push --delete origin v0.2.9-test && git tag -d v0.2.9-test`

**Installation Script Testing:**
- [ ] Test on Linux (amd64, arm64)
- [ ] Test on macOS (amd64, arm64)
- [ ] Test fresh install (no existing binary)
- [ ] Test upgrade (existing binary present)
- [ ] Test custom install directory: `INSTALL_DIR=/tmp/test-install ./install.sh`
- [ ] Verify installed binary has correct version
- [ ] Verify binary is executable
- [ ] Verify PATH warning shows if needed

**First Real Release:**
- [ ] Update version in git tag: `v0.3.0`
- [ ] Push tag and verify release
- [ ] Test installation script with real release
- [ ] Update README.md with actual release info
- [ ] Announce release (if applicable)

## Implementation Order

1. **Update version handling** (10 min)
   - Modify `app/constants.go`
   - Modify `app/main.go`
   - Test locally

2. **Create GoReleaser config** (15 min)
   - Create `.goreleaser.yml`
   - Validate syntax: `goreleaser check`

3. **Create GitHub Actions workflow** (10 min)
   - Create `.github/workflows/release.yml`
   - Verify YAML syntax

4. **Create installation script** (30 min)
   - Create `install.sh`
   - Test locally with mock downloads
   - Make executable: `chmod +x install.sh`

5. **Update documentation** (20 min)
   - Update `README.md`
   - Update `AGENTS.md`

6. **Test release workflow** (30 min)
   - Create test tag
   - Verify full release process
   - Test installation script
   - Clean up test release

7. **Create first real release** (10 min)
   - Tag v0.3.0
   - Monitor release
   - Test installation
   - Celebrate! 🎉

**Total Time:** ~2 hours

## Success Criteria

- [ ] Pushing a git tag triggers automated release
- [ ] All platforms build successfully (5 binaries)
- [ ] GitHub Release created with all artifacts
- [ ] Checksums generated for security
- [ ] Installation script works on Linux and macOS
- [ ] Installed binary shows correct version
- [ ] README includes installation instructions
- [ ] Documentation is up to date

## Future Enhancements

### v0.4.0+ (Post-MVP)
- [ ] In-app update checking (check GitHub releases API)
- [ ] Self-update command: `hippo update`
- [ ] Homebrew formula (via `goreleaser`)
- [ ] Detect installation method (Homebrew vs direct)
- [ ] Adapt update mechanism based on installation method
- [ ] Scoop manifest for Windows
- [ ] Arch Linux AUR package

### Nice-to-Have
- [ ] Code signing for macOS/Windows binaries
- [ ] Notarization for macOS
- [ ] Windows installer (.msi)
- [ ] Debian/RPM packages

## Notes

- **Keep it simple for v1:** Just curl install + manual updates
- **GoReleaser handles complexity:** Multi-platform builds, archives, checksums
- **Git tags are source of truth:** Version comes from tag, not code
- **Installation script is foundation:** Sets up self-update later
- **Test thoroughly:** Create test releases before v0.3.0

## Questions & Decisions

### Resolved:
- ✅ Version management: Git tags trigger releases
- ✅ Build tool: GoReleaser (industry standard)
- ✅ Installation: curl script (simple, works everywhere)
- ✅ Version embedding: `-ldflags` at build time
- ✅ Update mechanism: Manual for now, in-app later
- ✅ Platforms: Linux, macOS, Windows × amd64, arm64

### Open:
- ⏳ When to add Homebrew support? (After curl install is proven)
- ⏳ When to add in-app updates? (v0.4.0 or later)
- ⏳ Should we add Windows installer? (Later, if users request)

---

**Plan Version:** 1.0  
**Created:** 2025-11-20  
**Next Review:** After first release (v0.3.0)
