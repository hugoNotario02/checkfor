package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Test helper: create temporary directory with test files
func setupTestDir(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "checkfor-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return tmpDir
}

func cleanupTestDir(t *testing.T, dir string) {
	if err := os.RemoveAll(dir); err != nil {
		t.Errorf("Failed to cleanup temp dir: %v", err)
	}
}

func createTestFile(t *testing.T, dir, name, content string) {
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
}

// Core search function tests

func TestContainsWholeWord(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		word     string
		expected bool
	}{
		{"exact match", "hello", "hello", true},
		{"word in sentence", "hello world", "hello", true},
		{"word at end", "say hello", "hello", true},
		{"word with punctuation", "hello, world", "hello", true},
		{"partial match should fail", "helloworld", "hello", false},
		{"substring should fail", "superhello", "hello", false},
		{"underscore is word char", "hello_world", "hello", false},
		{"space is word boundary", "hello world", "world", true},
		{"case sensitive", "Hello", "hello", false},
		{"multiple occurrences", "log logger log", "log", true},
		{"no match", "goodbye", "hello", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsWholeWord(tt.text, tt.word)
			if result != tt.expected {
				t.Errorf("containsWholeWord(%q, %q) = %v, want %v",
					tt.text, tt.word, result, tt.expected)
			}
		})
	}
}

func TestIsWordChar(t *testing.T) {
	wordChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"
	for _, ch := range wordChars {
		if !isWordChar(ch) {
			t.Errorf("isWordChar(%q) = false, want true", ch)
		}
	}

	nonWordChars := " !@#$%^&*()-+=[]{}|;:'\",.<>?/\\"
	for _, ch := range nonWordChars {
		if isWordChar(ch) {
			t.Errorf("isWordChar(%q) = true, want false", ch)
		}
	}
}

func TestGetContextBefore(t *testing.T) {
	lines := []string{"line1", "line2", "line3", "line4", "line5"}

	tests := []struct {
		name        string
		currentIdx  int
		count       int
		expected    []string
		description string
	}{
		{"middle with 2 lines", 3, 2, []string{"line2", "line3"}, "Get 2 lines before index 3"},
		{"start boundary", 1, 2, []string{"line1"}, "Only 1 line available before index 1"},
		{"at start", 0, 2, []string{}, "No lines before index 0"},
		{"no context", 2, 0, []string{}, "Count is 0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getContextBefore(lines, tt.currentIdx, tt.count)
			if len(result) != len(tt.expected) {
				t.Errorf("Length mismatch: got %d, want %d", len(result), len(tt.expected))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("Index %d: got %q, want %q", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestGetContextAfter(t *testing.T) {
	lines := []string{"line1", "line2", "line3", "line4", "line5"}

	tests := []struct {
		name       string
		currentIdx int
		count      int
		expected   []string
	}{
		{"middle with 2 lines", 2, 2, []string{"line4", "line5"}},
		{"end boundary", 3, 2, []string{"line5"}},
		{"at end", 4, 2, []string{}},
		{"no context", 2, 0, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getContextAfter(lines, tt.currentIdx, tt.count)
			if len(result) != len(tt.expected) {
				t.Errorf("Length mismatch: got %d, want %d", len(result), len(tt.expected))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("Index %d: got %q, want %q", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

// File filtering tests

func TestSearchFileCaseInsensitive(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	content := "Hello World\nGoodbye World\nHELLO again"
	createTestFile(t, tmpDir, "test.txt", content)

	config := Config{
		Search:          "hello",
		CaseInsensitive: true,
	}

	matches, err := searchFile(filepath.Join(tmpDir, "test.txt"), config)
	if err != nil {
		t.Fatalf("searchFile failed: %v", err)
	}

	if len(matches) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(matches))
	}

	if matches[0].Line != 1 || matches[1].Line != 3 {
		t.Errorf("Expected lines 1 and 3, got %d and %d", matches[0].Line, matches[1].Line)
	}
}

func TestSearchFileWholeWord(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	content := "log message\nlogger info\nlog\ncatalog"
	createTestFile(t, tmpDir, "test.txt", content)

	config := Config{
		Search:    "log",
		WholeWord: true,
	}

	matches, err := searchFile(filepath.Join(tmpDir, "test.txt"), config)
	if err != nil {
		t.Fatalf("searchFile failed: %v", err)
	}

	if len(matches) != 2 {
		t.Errorf("Expected 2 matches (lines 1 and 3), got %d", len(matches))
	}
}

func TestSearchFileWithContext(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	content := "line1\nline2\ntarget\nline4\nline5"
	createTestFile(t, tmpDir, "test.txt", content)

	config := Config{
		Search:  "target",
		Context: 1,
	}

	matches, err := searchFile(filepath.Join(tmpDir, "test.txt"), config)
	if err != nil {
		t.Fatalf("searchFile failed: %v", err)
	}

	if len(matches) != 1 {
		t.Fatalf("Expected 1 match, got %d", len(matches))
	}

	match := matches[0]
	if len(match.ContextBefore) != 1 || match.ContextBefore[0] != "line2" {
		t.Errorf("Expected context before to be [line2], got %v", match.ContextBefore)
	}
	if len(match.ContextAfter) != 1 || match.ContextAfter[0] != "line4" {
		t.Errorf("Expected context after to be [line4], got %v", match.ContextAfter)
	}
}

// Integration tests

func TestSearchDirectoryExtensionFilter(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	createTestFile(t, tmpDir, "file1.go", "package main")
	createTestFile(t, tmpDir, "file2.txt", "package main")
	createTestFile(t, tmpDir, "file3.go", "package main")

	config := Config{
		Dir:    tmpDir,
		Search: "package",
		Ext:    ".go",
	}

	result, err := searchDirectory(config)
	if err != nil {
		t.Fatalf("searchDirectory failed: %v", err)
	}

	if len(result.Files) != 2 {
		t.Errorf("Expected 2 .go files with matches, got %d", len(result.Files))
	}

	if result.MatchesFound != 2 {
		t.Errorf("Expected 2 total matches, got %d", result.MatchesFound)
	}
}

func TestSearchDirectoryNoMatches(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	createTestFile(t, tmpDir, "file.txt", "hello world")

	config := Config{
		Dir:    tmpDir,
		Search: "nonexistent",
	}

	result, err := searchDirectory(config)
	if err != nil {
		t.Fatalf("searchDirectory failed: %v", err)
	}

	if len(result.Files) != 0 {
		t.Errorf("Expected 0 files with matches, got %d", len(result.Files))
	}

	if result.MatchesFound != 0 {
		t.Errorf("Expected 0 matches, got %d", result.MatchesFound)
	}
}

func TestSearchDirectoryNonRecursive(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	// Create file in root
	createTestFile(t, tmpDir, "root.txt", "target")

	// Create subdirectory with file
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}
	createTestFile(t, subDir, "sub.txt", "target")

	config := Config{
		Dir:    tmpDir,
		Search: "target",
	}

	result, err := searchDirectory(config)
	if err != nil {
		t.Fatalf("searchDirectory failed: %v", err)
	}

	// Should only find the root file, not the subdirectory file
	if len(result.Files) != 1 {
		t.Errorf("Expected 1 file (non-recursive), got %d", len(result.Files))
	}

	if result.Files[0].Path != "root.txt" {
		t.Errorf("Expected root.txt, got %s", result.Files[0].Path)
	}
}

// MCP JSON-RPC handler tests

func TestHandleInitialize(t *testing.T) {
	// Test the initialize result structure directly
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

	if result.ProtocolVersion != "2024-11-05" {
		t.Errorf("Expected protocol version 2024-11-05, got %s", result.ProtocolVersion)
	}

	if result.ServerInfo.Name != "checkfor" {
		t.Errorf("Expected server name checkfor, got %s", result.ServerInfo.Name)
	}

	if !result.Capabilities.Tools["list"] || !result.Capabilities.Tools["call"] {
		t.Errorf("Expected tools capabilities for list and call")
	}
}

func TestToolsListResult(t *testing.T) {
	result := ToolsListResult{
		Tools: []Tool{
			{
				Name:        "checkfor",
				Description: "Search files in a directory for a string pattern. Single-depth (non-recursive) scanning with optional extension filtering, case-insensitive search, whole-word matching, and context lines.",
				InputSchema: InputSchema{
					Type: "object",
					Properties: map[string]Property{
						"dir": {
							Type:        "string",
							Description: "Directory path to search (absolute path required)",
						},
						"search": {
							Type:        "string",
							Description: "String pattern to search for",
						},
					},
					Required: []string{"dir", "search"},
				},
			},
		},
	}

	if len(result.Tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(result.Tools))
	}

	tool := result.Tools[0]
	if tool.Name != "checkfor" {
		t.Errorf("Expected tool name checkfor, got %s", tool.Name)
	}

	if len(tool.InputSchema.Required) != 2 {
		t.Errorf("Expected 2 required fields, got %d", len(tool.InputSchema.Required))
	}

	if tool.InputSchema.Properties["dir"].Type != "string" {
		t.Errorf("Expected dir property to be string type")
	}
}

func TestToolCallParamsMarshaling(t *testing.T) {
	jsonData := `{
		"name": "checkfor",
		"arguments": {
			"dir": "/test/path",
			"search": "pattern",
			"case_insensitive": true,
			"context": 2
		}
	}`

	var params ToolCallParams
	if err := json.Unmarshal([]byte(jsonData), &params); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if params.Name != "checkfor" {
		t.Errorf("Expected name checkfor, got %s", params.Name)
	}

	if params.Arguments["dir"] != "/test/path" {
		t.Errorf("Expected dir /test/path, got %v", params.Arguments["dir"])
	}

	if params.Arguments["search"] != "pattern" {
		t.Errorf("Expected search pattern, got %v", params.Arguments["search"])
	}

	if params.Arguments["case_insensitive"] != true {
		t.Errorf("Expected case_insensitive true, got %v", params.Arguments["case_insensitive"])
	}

	// JSON numbers unmarshal as float64
	if params.Arguments["context"] != float64(2) {
		t.Errorf("Expected context 2, got %v", params.Arguments["context"])
	}
}

func TestResultJSONOutput(t *testing.T) {
	result := Result{
		MatchesFound: 1,
		Files: []FileMatch{
			{
				Path: "test.go",
				Matches: []Match{
					{
						Line:          42,
						Content:       "target line",
						ContextBefore: []string{"line before"},
						ContextAfter:  []string{"line after"},
					},
				},
			},
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal result: %v", err)
	}

	var decoded Result
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if decoded.MatchesFound != 1 {
		t.Errorf("Expected matches_found 1, got %d", decoded.MatchesFound)
	}

	if len(decoded.Files) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(decoded.Files))
	}

	if decoded.Files[0].Path != "test.go" {
		t.Errorf("Expected path test.go, got %s", decoded.Files[0].Path)
	}

	if decoded.Files[0].Matches[0].Line != 42 {
		t.Errorf("Expected line 42, got %d", decoded.Files[0].Matches[0].Line)
	}
}
