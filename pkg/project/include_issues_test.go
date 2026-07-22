package project_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pawnkit/pawnlint/pkg/project"
)

func TestIncludeIssues(t *testing.T) {
	dir := t.TempDir()
	includeDir := filepath.Join(dir, "includes")
	if err := os.Mkdir(includeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sourceDir := filepath.Join(dir, "src")
	if err := os.Mkdir(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	local := filepath.Join(sourceDir, "shared.inc")
	shadowed := filepath.Join(includeDir, "shared.inc")
	if err := os.WriteFile(local, []byte("stock Local() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(shadowed, []byte("stock Shadowed() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rootPath := filepath.Join(sourceDir, "main.pwn")
	source := []byte("#include <shared>\n#include \"missing.inc\"\n#tryinclude \"optional.inc\"\nmain() {}\n")
	model, err := project.Build([]project.Source{{Path: rootPath, Content: source}}, project.Options{WorkingDir: dir, IncludePaths: []string{includeDir}})
	if err != nil {
		t.Fatal(err)
	}
	missing := model.MissingIncludes()
	if len(missing) != 1 || missing[0].Include.Path != "missing.inc" {
		t.Fatalf("missing includes = %+v", missing)
	}
	ambiguous := model.AmbiguousIncludes()
	if len(ambiguous) != 0 {
		t.Fatalf("ambiguous includes = %+v", ambiguous)
	}
	resolved := model.File(rootPath).Includes[0].Resolved
	if resolved == nil || resolved.Path != shadowed {
		t.Fatalf("angle include resolved to %+v, want %q", resolved, shadowed)
	}
}

func TestQuotedIncludePrefersLocalFile(t *testing.T) {
	dir := t.TempDir()
	includeDir := filepath.Join(dir, "includes")
	if err := os.Mkdir(includeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sourceDir := filepath.Join(dir, "src")
	if err := os.Mkdir(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	local := filepath.Join(sourceDir, "shared.inc")
	shadowed := filepath.Join(includeDir, "shared.inc")
	for _, path := range []string{local, shadowed} {
		if err := os.WriteFile(path, []byte("stock Shared() {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	rootPath := filepath.Join(sourceDir, "main.pwn")
	model, err := project.Build([]project.Source{{Path: rootPath, Content: []byte("#include \"shared\"\n")}}, project.Options{
		WorkingDir: dir, IncludePaths: []string{includeDir},
	})
	if err != nil {
		t.Fatal(err)
	}
	issues := model.AmbiguousIncludes()
	if len(issues) != 1 || issues[0].Include.Candidates[0] != local || issues[0].Include.Candidates[1] != shadowed {
		t.Fatalf("ambiguous includes = %+v", issues)
	}
}

func TestIncludeIssuesDeduplicateSharedFiles(t *testing.T) {
	dir := t.TempDir()
	shared := filepath.Join(dir, "shared.inc")
	if err := os.WriteFile(shared, []byte("#include \"missing.inc\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	one := filepath.Join(dir, "one.pwn")
	two := filepath.Join(dir, "two.pwn")
	source := []byte("#include \"shared.inc\"\n")
	model, err := project.Build([]project.Source{{Path: one, Content: source}, {Path: two, Content: source}}, project.Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	if issues := model.MissingIncludes(); len(issues) != 1 || issues[0].Owner.Path != one {
		t.Fatalf("missing includes = %+v", issues)
	}
}

func TestDuplicateIncludes(t *testing.T) {
	dir := t.TempDir()
	shared := filepath.Join(dir, "shared.inc")
	if err := os.WriteFile(shared, []byte("stock Shared() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rootPath := filepath.Join(dir, "main.pwn")
	source := []byte("#include \"shared\"\n#include \"shared.inc\"\nmain() {}\n")
	model, err := project.Build([]project.Source{{Path: rootPath, Content: source}}, project.Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	issues := model.DuplicateIncludes()
	if len(issues) != 1 || issues[0].Include.Node.Start != len("#include \"shared\"\n") {
		t.Fatalf("duplicate includes = %+v", issues)
	}
}
