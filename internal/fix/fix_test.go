package fix_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/pawnkit/pawnlint/internal/fix"
	"github.com/pawnkit/pawnlint/internal/source"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
)

func TestBuildAppliesSortedEditsAndDeduplicates(t *testing.T) {
	sources := map[string][]byte{"test.pwn": []byte("abcdef")}
	edits := []diagnostic.Edit{{Range: offsetRange(4, 6), NewText: "Z"}, {Range: offsetRange(1, 3), NewText: "X"}, {Range: offsetRange(1, 3), NewText: "X"}}
	diagnostics := []diagnostic.Diagnostic{{Filename: "test.pwn", Fix: &diagnostic.Fix{Edits: edits}}}
	plan, err := fix.Build(sources, diagnostics)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Changes) != 1 || string(plan.Changes[0].After) != "aXdZ" {
		t.Fatalf("plan = %#v", plan)
	}
}

func TestBuildRejectsInvalidAndOverlappingEdits(t *testing.T) {
	for _, edits := range [][]diagnostic.Edit{
		{{Range: offsetRange(0, 4)}, {Range: offsetRange(3, 5)}},
		{{Range: offsetRange(-1, 1)}},
		{{Range: offsetRange(0, 7)}},
	} {
		_, err := fix.Build(map[string][]byte{"test.pwn": []byte("abcdef")}, []diagnostic.Diagnostic{{Filename: "test.pwn", Fix: &diagnostic.Fix{Edits: edits}}})
		if err == nil {
			t.Fatalf("edits accepted: %#v", edits)
		}
	}
}

func TestWriteAndDiff(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.pwn")
	if err := os.WriteFile(path, []byte("old\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	plan := fix.Plan{Changes: []fix.Change{{Path: path, Before: []byte("old\n"), After: []byte("new\n")}}}
	diff := fix.Diff(plan)
	if !strings.Contains(diff, "-old") || !strings.Contains(diff, "+new") {
		t.Fatalf("diff = %q", diff)
	}
	if err := fix.Write(plan); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "new\n" {
		t.Fatalf("written = %q, %v", got, err)
	}
	info, err := os.Stat(path)
	if err != nil || runtime.GOOS != "windows" && info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %v, %v", info.Mode().Perm(), err)
	}
}

func TestWriteRejectsStaleSource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.pwn")
	if err := os.WriteFile(path, []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := fix.Plan{Changes: []fix.Change{{Path: path, Before: []byte("old\n"), After: []byte("new\n")}}}
	if err := fix.Write(plan); err == nil {
		t.Fatal("stale source was overwritten")
	}
}

func TestWritePreflightsEveryChange(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.pwn")
	second := filepath.Join(dir, "second.pwn")
	if err := os.WriteFile(first, []byte("first\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := fix.Plan{Changes: []fix.Change{
		{Path: first, Before: []byte("first\n"), After: []byte("fixed first\n")},
		{Path: second, Before: []byte("second\n"), After: []byte("fixed second\n")},
	}}
	if err := fix.Write(plan); err == nil {
		t.Fatal("stale source was accepted")
	}
	got, err := os.ReadFile(first)
	if err != nil || string(got) != "first\n" {
		t.Fatalf("first source changed: %q, %v", got, err)
	}
}

func TestWriteRejectsDuplicateTargets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.pwn")
	if err := os.WriteFile(path, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	change := fix.Change{Path: path, Before: []byte("old\n"), After: []byte("new\n")}
	if err := fix.Write(fix.Plan{Changes: []fix.Change{change, change}}); err == nil {
		t.Fatal("duplicate targets were accepted")
	}
}

func TestDiffMarksMissingFinalNewline(t *testing.T) {
	plan := fix.Plan{Changes: []fix.Change{{Path: "test.pwn", Before: []byte("old"), After: []byte("new")}}}
	diff := fix.Diff(plan)
	if strings.Count(diff, "\\ No newline at end of file") != 2 {
		t.Fatalf("diff = %q", diff)
	}
}

func TestDiffUsesZeroRangeForEmptyFiles(t *testing.T) {
	plan := fix.Plan{Changes: []fix.Change{
		{Path: "created.pwn", After: []byte("new\n")},
		{Path: "cleared.pwn", Before: []byte("old\n")},
	}}
	diff := fix.Diff(plan)
	if !strings.Contains(diff, "@@ -0,0 +1,1 @@") || !strings.Contains(diff, "@@ -1,1 +0,0 @@") {
		t.Fatalf("diff = %q", diff)
	}
}

func offsetRange(start, end int) source.Range {
	return source.Range{Start: source.Position{Offset: start}, End: source.Position{Offset: end}}
}
