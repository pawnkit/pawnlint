package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestCyclomaticComplexityOptionValidation(t *testing.T) {
	registry := rules.Default()
	metadata, ok := registry.Lookup("cyclomatic-complexity")
	if !ok || len(metadata.Options) != 1 {
		t.Fatal("cyclomatic complexity options are missing")
	}
	if _, enabled := registry.EnabledForProfile(lint.ProfileStrict)["cyclomatic-complexity"]; !enabled {
		t.Fatal("cyclomatic complexity is not enabled by the strict profile")
	}
	option := metadata.Options[0]
	if _, err := lint.NormalizeOption(option, 20); err != nil {
		t.Fatalf("valid maximum rejected: %v", err)
	}
	for _, value := range []any{0, 10001, "ten"} {
		if _, err := lint.NormalizeOption(option, value); err == nil {
			t.Fatalf("invalid maximum %v accepted", value)
		}
	}
}
