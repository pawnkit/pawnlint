package project_test

import (
	"bytes"
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
	if entry == nil || entry.Parsed == nil || len(entry.Parsed.Tokens) == 0 {
		t.Fatal("entry tokens were released")
	}
	if include == nil || include.Parsed != nil || include.CompactParsed == nil {
		t.Fatal("include was not stored compactly")
	}
}

func TestReleaseIncludesStoresLargeTargetCompactly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	source := append(bytes.Repeat([]byte{' '}, 600<<10), []byte("\nmain() {}\n")...)
	model, err := project.Build([]project.Source{{Path: path, Content: source}}, project.Options{
		WorkingDir: dir, DefinesComplete: true, ReleaseExpanded: true, ReleaseIncludes: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	file := model.File(path)
	if file == nil || file.Parsed != nil || file.CompactParsed == nil {
		t.Fatal("large target was not stored compactly")
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

func TestPointerTriviaFollowsFeatures(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	source := []byte("// documentation\nmain() {}\n")
	for _, test := range []struct {
		features project.Features
		retained bool
	}{
		{features: project.Features(0)},
		{features: project.NewFeatures(project.FeatureRuntimeCalls)},
		{features: project.NewFeatures(project.FeatureTrivia), retained: true},
	} {
		model, err := project.Build([]project.Source{{Path: path, Content: source}}, project.Options{WorkingDir: dir, Features: &test.features})
		if err != nil {
			t.Fatal(err)
		}
		file := model.File(path)
		retained := false
		for _, current := range file.Parsed.Tokens {
			retained = retained || len(current.LeadingTrivia) != 0 || len(current.TrailingTrivia) != 0
		}
		if retained != test.retained {
			t.Fatalf("retained = %t, want %t", retained, test.retained)
		}
	}
}

func TestCompactRuntimeCallsDiscardTrivia(t *testing.T) {
	dir := t.TempDir()
	includePath := filepath.Join(dir, "timer.inc")
	if err := os.WriteFile(includePath, []byte("// documentation\n#define TIMER_CALLBACK \"Tick\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	entryPath := filepath.Join(dir, "main.pwn")
	source := []byte("#include \"timer.inc\"\nmain() { SetTimer(TIMER_CALLBACK, 1000, false); }\npublic Tick() {}\n")
	features := project.NewFeatures(project.FeatureRuntimeCalls)
	model, err := project.Build([]project.Source{{Path: entryPath, Content: source}}, project.Options{
		WorkingDir: dir, DefinesComplete: true, ReleaseExpanded: true, ReleaseIncludes: true, Features: &features,
	})
	if err != nil {
		t.Fatal(err)
	}
	include := model.File(includePath)
	if include == nil || include.CompactParsed == nil || len(include.CompactParsed.Trivia) != 0 {
		t.Fatal("compact runtime syntax retained trivia")
	}
	if len(model.CallGraph.AsyncCalls) != 1 || model.CallGraph.AsyncCalls[0].Callee.Name != "Tick" {
		t.Fatalf("async calls = %#v", model.CallGraph.AsyncCalls)
	}
}

func TestPointerTriviaRetainsSuppressions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	features := project.Features(0)
	model, err := project.Build([]project.Source{{Path: path, Content: []byte("// pawnlint-disable-next-line discarded-expression\nvalue + 1;\n")}}, project.Options{WorkingDir: dir, Features: &features})
	if err != nil {
		t.Fatal(err)
	}
	for _, current := range model.File(path).Parsed.Tokens {
		if len(current.LeadingTrivia) != 0 || len(current.TrailingTrivia) != 0 {
			return
		}
	}
	t.Fatal("suppression trivia was discarded")
}
