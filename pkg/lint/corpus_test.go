package lint_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestSharedTagDiagnosticsMatchCorpus(t *testing.T) {
	root := lintCorpusRoot()
	if root == "" {
		t.Skip("pawn-corpus is unavailable")
	}
	paths := []string{
		filepath.Join(root, "semantics", "compiler_tag_mismatch.pwn"),
		filepath.Join(root, "semantics", "compiler_tag_union.pwn"),
	}
	engine := lint.NewEngine(rules.Default())
	for _, path := range paths {
		t.Run(strings.TrimSuffix(filepath.Base(path), ".pwn"), func(t *testing.T) {
			text, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			diagnostics := engine.LintFile(path, text, lint.ProjectAnalysis, nil, nil, nil)
			for _, item := range diagnostics {
				if item.RuleID == "pawn-analysis:sema/undefined-symbol" {
					t.Fatalf("tag reported as undefined: %+v", item)
				}
			}
		})
	}
}

func lintCorpusRoot() string {
	if root := os.Getenv("PAWN_CORPUS_DIR"); root != "" {
		if info, err := os.Stat(root); err == nil && info.IsDir() {
			return root
		}
		return ""
	}
	root := filepath.Join("..", "..", "..", "pawn-corpus")
	if info, err := os.Stat(root); err == nil && info.IsDir() {
		return root
	}
	return ""
}
