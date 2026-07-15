package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestPreferConstStrictProfile(t *testing.T) {
	registry := rules.Default()
	if _, enabled := registry.EnabledForProfile(lint.ProfileStrict)["prefer-const"]; !enabled {
		t.Fatal("prefer const is not enabled by the strict profile")
	}
}
