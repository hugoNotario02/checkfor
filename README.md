# checkfor

[![Tests](https://github.com/hegner123/checkfor/actions/workflows/test.yml/badge.svg)](https://github.com/hegner123/checkfor/actions/workflows/test.yml)
[![Go Version](https://img.shields.io/badge/go-1.23-blue.svg)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/hegner123/checkfor)](https://goreportcard.com/report/github.com/hegner123/checkfor)

A lightweight CLI tool for searching files in a directory with JSON output. Designed for token-efficient verification during refactoring tasks.

## Features

- Single-depth directory scanning (non-recursive)
- File extension filtering
- Case-insensitive search option
- Whole-word matching option
- Context lines (show N lines before/after matches)
- JSON output for easy parsing

## Installation

### Build from source

```bash
go build -o checkfor
```

### Install to system PATH

```bash
sudo cp checkfor /usr/local/bin/
```

Or install to user-local bin (ensure `~/bin` or `~/.local/bin` is in your PATH):

```bash
mkdir -p ~/bin
cp checkfor ~/bin/
```

### Run tests

```bash
go test -v
```

### Configure for Claude Code (Recommended)

Add this to your global `~/.claude/CLAUDE.md` to help Claude Code use checkfor optimally:

```markdown
## Tool Usage - Search Optimization

### When to use checkfor (MCP tool)
Use the `checkfor` tool when searching a SINGLE directory at SINGLE depth (non-recursive) for a string pattern. This tool is optimized for token efficiency and returns minimal JSON output.

**Use checkfor when:**
- Verifying refactoring completion in a specific directory
- Checking a single package/module directory for remaining instances of old patterns
- Searching files in one directory without subdirectories
- You need minimal, token-efficient output

**Example:**

checkfor tool with:
- dir: "/absolute/path/to/directory"
- search: "oldFunctionName"
- ext: ".go" (optional, filters by extension)
- case_insensitive: false (optional)
- whole_word: false (optional)
- context: 0 (optional, number of context lines)


### When NOT to use checkfor
- Recursive/deep directory searches (use Grep instead)
- Complex regex patterns (use Grep instead)
- Searching across multiple directories (use Grep instead)
- When you need full file content (use Read instead)
```

This helps Claude Code automatically choose the most efficient tool for verification tasks.

## Usage

```bash
checkfor --dir <directory> --search <string> [options]
```

### Required Flags

- `--dir` - Directory to search
- `--search` - String to search for

### Optional Flags

- `--ext` - File extension to filter (e.g., `.go`, `.txt`, `.js`)
- `--case-insensitive` - Perform case-insensitive search
- `--whole-word` - Match whole words only
- `--context` - Number of context lines before and after each match (default: 0)

## Examples

### Basic search
```bash
checkfor --dir ./src --search "oldFunctionName"
```

### Search Go files only
```bash
checkfor --dir ./pkg/handlers --search "UserModel" --ext .go
```

### Case-insensitive search
```bash
checkfor --dir ./templates --search "todo" --case-insensitive
```

### Whole word matching
```bash
checkfor --dir ./utils --search "log" --whole-word --ext .go
```

### With context lines
```bash
checkfor --dir ./services --search "deprecated" --context 2
```

## Best Practices

- **Plan verification steps:** Include specific checkfor commands in your plans to ensure systematic progress tracking
- **Verify incrementally:** Run checkfor after each major file change to catch issues early
- **Track progress:** Use match counts to measure refactoring completion (e.g., "32 matches → 17 matches → 0 matches")
- **Document patterns:** Note which patterns to search for in your project documentation
- **Use whole-word matching:** Add `--whole-word` flag to avoid false positives from similar names
- **Add context when needed:** Use `--context 1` or `--context 2` to understand surrounding code

## Output Format

The tool outputs JSON with the following structure:

```json
{
  "matches_found": 2,
  "files": [
    {
      "path": "handler.go",
      "matches": [
        {
          "line": 42,
          "content": "  result := oldFunctionName()",
          "context_before": [
            "func processRequest() {",
            "  // Process the request"
          ],
          "context_after": [
            "  return result",
            "}"
          ]
        }
      ]
    }
  ]
}
```

## MCP Server Mode

`checkfor` can run as an MCP (Model Context Protocol) server for integration with Claude Code and other MCP clients.

### Setup

1. Build and install the binary to your PATH (see Installation above)

2. Add to your Claude Code MCP configuration (`~/.claude/mcp.json` or project `.mcp.json`):

```json
{
  "mcpServers": {
    "checkfor": {
      "command": "checkfor",
      "args": ["--mcp"]
    }
  }
}
```

3. Restart Claude Code or reload MCP servers

### Usage

Once configured, the `checkfor` tool will be available in Claude Code as an MCP tool with the same parameters as the CLI:

- `dir` - Directory path (absolute path required)
- `search` - String pattern to search for
- `ext` - File extension filter (optional)
- `case_insensitive` - Case-insensitive search (optional)
- `whole_word` - Match whole words only (optional)
- `context` - Number of context lines (optional)

## Performance

checkfor is designed for token efficiency during AI-assisted refactoring workflows. In a real-world multi-phase refactoring session:

- **~8,000 tokens** used with checkfor
- **~155,250 tokens** would have been used with Read tool (19.4x more efficient)
- **~35,100 tokens** would have been used with Grep tool (4.4x more efficient)

This efficiency prevented exceeding the 200K token context limit and enabled completion in a single session, saving approximately $1.77 in API costs for the project.

**Key benefits:**
- 95% token reduction vs Read tool
- 77% token reduction vs Grep tool
- 3-5x faster response time
- Near-zero error rate with exact counts

See [detailed case study](docs/CASE_STUDY.md) for full analysis with real-world data.

## Exit Codes

- `0` - Success (matches found or not)
- `1` - Error (invalid arguments, directory not found, etc.)
