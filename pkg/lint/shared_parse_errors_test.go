package lint

import (
	"path/filepath"
	"testing"

	"github.com/pawnkit/pawn-analysis"
	coresource "github.com/pawnkit/pawnkit-core/source"
	"github.com/pawnkit/pawnlint/internal/source"
)

func TestSharedParseErrorsUseTheMatchingAnalysis(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.pwn")
	text := []byte("main() { return 1 + ; }\n")
	lines := source.NewLineTable(text)
	result := analysis.Analyze(text, analysis.Options{URI: coresource.FileURI(path)})
	findings, ok := sharedParseErrors(path, text, lines, result)
	if !ok || len(findings) == 0 || findings[0].RuleID != ParseErrorID {
		t.Fatalf("shared parse findings = %#v, accepted = %v", findings, ok)
	}
	if _, ok := sharedParseErrors(path, []byte("main() {}\n"), lines, result); ok {
		t.Fatal("accepted a stale revision")
	}
	if _, ok := sharedParseErrors(filepath.Join(filepath.Dir(path), "other.pwn"), text, lines, result); ok {
		t.Fatal("accepted a different file")
	}
	result.Diagnostics = nil // The shared analyzer has reconciled the raw findings.
	findings, ok = sharedParseErrors(path, text, lines, result)
	if !ok || len(findings) != 0 {
		t.Fatalf("reconciled parse findings = %#v, accepted = %v", findings, ok)
	}
}
