package project_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pawnkit/pawnlint/pkg/project"
)

func TestReleaseIncludesDropsOnlyIncludeTokens(t *testing.T) {
	dir := t.TempDir()
	includePath := filepath.Join(dir, "shared.inc")
	if err := os.WriteFile(includePath, []byte("stock Shared() { return 1; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	entryPath := filepath.Join(dir, "main.pwn")
	source := []byte("#include \"shared.inc\"\nmain() { return Shared(); }\n")
	model, err := project.Build([]project.Source{{Path: entryPath, Content: source}}, project.Options{
		WorkingDir: dir, DefinesComplete: true, ReleaseExpanded: true, ReleaseIncludes: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	entry := model.File(entryPath)
	include := model.File(includePath)
	if entry == nil || entry.Parsed != nil || entry.CompactParsed == nil {
		t.Fatal("entry was not stored compactly")
	}
	if include == nil || include.Parsed != nil || include.CompactParsed == nil {
		t.Fatal("include was not stored compactly")
	}
}

func TestCompactIncludeMaterializesPointerSyntaxOnDemand(t *testing.T) {
	dir := t.TempDir()
	includePath := filepath.Join(dir, "shared.inc")
	if err := os.WriteFile(includePath, []byte("stock Shared() { return 1; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	entryPath := filepath.Join(dir, "main.pwn")
	source := []byte("#include \"shared.inc\"\nmain() { return Shared(); }\n")
	model, err := project.Build([]project.Source{{Path: entryPath, Content: source}}, project.Options{
		WorkingDir: dir, DefinesComplete: true, ReleaseExpanded: true, ReleaseIncludes: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	declarations := model.Declarations["Shared"]
	if len(declarations) != 1 || declarations[0].Node != nil || declarations[0].PointerNode() == nil {
		t.Fatal("compact declaration was not materialized")
	}
	include := model.File(includePath)
	if include == nil || include.Walk == nil || include.Semantic == nil {
		t.Fatal("pointer models were not materialized")
	}
}
