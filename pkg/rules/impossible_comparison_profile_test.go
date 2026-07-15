package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestImpossibleComparisonRecommendedProfile(t *testing.T) {
	if _, enabled := rules.Default().EnabledForProfile(lint.ProfileRecommended)["impossible-comparison"]; !enabled {
		t.Fatal("impossible comparison is not enabled by the recommended profile")
	}
}
