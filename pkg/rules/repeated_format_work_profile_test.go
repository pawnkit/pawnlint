package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestRepeatedFormatWorkStrictProfile(t *testing.T) {
	if _, enabled := rules.Default().EnabledForProfile(lint.ProfileStrict)["repeated-format-work"]; !enabled {
		t.Fatal("repeated format work is not enabled by the strict profile")
	}
}
