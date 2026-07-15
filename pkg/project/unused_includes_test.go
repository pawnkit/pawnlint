package project_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pawnkit/pawnlint/pkg/project"
)

func TestUnusedIncludes(t *testing.T) {
	dir := t.TempDir()
	unused := filepath.Join(dir, "unused.inc")
	used := filepath.Join(dir, "used.inc")
	macro := filepath.Join(dir, "macro.inc")
	files := map[string]string{
		unused: "stock Unused() {}\n",
		used:   "stock Used() {}\n",
		macro:  "#define VALUE (1)\n",
	}
	for path, source := range files {
		if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	root := filepath.Join(dir, "main.pwn")
	source := []byte("#include \"unused.inc\"\n#include \"used.inc\"\n#include \"macro.inc\"\nmain() { Used(); }\n")
	model, err := project.Build([]project.Source{{Path: root, Content: source}}, project.Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	issues := model.UnusedIncludes()
	if len(issues) != 1 || issues[0].Include.Path != "unused.inc" {
		t.Fatalf("unused includes = %+v", issues)
	}
}

func TestUnusedIncludesSkipSharedDependency(t *testing.T) {
	dir := t.TempDir()
	shared := filepath.Join(dir, "shared.inc")
	wrapper := filepath.Join(dir, "wrapper.inc")
	if err := os.WriteFile(shared, []byte("stock Shared() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(wrapper, []byte("#include \"shared.inc\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(dir, "main.pwn")
	source := []byte("#include \"shared.inc\"\n#include \"wrapper.inc\"\nmain() {}\n")
	model, err := project.Build([]project.Source{{Path: root, Content: source}}, project.Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, issue := range model.UnusedIncludes() {
		if issue.Include.Path == "shared.inc" {
			t.Fatalf("shared dependency reported as unused: %+v", issue)
		}
	}
}
