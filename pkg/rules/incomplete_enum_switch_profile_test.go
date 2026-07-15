package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestIncompleteEnumSwitchStrictProfile(t *testing.T) {
	if _, enabled := rules.Default().EnabledForProfile(lint.ProfileStrict)["incomplete-enum-switch"]; !enabled {
		t.Fatal("incomplete enum switch is not enabled by the strict profile")
	}
}
