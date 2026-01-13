package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Match struct {
	Line          int      `json:"line"`
	Content       string   `json:"content"`
	ContextBefore []string `json:"context_before,omitempty"`
	ContextAfter  []string `json:"context_after,omitempty"`
}

type FileMatch struct {
	Path    string  `json:"path"`
	Matches []Match `json:"matches"`
}

type DirectoryResult struct {
	Dir             string      `json:"dir"`
	MatchesFound    int         `json:"matches_found"`
	OriginalMatches int         `json:"original_matches,omitempty"`
	FilteredMatches int         `json:"filtered_matches,omitempty"`
	Files           []FileMatch `json:"files"`
}

type Result struct {
	Directories []DirectoryResult `json:"directories"`
}

type Config struct {
	Dirs            []string
	Search          string
	Ext             string
	Exclude         []string
	CaseInsensitive bool
	WholeWord       bool
	Context         int
	HideFilterStats bool
	CLIMode         bool
}

// MCP JSON-RPC types
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type InitializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	ServerInfo      ServerInfo   `json:"serverInfo"`
	Capabilities    Capabilities `json:"capabilities"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Capabilities struct {
	Tools map[string]bool `json:"tools"`
}

type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required"`
}

type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Default     any    `json:"default,omitempty"`
}

type ToolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type ToolCallResult struct {
	Content []ContentItem `json:"content"`
}

type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func main() {
	config := parseFlags()

	if config.CLIMode {
		runCLI(config)
		return
	}

	runMCPServer()
}

func parseFlags() Config {
	config := Config{}
	var dirStr string
	var excludeStr string

	flag.BoolVar(&config.CLIMode, "cli", false, "Run in CLI mode (default is MCP server mode)")
	flag.StringVar(&dirStr, "dir", "", "Comma-separated list of directories to search (defaults to current directory)")
	flag.StringVar(&config.Search, "search", "", "String to search for (required)")
	flag.StringVar(&config.Ext, "ext", "", "File extension to filter (e.g., .go, .rtf)")
	flag.StringVar(&excludeStr, "exclude", "", "Comma-separated list of strings to exclude from results")
	flag.BoolVar(&config.CaseInsensitive, "case-insensitive", false, "Perform case-insensitive search")
	flag.BoolVar(&config.WholeWord, "whole-word", false, "Match whole words only")
	flag.IntVar(&config.Context, "context", 0, "Number of context lines before and after match")
	flag.BoolVar(&config.HideFilterStats, "hide-filter-stats", false, "Hide original_matches and filtered_matches from output")

	flag.Parse()

	if dirStr != "" {
		config.Dirs = strings.Split(dirStr, ",")
		for i := range config.Dirs {
			config.Dirs[i] = strings.TrimSpace(config.Dirs[i])
		}
	} else {
		config.Dirs = []string{"."}
	}

	if excludeStr != "" {
		config.Exclude = strings.Split(excludeStr, ",")
		for i := range config.Exclude {
			config.Exclude[i] = strings.TrimSpace(config.Exclude[i])
		}
	}

	return config
}

func runCLI(config Config) {
	if config.Search == "" {
		fmt.Fprintln(os.Stderr, "Error: --search is required")
		flag.Usage()
		os.Exit(1)
	}

	result, err := searchDirectories(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	output, err := json.Marshal(result)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output))
}

func runMCPServer() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			sendError(nil, -32700, "Parse error")
			continue
		}

		handleRequest(req)
	}
}

func handleRequest(req JSONRPCRequest) {
	switch req.Method {
	case "initialize":
		handleInitialize(req)
	case "tools/list":
		handleToolsList(req)
	case "tools/call":
		handleToolsCall(req)
	default:
		sendError(req.ID, -32601, "Method not found")
	}
}

func handleInitialize(req JSONRPCRequest) {
	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
		ServerInfo: ServerInfo{
			Name:    "checkfor",
			Version: "1.0.0",
		},
		Capabilities: Capabilities{
			Tools: map[string]bool{
				"list": true,
				"call": true,
			},
		},
	}
	sendResponse(req.ID, result)
}

func handleToolsList(req JSONRPCRequest) {
	result := ToolsListResult{
		Tools: []Tool{
			{
				Name:        "checkfor",
				Description: "Search files in directories for a string pattern. Single-depth (non-recursive) scanning with optional extension filtering, case-insensitive search, whole-word matching, and context lines.",
				InputSchema: InputSchema{
					Type: "object",
					Properties: map[string]Property{
						"dir": {
							Type:        "array",
							Description: "Array of directory paths to search. Can also accept a single string for backwards compatibility. Defaults to current directory if not provided.",
						},
						"search": {
							Type:        "string",
							Description: "String pattern to search for",
						},
						"ext": {
							Type:        "string",
							Description: "File extension to filter (e.g., '.go', '.rtf'). Optional.",
						},
						"exclude": {
							Type:        "array",
							Description: "Array of strings to exclude from results. Matches containing any of these strings will be filtered out. Optional.",
						},
						"case_insensitive": {
							Type:        "boolean",
							Description: "Perform case-insensitive search. Optional, defaults to false.",
							Default:     false,
						},
						"whole_word": {
							Type:        "boolean",
							Description: "Match whole words only. Optional, defaults to false.",
							Default:     false,
						},
						"context": {
							Type:        "integer",
							Description: "Number of context lines before and after each match. Optional, defaults to 0.",
							Default:     0,
						},
						"hide_filter_stats": {
							Type:        "boolean",
							Description: "Hide original_matches and filtered_matches from output. Optional, defaults to false.",
							Default:     false,
						},
					},
					Required: []string{"search"},
				},
			},
		},
	}
	sendResponse(req.ID, result)
}

func handleToolsCall(req JSONRPCRequest) {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		sendError(req.ID, -32602, "Invalid params")
		return
	}

	if params.Name != "checkfor" {
		sendError(req.ID, -32602, "Unknown tool")
		return
	}

	search, ok := params.Arguments["search"].(string)
	if !ok {
		sendError(req.ID, -32602, "Missing or invalid 'search' parameter")
		return
	}

	config := Config{
		Search: search,
	}

	if dirParam, exists := params.Arguments["dir"]; exists {
		switch v := dirParam.(type) {
		case string:
			config.Dirs = []string{v}
		case []any:
			config.Dirs = make([]string, 0, len(v))
			for _, d := range v {
				if str, ok := d.(string); ok {
					config.Dirs = append(config.Dirs, str)
				}
			}
		}
	}

	if len(config.Dirs) == 0 {
		config.Dirs = []string{"."}
	}

	if ext, ok := params.Arguments["ext"].(string); ok {
		config.Ext = ext
	}

	if excludeArray, ok := params.Arguments["exclude"].([]any); ok {
		config.Exclude = make([]string, 0, len(excludeArray))
		for _, v := range excludeArray {
			if str, ok := v.(string); ok {
				config.Exclude = append(config.Exclude, str)
			}
		}
	}

	if caseInsensitive, ok := params.Arguments["case_insensitive"].(bool); ok {
		config.CaseInsensitive = caseInsensitive
	}

	if wholeWord, ok := params.Arguments["whole_word"].(bool); ok {
		config.WholeWord = wholeWord
	}

	if context, ok := params.Arguments["context"].(float64); ok {
		config.Context = int(context)
	}

	if hideFilterStats, ok := params.Arguments["hide_filter_stats"].(bool); ok {
		config.HideFilterStats = hideFilterStats
	}

	result, err := searchDirectories(config)
	if err != nil {
		sendError(req.ID, -32603, fmt.Sprintf("Search failed: %v", err))
		return
	}

	jsonResult, err := json.Marshal(result)
	if err != nil {
		sendError(req.ID, -32603, "Failed to marshal result")
		return
	}

	response := ToolCallResult{
		Content: []ContentItem{
			{
				Type: "text",
				Text: string(jsonResult),
			},
		},
	}

	sendResponse(req.ID, response)
}

func sendResponse(id any, result any) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	data, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal response: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

func sendError(id any, code int, message string) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &Error{
			Code:    code,
			Message: message,
		},
	}
	data, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal error response: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

func searchDirectories(config Config) (*Result, error) {
	result := &Result{
		Directories: make([]DirectoryResult, 0, len(config.Dirs)),
	}

	for _, dir := range config.Dirs {
		dirResult, err := searchDirectory(dir, config)
		if err != nil {
			return nil, err
		}

		result.Directories = append(result.Directories, *dirResult)
	}

	return result, nil
}

func searchDirectory(dir string, config Config) (*DirectoryResult, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	dirResult := &DirectoryResult{
		Dir:   dir,
		Files: make([]FileMatch, 0),
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()

		if config.Ext != "" && !strings.HasSuffix(filename, config.Ext) {
			continue
		}

		fullPath := filepath.Join(dir, filename)
		matches, originalCount, filteredCount, err := searchFile(fullPath, config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to search %s: %v\n", fullPath, err)
			continue
		}

		if !config.HideFilterStats && len(config.Exclude) > 0 {
			dirResult.OriginalMatches += originalCount
			dirResult.FilteredMatches += filteredCount
		}

		if len(matches) > 0 {
			dirResult.Files = append(dirResult.Files, FileMatch{
				Path:    filename,
				Matches: matches,
			})
			dirResult.MatchesFound += len(matches)
		}
	}

	return dirResult, nil
}

func searchFile(path string, config Config) ([]Match, int, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, 0, err
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close file %s: %v\n", path, cerr)
		}
	}()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, 0, 0, err
	}

	var matches []Match
	originalCount := 0
	filteredCount := 0
	searchTerm := config.Search
	if config.CaseInsensitive {
		searchTerm = strings.ToLower(searchTerm)
	}

	for i, line := range lines {
		lineToCheck := line
		if config.CaseInsensitive {
			lineToCheck = strings.ToLower(line)
		}

		found := false
		if config.WholeWord {
			found = containsWholeWord(lineToCheck, searchTerm)
		} else {
			found = strings.Contains(lineToCheck, searchTerm)
		}

		if found {
			originalCount++
			excluded := false
			for _, excludePattern := range config.Exclude {
				excludeToCheck := excludePattern
				lineForExclude := line
				if config.CaseInsensitive {
					excludeToCheck = strings.ToLower(excludePattern)
					lineForExclude = lineToCheck
				}
				if strings.Contains(lineForExclude, excludeToCheck) {
					excluded = true
					break
				}
			}

			if excluded {
				filteredCount++
			} else {
				match := Match{
					Line:    i + 1,
					Content: line,
				}

				if config.Context > 0 {
					match.ContextBefore = getContextBefore(lines, i, config.Context)
					match.ContextAfter = getContextAfter(lines, i, config.Context)
				}

				matches = append(matches, match)
			}
		}
	}

	return matches, originalCount, filteredCount, nil
}

func containsWholeWord(text, word string) bool {
	if !strings.Contains(text, word) {
		return false
	}

	startIdx := 0
	for {
		idx := strings.Index(text[startIdx:], word)
		if idx == -1 {
			return false
		}

		actualIdx := startIdx + idx

		beforeOk := actualIdx == 0 || !isWordChar(rune(text[actualIdx-1]))
		afterIdx := actualIdx + len(word)
		afterOk := afterIdx >= len(text) || !isWordChar(rune(text[afterIdx]))

		if beforeOk && afterOk {
			return true
		}

		startIdx = actualIdx + 1
	}
}

func isWordChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}

func getContextBefore(lines []string, currentIdx, count int) []string {
	start := currentIdx - count
	if start < 0 {
		start = 0
	}
	return lines[start:currentIdx]
}

func getContextAfter(lines []string, currentIdx, count int) []string {
	end := currentIdx + count + 1
	if end > len(lines) {
		end = len(lines)
	}
	return lines[currentIdx+1 : end]
}
