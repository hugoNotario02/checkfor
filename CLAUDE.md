# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`checkfor` is a lightweight Go CLI tool for single-depth directory file searching with JSON output. It's optimized for token-efficient verification during refactoring tasks. The tool has dual operation modes: standalone CLI and MCP (Model Context Protocol) server.

## Build and Development Commands

### Build
```bash
go build -o checkfor
```

### Install to System Path
```bash
sudo cp checkfor /usr/local/bin/
```

### Run Tests
```bash
go test -v
```

Test coverage includes:
- Core search functions (whole-word matching, context extraction)
- File filtering (extension, case-insensitive, whole-word)
- MCP JSON-RPC protocol compliance
- Integration tests (directory scanning, non-recursive behavior)

### CLI Usage
```bash
checkfor --dir <directory> --search <string> [options]
```

Required flags:
- `--dir` - Directory to search (absolute path recommended)
- `--search` - String pattern to search for

Optional flags:
- `--ext` - File extension filter (e.g., `.go`, `.txt`)
- `--case-insensitive` - Case-insensitive search
- `--whole-word` - Match whole words only
- `--context` - Number of context lines (default: 0)

### MCP Server Mode
```bash
checkfor --mcp
```

The MCP server communicates via JSON-RPC 2.0 over stdin/stdout. Configuration is in `.mcp.json`.

## Architecture

### Dual-Mode Design

The application operates in two distinct modes determined at startup:

1. **CLI Mode** (main.go:138-158): Standard command-line tool that outputs JSON results to stdout
2. **MCP Server Mode** (main.go:160-176): JSON-RPC 2.0 server for integration with MCP-compatible clients

Mode selection happens in `main()` based on the `--mcp` flag.

### Search Algorithm

The core search logic in `searchDirectory()` (main.go:345-383):
- **Non-recursive**: Only scans immediate directory contents (skips subdirectories)
- **Extension filtering**: Applied before file reading for efficiency
- **Per-file searching**: Each file processed independently via `searchFile()`

Key implementation details:
- Files are read entirely into memory as line slices (main.go:392-400)
- Whole-word matching uses custom `containsWholeWord()` (main.go:439-463) that checks word boundaries using `isWordChar()` (alphanumeric + underscore)
- Context lines extracted via `getContextBefore()` and `getContextAfter()` (main.go:469-483)
- Case-insensitive search converts both search term and line content to lowercase

### MCP Protocol Implementation

The MCP server implements three JSON-RPC methods:
- `initialize`: Returns protocol version "2024-11-05" and server capabilities
- `tools/list`: Exposes the "checkfor" tool with its schema
- `tools/call`: Executes searches and returns formatted results

### Data Structures

Key types:
- `Config`: Unified configuration for both CLI and MCP modes
- `Result`: Search results with `matches_found` count and file matches array
- `FileMatch`: Contains relative file path and array of matches
- `Match`: Line number, content, and optional context arrays

## Important Notes

- The tool is **single-depth only** - it does not recurse into subdirectories
- File paths in results are relative (filename only, not full path)
- All JSON output uses 2-space indentation
- Warnings for unreadable files go to stderr (main.go:369)
- MCP mode expects one JSON-RPC request per line on stdin
