package rules_test

import (
	"os"
	"path/filepath"
	"testing"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/fix"
	"github.com/pawnkit/pawnlint/internal/ruletest"
)

func TestRedundantElseFixes(t *testing.T) {
	path := filepath.Join(fixtureRoot(t), "redundant-else", "invalid.pwn")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	diagnostics := ruletest.RunRule(t, path, "redundant-else", source)
	plan, err := fix.Build(map[string][]byte{path: source}, diagnostics)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Changes) != 1 {
		t.Fatalf("fix changes = %d", len(plan.Changes))
	}
	updated := plan.Changes[0].After
	parsed := parser.Parse(updated)
	if parsed.Broken || parsed.Root.HasError {
		t.Fatal("fixed source is malformed")
	}
	if remaining := ruletest.RunRule(t, path, "redundant-else", updated); len(remaining) != 0 {
		t.Fatalf("remaining diagnostics = %d", len(remaining))
	}
}
