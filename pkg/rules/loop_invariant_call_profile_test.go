package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestLoopInvariantCallStrictProfile(t *testing.T) {
	if _, enabled := rules.Default().EnabledForProfile(lint.ProfileStrict)["loop-invariant-call"]; !enabled {
		t.Fatal("loop invariant call is not enabled by the strict profile")
	}
}
