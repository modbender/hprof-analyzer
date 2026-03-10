# Java Heap Dump (HPROF) Analyzer — Research

## The Gap

There is no proper CLI equivalent to Eclipse MAT for Java heap dump analysis.
MAT is the gold standard but it's a GUI tool. Its "batch mode" (`ParseHeapDump.sh`) is clunky,
requires a full Eclipse runtime, and is painful to automate.

No single CLI tool provides: dominator trees, retained size computation, OQL queries,
leak suspect reports, class histograms, and GC root analysis — all from the terminal.

## Existing Tools

### Go

| Project | Status | Notes |
|---|---|---|
| [google/hprof-parser](https://pkg.go.dev/github.com/google/hprof-parser) | Stale (~2020) | Parser only — reads hprof binary format, no analysis. Incomplete record type support. Google 20% project, abandoned. |
| [randall77/hprof](https://github.com/randall77/hprof) | Stale | Converts Go's internal heap dump to hprof. Not a Java hprof analyzer. |

### Rust

| Project | Status | Notes |
|---|---|---|
| [hprof-slurp](https://github.com/agourlay/hprof-slurp) | Active | Streaming single-pass. ~2GB/s throughput. Class histogram, top allocators, strings. **No dominator tree, retained size, OQL, or GC root analysis.** Trades features for speed. |

### Kotlin/Java

| Project | Status | Notes |
|---|---|---|
| [Shark](https://square.github.io/leakcanary/shark/) (LeakCanary) | Active | Full heap graph navigation, leak detection. Kotlin, Android-focused. Has CLI. Requires JVM. |
| [heaplib](https://github.com/aragozin/heaplib) | Semi-active | Java library for heap dump processing. Programmatic use. |
| [eaftan/hprof-parser](https://github.com/eaftan/hprof-parser) | Stale | Java extensible parser. |

### Python

| Project | Status | Notes |
|---|---|---|
| [py-hprof](https://github.com/SonyMobile/py-hprof) | Stale | Read-only access to hprof files. Slow, incomplete, abandoned. |

### Other

- **jhat** (Oracle) — deprecated since Java 9. Dead.
- **HeapHero** — SaaS only, not a CLI tool.
- **Auto-MAT** ([jfrog/auto-mat](https://github.com/jfrog/auto-mat)) — wraps MAT batch mode, still needs MAT installed.

## Why Go

1. **Single binary** — no JVM required (MAT's biggest pain point)
2. **Low memory footprint** — predictable memory model, can mmap files and stream
3. **Concurrency** — goroutines for parallel graph traversal (dominator tree computation)
4. **Cross-platform** — compile for Linux/Mac/Windows trivially
5. **google/hprof-parser as starting point** — parsing foundation exists (even if incomplete)
6. **hprof format is well-documented** — OpenJDK's `heapDumper.cpp` is the de facto spec

## Target Feature Set (MAT Parity)

- **Parsing**: Full HPROF 1.0.1 / 1.0.2 binary format
- **Class histogram**: Instance count + shallow size per class
- **Dominator tree**: Retained size computation (the killer feature)
- **GC root analysis**: Paths from GC roots to any object
- **OQL**: SQL-like query language over the object graph
- **Leak suspects**: Automated memory leak pattern detection
- **Streaming mode**: Handle dumps larger than available RAM
- **Output formats**: Human-readable tables, JSON, CSV
