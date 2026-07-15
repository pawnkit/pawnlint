package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestRedundantForwardStrictProfile(t *testing.T) {
	registry := rules.Default()
	if _, enabled := registry.EnabledForProfile(lint.ProfileStrict)["redundant-forward"]; !enabled {
		t.Fatal("redundant forward is not enabled by the strict profile")
	}
}
