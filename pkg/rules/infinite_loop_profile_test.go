package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestInfiniteLoopRecommendedProfile(t *testing.T) {
	if _, enabled := rules.Default().EnabledForProfile(lint.ProfileRecommended)["infinite-loop"]; !enabled {
		t.Fatal("infinite loop is not enabled by the recommended profile")
	}
}
