package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestTodoPolicyOptionValidation(t *testing.T) {
	metadata, ok := rules.Default().Lookup("todo-policy")
	if !ok || len(metadata.Options) != 7 {
		t.Fatal("TODO policy options are missing")
	}
	valid := map[string]any{
		"tags":             []any{"TODO", "FIXME"},
		"allowed-owners":   []any{"alice", "team-core"},
		"require-owner":    true,
		"require-date":     true,
		"require-issue":    true,
		"issue-pattern":    `[A-Z]+-[0-9]+`,
		"maximum-age-days": 30,
	}
	invalid := map[string]any{
		"tags":             []any{},
		"allowed-owners":   []any{"bad owner"},
		"require-owner":    "yes",
		"require-date":     1,
		"require-issue":    []any{},
		"issue-pattern":    "[",
		"maximum-age-days": -1,
	}
	for _, option := range metadata.Options {
		if _, err := lint.NormalizeOption(option, valid[option.Name]); err != nil {
			t.Fatalf("valid %s rejected: %v", option.Name, err)
		}
		if _, err := lint.NormalizeOption(option, invalid[option.Name]); err == nil {
			t.Fatalf("invalid %s accepted", option.Name)
		}
	}
}
