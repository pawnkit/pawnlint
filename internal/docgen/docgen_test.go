package docgen_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pawnkit/pawnlint/internal/docgen"
)

func TestGenerateDeterministic(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	if err := docgen.Generate(dir1); err != nil {
		t.Fatal(err)
	}
	if err := docgen.Generate(dir2); err != nil {
		t.Fatal(err)
	}
	if err := equalTrees(dir1, dir2); err != nil {
		t.Fatalf("non-deterministic docs: %v", err)
	}
}

func TestGeneratedMatchesCommitted(t *testing.T) {
	dir := t.TempDir()
	if err := docgen.Generate(dir); err != nil {
		t.Fatal(err)
	}
	committed := filepath.Join("..", "..", "docs", "rules")
	if err := equalTrees(committed, dir); err != nil {
		t.Fatalf("generated rule docs are stale: %v", err)
	}
}

func equalTrees(a, b string) error {
	ea, err := os.ReadDir(a)
	if err != nil {
		return err
	}
	eb, err := os.ReadDir(b)
	if err != nil {
		return err
	}
	if len(ea) != len(eb) {
		return fileErr{path: b, msg: "file count differs"}
	}
	for i, e := range ea {
		if e.Name() != eb[i].Name() || e.IsDir() != eb[i].IsDir() {
			return fileErr{path: e.Name(), msg: "directory entries differ"}
		}
		if e.IsDir() {
			continue
		}
		ca, err := os.ReadFile(filepath.Join(a, e.Name()))
		if err != nil {
			return err
		}
		cb, err := os.ReadFile(filepath.Join(b, e.Name()))
		if err != nil {
			return err
		}
		if string(ca) != string(cb) {
			return fileErr{path: e.Name(), msg: "differs"}
		}
	}
	return nil
}

type fileErr struct{ path, msg string }

func (e fileErr) Error() string { return e.path + ": " + e.msg }
