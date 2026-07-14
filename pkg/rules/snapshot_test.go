package rules_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pawnkit/pawnlint/internal/ruletest"
)

func fixtureRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		cand := filepath.Join(dir, "testdata", "rules")
		if st, err := os.Stat(cand); err == nil && st.IsDir() {
			return cand
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("testdata/rules not found from %s", dir)
	return ""
}

func TestRuleSnapshots(t *testing.T) {
	root := fixtureRoot(t)
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read testdata/rules: %v", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		ruleID := e.Name()
		t.Run(ruleID, func(t *testing.T) {
			ruletest.RunSnapshot(t, ruleID, filepath.Join(root, ruleID))
		})
	}
}
