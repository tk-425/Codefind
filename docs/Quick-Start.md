# Quick Start Guide

Get started with codefind in 5 minutes.

## Prerequisites

- **Go 1.21+** installed
- **Server access** (Tailscale or local network to codefind-server)

## Installation

```bash
# Clone and build
git clone https://github.com/tk-425/code-search.git
cd code-search
go build -o codefind ./cmd/codefind

# Move to PATH (optional)
sudo mv codefind /usr/local/bin/
```

## Initial Setup

### 1. Initialize codefind

```bash
codefind init
```

Enter your server URL when prompted (e.g., `http://100.x.y.z:8080`).

### 2. Authenticate

```bash
codefind auth login
```

Enter your auth key provided by the server admin.

### 3. Verify Connection

```bash
codefind health
```

Expected output:

```
✅ Server: OK
✅ Ollama: OK
✅ ChromaDB: OK
```

## Basic Usage

### Index Your Code

```bash
cd /path/to/your/project
codefind index
```

### Check Index Stats

```bash
codefind stats
```

### Search Your Code

```bash
codefind query "error handling"
codefind query "authentication" --limit=5
codefind query "api endpoints" --lang=go
```

### List Indexed Projects

```bash
codefind list
```

### Remove Indexed Project

```bash
codefind clear  # Remove current project
```

## Common Workflows

### Find a Function

```bash
codefind query "validate user input"
```

### Search by Language

```bash
codefind query "database connection" --lang=python
```

### Update Index After Changes

```bash
codefind index  # Incremental update
```

## Next Steps

- [CLI Commands Reference](./Commands.md)
- [Troubleshooting Guide](./Troubleshooting.md)
- [Configuration Options](./Configuration.md)
