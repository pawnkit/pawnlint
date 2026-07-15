package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestRedundantElseStrictProfile(t *testing.T) {
	registry := rules.Default()
	if _, enabled := registry.EnabledForProfile(lint.ProfileStrict)["redundant-else"]; !enabled {
		t.Fatal("redundant else is not enabled by the strict profile")
	}
}
