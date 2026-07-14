package lint

import (
	"github.com/pawnkit/pawnlint/internal/source"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint/suppress"
)

func (e *Engine) unusedSuppressionDiagnostics(path string, ds []suppress.Directive, used []bool, lt *source.LineTable) []diagnostic.Diagnostic {
	var out []diagnostic.Diagnostic
	for i, d := range ds {
		if d.Kind == suppress.KindEnable {
			continue
		}
		if d.Malformed {
			continue
		}
		if i < len(used) && used[i] {
			continue
		}
		r := lt.Range(d.Offset, d.End)
		out = append(out, diagnostic.Diagnostic{
			RuleID:   SuppressionID,
			Severity: diagnostic.SeverityHint,
			Category: diagnostic.CategoryMaintainability,
			Message:  "unused pawnlint suppression directive",
			Filename: path,
			Range:    r,
		})
	}
	return out
}
