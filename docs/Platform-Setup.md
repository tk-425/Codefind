# Platform Setup Guide

Setup instructions for codefind on different platforms.

---

## macOS ✅ Tested

### Supported Architectures

- Apple Silicon (M1/M2/M3) ✅ Verified
- Intel x86_64 (expected to work)

### Installation

```bash
# Clone and build
git clone https://github.com/tk-425/code-search.git
cd code-search
go build -o codefind ./cmd/codefind

# Install globally
sudo mv codefind /usr/local/bin/
```

### Prerequisites

- **Go 1.21+**: `brew install go`
- **LSPs (optional)**:

  ```bash
  # Go
  go install golang.org/x/tools/gopls@latest

  # Python
  npm install -g pyright

  # TypeScript
  npm install -g typescript-language-server typescript
  ```

### Network Access

- **Tailscale**: ✅ Verified working
- Use Tailscale IP for server URL (e.g., `http://100.x.y.z:8080`)

---

## Linux 🔄 Untested

### Expected to Work

Based on Go's cross-platform compatibility, codefind should work on:

- Ubuntu 20.04+
- Debian 11+
- Fedora 36+
- Other systemd-based distributions

### Installation

```bash
# Clone and build
git clone https://github.com/tk-425/code-search.git
cd code-search
go build -o codefind ./cmd/codefind

# Install globally
sudo mv codefind /usr/local/bin/
```

### Prerequisites

- **Go 1.21+**:

  ```bash
  # Ubuntu/Debian
  sudo apt install golang-go

  # Fedora
  sudo dnf install golang
  ```

- **LSPs**: Same as macOS

### Network Access

- **Tailscale**: Expected to work (install via `curl -fsSL https://tailscale.com/install.sh | sh`)

---

## Cross-Compilation

Build for different platforms from any machine:

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o codefind-linux-amd64 ./cmd/codefind

# Linux ARM64
GOOS=linux GOARCH=arm64 go build -o codefind-linux-arm64 ./cmd/codefind

# macOS Intel
GOOS=darwin GOARCH=amd64 go build -o codefind-darwin-amd64 ./cmd/codefind

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 go build -o codefind-darwin-arm64 ./cmd/codefind
```

---

## Verification

After installation, verify setup:

```bash
# Check binary works
codefind --help

# Check server connectivity
codefind health

# Check auth
codefind auth status
```

---

## Known Platform Differences

| Feature     | macOS             | Linux                |
| ----------- | ----------------- | -------------------- |
| Keychain    | ✅ macOS Keychain | File-based (planned) |
| LSP gopls   | ✅                | ✅ Expected          |
| LSP pyright | ✅                | ✅ Expected          |
| Tailscale   | ✅ Verified       | Expected             |
| Editor open | ✅ `open` command | `xdg-open` (planned) |
