package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestInvariantLoopConditionRecommendedProfile(t *testing.T) {
	if _, enabled := rules.Default().EnabledForProfile(lint.ProfileRecommended)["invariant-loop-condition"]; !enabled {
		t.Fatal("invariant loop condition is not enabled by the recommended profile")
	}
}
