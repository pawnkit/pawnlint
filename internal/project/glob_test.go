package project

import (
	"strings"
	"testing"
	"time"
)

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"", "a.pwn", false},
		{"*.pwn", "a.pwn", true},
		{"*.pwn", "dir/a.pwn", false},
		{"**/*.pwn", "a.pwn", true},
		{"**/*.pwn", "dir/a.pwn", true},
		{"**/*.pwn", "dir/sub/a.pwn", true},
		{"gamemodes/**", "gamemodes/a/b/c.pwn", true},
		{"gamemodes/**", "gamemodes/a.pwn", true},
		{"gamemodes/**", "other/a.pwn", false},
		{"**/vendor/**", "a/vendor/b/c.pwn", true},
		{"**/vendor/**", "vendor/c.pwn", true},
		{"**/vendor/**", "a/b/c.pwn", false},
		{"a/**/**/b.pwn", "a/b.pwn", true},
		{"a/**/**/b.pwn", "a/x/y/z/b.pwn", true},
		{"a/?.pwn", "a/x.pwn", true},
		{"a/?.pwn", "a/xy.pwn", false},
	}
	for _, tt := range tests {
		got := MatchGlob(tt.pattern, tt.path)
		if got != tt.want {
			t.Errorf("MatchGlob(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
		}
	}
}

func TestMatchGlobPathologicalPatternDoesNotHang(t *testing.T) {
	pattern := strings.Repeat("**/", 20) + "no-such-file.pwn"
	path := strings.Repeat("segment/", 30) + "other.pwn"

	done := make(chan bool, 1)
	go func() {
		done <- MatchGlob(pattern, path)
	}()

	select {
	case got := <-done:
		if got {
			t.Errorf("MatchGlob(%q, %q) = true, want false", pattern, path)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("MatchGlob did not return within 2s; likely exponential blowup")
	}
}
