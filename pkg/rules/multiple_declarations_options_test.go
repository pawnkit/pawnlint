package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestMultipleDeclarationsOptionValidation(t *testing.T) {
	metadata, ok := rules.Default().Lookup("multiple-declarations")
	if !ok || len(metadata.Options) != 2 {
		t.Fatal("multiple declarations options are missing")
	}
	valid := map[string]any{
		"scopes":         []any{"global", "local"},
		"allow-for-loop": false,
	}
	invalid := map[string]any{
		"scopes":         []any{},
		"allow-for-loop": "yes",
	}
	for _, option := range metadata.Options {
		if _, err := lint.NormalizeOption(option, valid[option.Name]); err != nil {
			t.Fatalf("valid %s rejected: %v", option.Name, err)
		}
		if _, err := lint.NormalizeOption(option, invalid[option.Name]); err == nil {
			t.Fatalf("invalid %s accepted", option.Name)
		}
	}
	if metadata.Options[1].Default != true {
		t.Fatal("for-loop declarations are not allowed by default")
	}
}
