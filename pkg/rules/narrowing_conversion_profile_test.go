package rules_test

import (
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestNarrowingConversionStrictProfile(t *testing.T) {
	if _, enabled := rules.Default().EnabledForProfile(lint.ProfileStrict)["narrowing-conversion"]; !enabled {
		t.Fatal("narrowing conversion is not enabled by the strict profile")
	}
}
