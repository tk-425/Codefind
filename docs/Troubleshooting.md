# Troubleshooting Guide

Common issues and solutions for codefind.

---

## Connection Issues

### Server Unreachable

**Symptom:**

```
Error: Failed to connect to server
```

**Solutions:**

1. **Check server URL:**

   ```bash
   codefind config get server_url
   ```

2. **Verify server is running:**

   ```bash
   curl http://YOUR_SERVER_IP:8080/health
   ```

3. **Check Tailscale (if using):**

   ```bash
   tailscale status
   ```

4. **Update server URL:**
   ```bash
   codefind config set server_url http://CORRECT_IP:8080
   ```

---

### Authentication Failed

**Symptom:**

```
Error: 401 Unauthorized
```

**Solutions:**

1. **Re-authenticate:**

   ```bash
   codefind auth login
   ```

2. **Verify auth status:**

   ```bash
   codefind auth status
   ```

3. **Contact admin** for new auth key if expired.

---

## Indexing Issues

### No Files Found

**Symptom:**

```
Found 0 files to index
```

**Solutions:**

1. **Check directory:**

   ```bash
   pwd  # Verify you're in the right directory
   ```

2. **Preview discoverable files:**

   ```bash
   codefind list-files
   ```

3. **Check .gitignore:** Files matching .gitignore patterns are excluded.

---

### LSP Not Available

**Symptom:**

```
✗ gopls not found, using window chunking
```

**Solutions:**

1. **Install missing LSP:**

   ```bash
   # Go
   go install golang.org/x/tools/gopls@latest

   # Python
   npm install -g pyright

   # TypeScript
   npm install -g typescript-language-server
   ```

2. **Verify LSP in PATH:**

   ```bash
   which gopls
   ```

3. **Use window-only mode (workaround):**
   ```bash
   codefind index --window-only
   ```

---

### Tokenization Timeout

**Symptom:**

```
tokenization failed: server error: The read operation timed out
```

**Solutions:**

1. **Check server resources:**
   - Ensure Ollama has enough memory
   - Check server CPU usage

2. **Restart Ollama:**

   ```bash
   # On server
   sudo systemctl restart ollama
   ```

3. **Re-run index:** Timeouts are often transient.

---

## Query Issues

### No Results Found

**Symptom:**

```
No results found for "my query"
```

**Solutions:**

1. **Verify project is indexed:**

   ```bash
   codefind list
   ```

2. **Try broader search terms.**

3. **Re-index if code changed:**
   ```bash
   codefind index
   ```

---

### Project Not Found

**Symptom:**

```
Error: Project "MyProject" not found
```

**Solutions:**

1. **List available projects:**

   ```bash
   codefind list
   ```

2. **Use exact project name** (case-sensitive).

3. **Index the project first:**
   ```bash
   cd /path/to/project
   codefind index
   ```

---

## Server Issues

### Ollama Not Responding

**Symptom:**

```
Health check: Ollama FAILED
```

**Solutions (on server):**

1. **Check Ollama status:**

   ```bash
   sudo systemctl status ollama
   ```

2. **Restart Ollama:**

   ```bash
   sudo systemctl restart ollama
   ```

3. **Verify models loaded:**
   ```bash
   ollama list
   ```

---

### ChromaDB Not Responding

**Symptom:**

```
Health check: ChromaDB FAILED
```

**Solutions (on server):**

1. **Check Docker container:**

   ```bash
   docker ps | grep chroma
   ```

2. **Restart container:**
   ```bash
   cd ~/.codefind-server
   docker compose restart
   ```

---

## Configuration Issues

### Config File Not Found

**Symptom:**

```
Error: config file not found
```

**Solutions:**

1. **Run init:**

   ```bash
   codefind init
   ```

2. **Check config location:**
   ```bash
   ls ~/.codefind/
   ```

---

## Getting Help

- Check `codefind --help` for command usage
- Check `codefind <command> --help` for specific command help
- Review [CLI Commands Reference](./CLI-COMMANDS-AND-WORKFLOW.md)
