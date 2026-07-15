package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestMacroRepeatedParameterStrictProfile(t *testing.T) {
	if _, enabled := rules.Default().EnabledForProfile(lint.ProfileStrict)["macro-repeated-parameter"]; !enabled {
		t.Fatal("macro repeated parameter is not enabled by the strict profile")
	}
}
