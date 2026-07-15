package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestTooManyParametersOptionValidation(t *testing.T) {
	registry := rules.Default()
	metadata, ok := registry.Lookup("too-many-parameters")
	if !ok || len(metadata.Options) != 4 {
		t.Fatal("too many parameters options are missing")
	}
	if _, enabled := registry.EnabledForProfile(lint.ProfileStrict)["too-many-parameters"]; !enabled {
		t.Fatal("too many parameters is not enabled by the strict profile")
	}
	valid := map[string]any{
		"maximum":           7,
		"include-public":    true,
		"include-callbacks": true,
		"exclude":           []any{"^Generated_"},
	}
	invalid := map[string]any{
		"maximum":           0,
		"include-public":    "yes",
		"include-callbacks": 1,
		"exclude":           []any{"["},
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
