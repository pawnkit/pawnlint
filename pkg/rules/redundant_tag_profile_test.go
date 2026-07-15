package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestRedundantTagStrictProfile(t *testing.T) {
	registry := rules.Default()
	if _, enabled := registry.EnabledForProfile(lint.ProfileStrict)["redundant-tag"]; !enabled {
		t.Fatal("redundant tag is not enabled by the strict profile")
	}
}
