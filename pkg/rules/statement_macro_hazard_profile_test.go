package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestStatementMacroHazardStrictProfile(t *testing.T) {
	if _, enabled := rules.Default().EnabledForProfile(lint.ProfileStrict)["statement-macro-hazard"]; !enabled {
		t.Fatal("statement macro hazard is not enabled by the strict profile")
	}
}
