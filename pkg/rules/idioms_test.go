package rules_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pawnkit/pawnlint/internal/ruletest"
)

func TestEcosystemIdioms(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("..", "..", "testdata", "idioms", "ecosystem.pwn"))
	if err != nil {
		t.Fatal(err)
	}
	ruleIDs := []string{
		"non-public-callback",
		"unused-parameter",
		"discarded-expression",
		"unreleased-resource-handle",
		"read-after-release",
		"dead-write",
		"negative-or-zero-array-size",
	}
	for _, ruleID := range ruleIDs {
		t.Run(ruleID, func(t *testing.T) {
			diags := ruletest.RunRule(t, "ecosystem.pwn", ruleID, src)
			for _, d := range diags {
				t.Errorf("unexpected finding on idiomatic code: %s:%d: %s", ruleID, d.Range.Start.Line, d.Message)
			}
		})
	}
}
