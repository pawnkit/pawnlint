package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestUnconditionalRecursionRecommendedProfile(t *testing.T) {
	if _, enabled := rules.Default().EnabledForProfile(lint.ProfileRecommended)["unconditional-recursion"]; !enabled {
		t.Fatal("unconditional recursion is not enabled by the recommended profile")
	}
}
