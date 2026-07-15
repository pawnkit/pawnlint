package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestConstantOverflowRecommendedProfile(t *testing.T) {
	if _, enabled := rules.Default().EnabledForProfile(lint.ProfileRecommended)["constant-overflow"]; !enabled {
		t.Fatal("constant overflow is not enabled by the recommended profile")
	}
}
