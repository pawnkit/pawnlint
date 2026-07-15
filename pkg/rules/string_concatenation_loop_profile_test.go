package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestStringConcatenationLoopStrictProfile(t *testing.T) {
	if _, enabled := rules.Default().EnabledForProfile(lint.ProfileStrict)["string-concatenation-loop"]; !enabled {
		t.Fatal("string concatenation loop is not enabled by the strict profile")
	}
}
