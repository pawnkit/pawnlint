package cache_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pawnkit/pawnlint/internal/cache"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
)

func TestKeyInvalidation(t *testing.T) {
	base := cache.KeyInput{
		Context: "main",
		Config:  map[string]any{"profile": "recommended"},
		API:     map[string]any{"target": "openmp"},
		Sources: []cache.Source{{Path: "main.pwn", Content: []byte("main() {}\n")}},
	}
	first, err := cache.Key(base)
	if err != nil {
		t.Fatal(err)
	}
	same := base
	same.Sources = []cache.Source{{Path: "main.pwn", Content: []byte("main() {}\n")}}
	second, err := cache.Key(same)
	if err != nil || second != first {
		t.Fatalf("same key = %q, err = %v", second, err)
	}
	changes := []cache.KeyInput{base, base, base, base}
	changes[0].Context = "other"
	changes[1].Config = map[string]any{"profile": "strict"}
	changes[2].API = map[string]any{"target": "samp"}
	changes[3].Sources = []cache.Source{{Path: "main.pwn", Content: []byte("main() { return; }\n")}}
	for index, changed := range changes {
		key, err := cache.Key(changed)
		if err != nil || key == first {
			t.Errorf("change %d key = %q, err = %v", index, key, err)
		}
	}
}

func TestWriteLoadAndReplace(t *testing.T) {
	dir := t.TempDir()
	slot := cache.Slot("main")
	diagnostics := []diagnostic.Diagnostic{{RuleID: "rule", Message: "message"}}
	if err := cache.Write(dir, slot, "first", diagnostics); err != nil {
		t.Fatal(err)
	}
	loaded, hit := cache.Load(dir, slot, "first")
	if !hit || len(loaded) != 1 || loaded[0].RuleID != "rule" {
		t.Fatalf("loaded = %+v, hit = %v", loaded, hit)
	}
	if _, hit := cache.Load(dir, slot, "other"); hit {
		t.Fatal("mismatched key hit")
	}
	if err := cache.Write(dir, slot, "second", nil); err != nil {
		t.Fatal(err)
	}
	if loaded, hit := cache.Load(dir, slot, "second"); !hit || len(loaded) != 0 {
		t.Fatalf("loaded = %+v, hit = %v", loaded, hit)
	}
}

func TestCorruptEntryMisses(t *testing.T) {
	dir := t.TempDir()
	slot := cache.Slot("main")
	if err := os.WriteFile(filepath.Join(dir, slot+".json"), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, hit := cache.Load(dir, slot, "key"); hit {
		t.Fatal("corrupt entry hit")
	}
}

func TestValidateDiagnostics(t *testing.T) {
	sources := []cache.Source{{Path: "main.pwn", Content: []byte("main() {}\n")}}
	valid := []diagnostic.Diagnostic{{
		RuleID:   "rule",
		Severity: diagnostic.SeverityWarning,
		Category: diagnostic.CategoryCorrectness,
		Filename: "main.pwn",
	}}
	if !cache.Validate(valid, sources) {
		t.Fatal("valid diagnostics rejected")
	}
	invalidPath := append([]diagnostic.Diagnostic(nil), valid...)
	invalidPath[0].Filename = "other.pwn"
	if cache.Validate(invalidPath, sources) {
		t.Fatal("unknown path accepted")
	}
	invalidRange := append([]diagnostic.Diagnostic(nil), valid...)
	invalidRange[0].Range.End.Offset = len(sources[0].Content) + 1
	if cache.Validate(invalidRange, sources) {
		t.Fatal("out-of-bounds range accepted")
	}
}
