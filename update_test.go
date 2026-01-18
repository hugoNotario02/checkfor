package main

import "testing"

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int
	}{
		{"equal versions", "1.0.0", "1.0.0", 0},
		{"v1 greater major", "2.0.0", "1.0.0", 1},
		{"v2 greater major", "1.0.0", "2.0.0", -1},
		{"v1 greater minor", "1.1.0", "1.0.0", 1},
		{"v2 greater minor", "1.0.0", "1.1.0", -1},
		{"v1 greater patch", "1.0.1", "1.0.0", 1},
		{"v2 greater patch", "1.0.0", "1.0.1", -1},
		{"with v prefix v1", "v1.2.3", "1.2.3", 0},
		{"with v prefix v2", "1.2.3", "v1.2.3", 0},
		{"both v prefix", "v1.2.3", "v1.2.3", 0},
		{"complex greater", "1.10.5", "1.9.20", 1},
		{"complex lesser", "1.9.20", "1.10.5", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareVersions(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("compareVersions(%q, %q) = %d; want %d", tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

func TestUpdateCachePath(t *testing.T) {
	path, err := getUpdateCachePath()
	if err != nil {
		t.Fatalf("getUpdateCachePath() error = %v", err)
	}

	if path == "" {
		t.Error("getUpdateCachePath() returned empty path")
	}

	// Should contain the cache filename
	if len(path) < len(updateCacheFile) {
		t.Errorf("getUpdateCachePath() = %q; expected to contain %q", path, updateCacheFile)
	}
}
