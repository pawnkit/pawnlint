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
	if entry == nil || len(entry.Parsed.Tokens) == 0 {
		t.Fatal("entry tokens were released")
	}
	if include == nil || include.Parsed.Tokens != nil {
		t.Fatal("include tokens were retained")
	}
}
