package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestArgumentTagMismatchRecommendedProfile(t *testing.T) {
	if _, enabled := rules.Default().EnabledForProfile(lint.ProfileRecommended)["argument-tag-mismatch"]; !enabled {
		t.Fatal("argument tag mismatch is not enabled by the recommended profile")
	}
}
