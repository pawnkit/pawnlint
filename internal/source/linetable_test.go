package source_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/internal/source"
)

func TestLineTable(t *testing.T) {
	src := []byte("line one\nsecond line\nthird\n")
	lt := source.NewLineTable(src)
	if got := lt.LineCount(); got != 4 {
		t.Fatalf("LineCount: got %d want 4", got)
	}
	cases := []struct {
		off       int
		line, col int
	}{
		{0, 1, 1},
		{4, 1, 5},
		{8, 1, 9},
		{9, 2, 1},
		{20, 2, 12},
		{24, 3, 4},
		{99, 4, 1},
		{-5, 1, 1},
	}
	for _, c := range cases {
		pos := lt.Lookup(c.off)
		if pos.Line != c.line || pos.Col != c.col {
			t.Errorf("Lookup(%d): got %d:%d want %d:%d", c.off, pos.Line, pos.Col, c.line, c.col)
		}
		if pos.Offset != clamp(c.off, 0, len(src)) {
			t.Errorf("Lookup(%d).Offset: got %d want %d", c.off, pos.Offset, clamp(c.off, 0, len(src)))
		}
	}
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func TestLineTableCrLf(t *testing.T) {
	src := []byte("a\r\nb\r\n")
	lt := source.NewLineTable(src)
	if lt.LineCount() != 3 {
		t.Fatalf("LineCount: got %d want 3", lt.LineCount())
	}
	if got := lt.LineText(1); got != "a" {
		t.Fatalf("line1 %q want %q", got, "a")
	}
	if got := lt.LineText(-1); got != "" {
		t.Fatalf("neg %q want empty", got)
	}
}

func TestRange(t *testing.T) {
	src := []byte("ab\ncd\n")
	lt := source.NewLineTable(src)
	r := lt.Range(0, 2)
	if !r.Contains(1) || r.Contains(2) {
		t.Errorf("Contains wrong")
	}
	if (source.Range{}).IsEmpty() != true {
		t.Errorf("empty")
	}
}

func TestSplitLines(t *testing.T) {
	got := source.SplitLines("a\nb\r\nc\n")
	want := []string{"a", "b", "c", ""}
	if len(got) != len(want) || got[0] != "a" || got[1] != "b" || got[2] != "c" || got[3] != "" {
		t.Fatalf("got %v want %v", got, want)
	}
}
