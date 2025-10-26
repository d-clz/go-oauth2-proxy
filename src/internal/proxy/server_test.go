package proxy

import (
	"testing"
)

func TestMatchPath(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		want    bool
	}{
		// Exact matches
		{"/run_sse", "/run_sse", true},
		{"/run_sse", "/run_sse/other", false},
		{"/apps", "/apps", true},

		// Wildcard matches
		{"/apps/*", "/apps", true},
		{"/apps/*", "/apps/", true},
		{"/apps/*", "/apps/foo", true},
		{"/apps/*", "/apps/foo/bar", true},
		{"/apps/*", "/other", false},
		{"/apps/*", "/apps", true},  // Should match the prefix itself

		// Double wildcard matches
		{"/apps/**", "/apps", true},
		{"/apps/**", "/apps/", true},
		{"/apps/**", "/apps/foo", true},
		{"/apps/**", "/apps/foo/bar/baz", true},
		{"/apps/**", "/other", false},

		// No match cases
		{"/run_sse", "/inform", false},
		{"/apps/*", "/inform", false},
		{"/api/*", "/apps/test", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.path, func(t *testing.T) {
			got := matchPath(tt.pattern, tt.path)
			if got != tt.want {
				t.Errorf("matchPath(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}
