package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestMaximumNestingOptionValidation(t *testing.T) {
	registry := rules.Default()
	metadata, ok := registry.Lookup("maximum-nesting")
	if !ok || len(metadata.Options) != 1 {
		t.Fatal("maximum nesting options are missing")
	}
	if _, enabled := registry.EnabledForProfile(lint.ProfileStrict)["maximum-nesting"]; !enabled {
		t.Fatal("maximum nesting is not enabled by the strict profile")
	}
	option := metadata.Options[0]
	if _, err := lint.NormalizeOption(option, 5); err != nil {
		t.Fatalf("valid maximum rejected: %v", err)
	}
	for _, value := range []any{0, 1001, "four"} {
		if _, err := lint.NormalizeOption(option, value); err == nil {
			t.Fatalf("invalid maximum %v accepted", value)
		}
	}
}
