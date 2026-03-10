# hprof-analyzer

A fast CLI tool for analyzing Java HPROF heap dumps — a lightweight alternative to Eclipse MAT.

## Features

- **Streaming analysis** — summary, histogram, and string extraction without building a full index
- **In-memory indexing** — build an index for deep analysis of object graphs
- **Dominator tree** — find the largest retained-size contributors
- **GC root paths** — trace why an object is kept alive
- **OQL engine** — query heap dumps with SQL-like syntax (SELECT/FROM/WHERE/GROUP BY/ORDER BY/LIMIT)
- **Leak detection** — automated leak-suspect analysis
- **Multiple output formats** — table, JSON, CSV
- **Self-upgrade** — update to the latest release with a single command
- **Zero dependencies** — single static binary, no JVM required

## Installation

### Download from releases (Linux / macOS)

```bash
curl -sLo /usr/local/bin/hprof-analyzer "https://github.com/modbender/hprof-analyzer/releases/latest/download/hprof-analyzer_$(uname -s | tr '[:upper:]' '[:lower:]')_$(uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/')"
chmod +x /usr/local/bin/hprof-analyzer
```

### Download from releases (Windows)

```powershell
$arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
Invoke-WebRequest -Uri "https://github.com/modbender/hprof-analyzer/releases/latest/download/hprof-analyzer_windows_${arch}" -OutFile "$env:LOCALAPPDATA\Microsoft\WindowsApps\hprof-analyzer.exe"
```

### Go install

```bash
go install github.com/modbender/hprof-analyzer/cmd/hprof-analyzer@latest
```

### Self-upgrade

```bash
hprof-analyzer upgrade
```

## Quick Start

```bash
# Print heap dump summary
hprof-analyzer summary dump.hprof

# Class histogram (top 20 by retained size)
hprof-analyzer histogram dump.hprof --top 20

# Find large strings
hprof-analyzer strings dump.hprof --min-length 100

# Build index for deep analysis
hprof-analyzer index dump.hprof

# Dominator tree (requires index)
hprof-analyzer domtree dump.hprof --top 10

# GC root paths for a specific object
hprof-analyzer gcroots dump.hprof --id 0x7f3a00

# OQL query
hprof-analyzer oql dump.hprof "SELECT className, count(*) FROM instanceof java.lang.String GROUP BY className"

# Automated leak detection
hprof-analyzer leaks dump.hprof

# Output as JSON
hprof-analyzer histogram dump.hprof --format json
```

## Building from Source

```bash
git clone https://github.com/modbender/hprof-analyzer.git
cd hprof-analyzer
make build
./bin/hprof-analyzer version
```

## Contributing

This project uses [conventional commits](https://www.conventionalcommits.org/):

- `feat:` — new feature (minor version bump)
- `fix:` — bug fix (patch version bump)
- `feat!:` or `BREAKING CHANGE:` — breaking change (major version bump)
- `docs:`, `test:`, `ci:`, `chore:` — no version bump

```bash
# Run tests
make test

# Run linter
make vet
```

## License

MIT
