package project_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pawnkit/pawnlint/pkg/project"
)

func TestIncludeCycles(t *testing.T) {
	dir := t.TempDir()
	aPath := filepath.Join(dir, "a.inc")
	bPath := filepath.Join(dir, "b.inc")
	if err := os.WriteFile(aPath, []byte("#include \"b.inc\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bPath, []byte("#include \"a.inc\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rootPath := filepath.Join(dir, "main.pwn")
	model, err := project.Build([]project.Source{{Path: rootPath, Content: []byte("#include \"a.inc\"\nmain() {}\n")}}, project.Options{WorkingDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	cycles := model.IncludeCycles()
	if len(cycles) != 1 || len(cycles[0].Edges) != 2 {
		t.Fatalf("cycles = %+v", cycles)
	}
	if filepath.Base(cycles[0].Edges[0].From.Path) != "a.inc" || filepath.Base(cycles[0].Edges[1].From.Path) != "b.inc" {
		t.Fatalf("cycle edges = %+v", cycles[0].Edges)
	}
}

func TestIncludeCyclesSkipUncertainEdges(t *testing.T) {
	dir := t.TempDir()
	rootPath := filepath.Join(dir, "main.pwn")
	source := []byte("#if UNKNOWN\n#include \"main.pwn\"\n#endif\nmain() {}\n")
	model, err := project.Build([]project.Source{{Path: rootPath, Content: source}}, project.Options{WorkingDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if cycles := model.IncludeCycles(); len(cycles) != 0 {
		t.Fatalf("cycles = %+v", cycles)
	}
}

func TestIncludeCyclesFindSelfInclude(t *testing.T) {
	dir := t.TempDir()
	rootPath := filepath.Join(dir, "main.pwn")
	source := []byte("#include \"main.pwn\"\nmain() {}\n")
	if err := os.WriteFile(rootPath, source, 0o644); err != nil {
		t.Fatal(err)
	}
	model, err := project.Build([]project.Source{{Path: rootPath, Content: source}}, project.Options{WorkingDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	cycles := model.IncludeCycles()
	if len(cycles) != 1 || len(cycles[0].Edges) != 1 {
		t.Fatalf("cycles = %+v", cycles)
	}
}
