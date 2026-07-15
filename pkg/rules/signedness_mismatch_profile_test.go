package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestSignednessMismatchStrictProfile(t *testing.T) {
	if _, enabled := rules.Default().EnabledForProfile(lint.ProfileStrict)["signedness-mismatch"]; !enabled {
		t.Fatal("signedness mismatch is not enabled by the strict profile")
	}
}
