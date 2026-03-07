# CLI Install

Build the Phase 4 CLI binary from the repo root:

```bash
go build -o ./bin/codefind ./cmd/codefind
```

Verify it runs locally:

```bash
./bin/codefind --help
```

Install it globally to `/usr/local/bin/codefind`:

```bash
sudo install -m 0755 ./bin/codefind /usr/local/bin/codefind
```

Verify the global install:

```bash
codefind --help
which codefind
```

Upgrade flow:

```bash
go build -o ./bin/codefind ./cmd/codefind
sudo install -m 0755 ./bin/codefind /usr/local/bin/codefind
```

The privileged `/usr/local/bin` step is user-run. The repo does not automate `sudo`.
