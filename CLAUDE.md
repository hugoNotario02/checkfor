# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`checkfor` is an MCP server tool for multi-directory file searching with compact JSON output. It's optimized for AI-driven, token-efficient verification during refactoring tasks. The tool operates primarily as an MCP server with an optional CLI mode for testing.

## Updates

### Automatic Update Notifications

checkfor checks for updates every 6 hours when running as an MCP server. When a new version is available, you'll see stderr output like:

```
[checkfor] Update available: v1.0.0 → v1.2.0
[checkfor] GitHub: https://github.com/hegner123/checkfor/releases/tag/v1.2.0
[checkfor] To update: checkfor --update
```

### Updating checkfor

When Claude sees an update notification:

1. **Inform the user** about the available update
2. **Ask permission** to update
3. **Run the update**:
   ```bash
   checkfor --update
   ```
4. **Restart the MCP server** (inform user they need to restart Claude Code or reload MCP servers)

Alternatively, users can update manually:
```bash
go install github.com/hegner123/checkfor@latest
```

### Update Cache

Update checks are cached in `~/.checkfor-update-cache` to respect GitHub API rate limits (60 requests/hour unauthenticated). The cache is valid for 6 hours during alpha development.

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
- Core search functions (whole-word matching, context extraction, exclude filtering)
- File filtering (extension, case-insensitive, whole-word)
- Filter statistics tracking (original_matches, filtered_matches)
- MCP JSON-RPC protocol compliance
- Integration tests (multi-directory scanning, non-recursive behavior, per-directory results)

### MCP Mode (Default)
```bash
checkfor
```

The MCP server runs by default and communicates via JSON-RPC 2.0 over stdin/stdout. Configuration is in `.mcp.json`.

### CLI Usage (Testing/Scripting)
```bash
checkfor --cli --search <string> [options]
```

Required flags:
- `--search` - String pattern to search for

Optional flags:
- `--cli` - Run in CLI mode (default is MCP server mode)
- `--dir` - Comma-separated list of directories to search (defaults to current directory)
- `--ext` - File extension filter (e.g., `.go`, `.txt`)
- `--exclude` - Comma-separated list of strings to exclude from results
- `--case-insensitive` - Case-insensitive search
- `--whole-word` - Match whole words only
- `--context` - Number of context lines (default: 0)
- `--hide-filter-stats` - Hide original_matches and filtered_matches from output

## Architecture

### Mode Design

The application operates in two modes determined at startup:

1. **MCP Server Mode (Default)**: JSON-RPC 2.0 server for integration with MCP-compatible clients (Claude Code)
2. **CLI Mode**: Standard command-line tool that outputs JSON results to stdout (requires `--cli` flag)

Mode selection happens in `main()` based on the `--cli` flag. Default behavior is MCP server mode.

### Search Algorithm

The search flow:
1. `searchDirectories()` - Iterates over all provided directories
2. `searchDirectory()` - Processes single directory, returns DirectoryResult
3. `searchFile()` - Processes individual files, applies filters, returns matches with statistics

**Key features:**
- **Non-recursive**: Only scans immediate directory contents (skips subdirectories)
- **Multi-directory**: Searches multiple directories in one invocation
- **Exclude filtering**: Filters out matches containing any of the exclude patterns
- **Filter statistics**: Tracks original_matches (before filtering) and filtered_matches (excluded count)
- **Extension filtering**: Applied before file reading for efficiency
- **Per-file searching**: Each file processed independently

Key implementation details:
- Files are read entirely into memory as line slices
- Whole-word matching uses custom `containsWholeWord()` that checks word boundaries using `isWordChar()` (alphanumeric + underscore)
- Context lines extracted via `getContextBefore()` and `getContextAfter()`
- Case-insensitive search converts both search term and line content to lowercase
- Exclude patterns also respect case-insensitive flag

### Output Format

**Compact JSON with per-directory structure:**
```json
{"directories":[{"dir":"./pkg","matches_found":3,"original_matches":5,"filtered_matches":2,"files":[{"path":"user.go","matches":[...]}]}]}
```

**Token efficiency:**
- Compact JSON (no newlines): 41% smaller
- Relative file paths: 68% reduction in path data
- Per-directory organization: Better AI reasoning, no path parsing needed

### MCP Protocol Implementation

The MCP server implements three JSON-RPC methods:
- `initialize`: Returns protocol version "2024-11-05" and server capabilities
  - Includes `listChanged: true` capability for future dynamic tool updates
  - Reports current version number
- `tools/list`: Exposes the "checkfor" tool with its schema
- `tools/call`: Executes searches and returns formatted results

### Update Checking

On MCP server startup, a background goroutine checks for updates:
- Non-blocking check with 3-second timeout
- Checks GitHub releases API for latest version
- Caches results for 6 hours to respect rate limits
- Notifies via stderr when updates are available
- Semantic versioning comparison (major.minor.patch)

### Data Structures

Key types:
- `Config`: Unified configuration for both CLI and MCP modes
  - `Dirs []string` - List of directories to search
  - `Exclude []string` - List of exclude patterns
  - `HideFilterStats bool` - Whether to hide filter statistics
  - `CLIMode bool` - Whether to run in CLI mode
- `Result`: Top-level result with `Directories []DirectoryResult`
- `DirectoryResult`: Per-directory results with:
  - `Dir string` - Directory path
  - `MatchesFound int` - Matches in this directory after filtering
  - `OriginalMatches int` - Matches before filtering (omitted if no exclude or hidden)
  - `FilteredMatches int` - Excluded matches count (omitted if no exclude or hidden)
  - `Files []FileMatch` - File matches with relative paths
- `FileMatch`: Contains relative file path and array of matches
- `Match`: Line number, content, and optional context arrays

## Important Notes

- The tool is **single-depth only** - it does not recurse into subdirectories
- Default mode is **MCP server** - runs without flags
- CLI mode requires `--cli` flag
- File paths in results are **relative to each directory**
- All JSON output is **compact** (no indentation/newlines) for token efficiency
- Filter statistics only appear when exclude patterns are used and not hidden
- Multi-directory support enables controlled searches across specific packages
- Warnings for unreadable files go to stderr
- MCP mode expects one JSON-RPC request per line on stdin
