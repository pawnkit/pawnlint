package correctness

import (
	"strings"

	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/lint/suppress"
)

type UnknownSuppression struct{}

func (UnknownSuppression) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "unknown-suppression",
		Name:            "Unknown suppression",
		Summary:         "Reports unknown, malformed, or unused pawnlint suppression directives",
		Explanation:     explanationUnknownSuppression,
		Category:        diagnostic.CategoryMaintainability,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SyntaxAnalysis,
		DefaultEnabled:  true,
		Fixable:         false,
		Tags:            []string{"suppression", "tooling"},
	}
}

const explanationUnknownSuppression = `Reports suppression comments that are malformed, name unknown rules, have no
matching disable, or suppress no finding.

These directives can be removed safely because they do not affect reported
diagnostics. Parser errors cannot be suppressed.`

func (UnknownSuppression) Run(ctx *lint.Context) {
	m := ctx.Walk
	known := ctx.Known
	if known == nil {
		known = map[string]struct{}{}
	}
	disableStack := make(map[string]int)
	allDepth := 0
	for _, d := range ctx.Supp {
		r := m.LineTable.Range(d.Offset, d.Offset)
		switch d.Kind {
		case suppress.KindDisableNextLine, suppress.KindDisable:
			if d.Malformed {
				ctx.Report(diagnostic.Diagnostic{
					RuleID:   "unknown-suppression",
					Message:  "malformed suppression: no rule id given",
					Filename: ctx.File.Path,
					Range:    r,
				})
			}
			if !d.All {
				for _, id := range d.IDs {
					if _, ok := known[id]; !ok {
						ctx.Report(diagnostic.Diagnostic{
							RuleID:   "unknown-suppression",
							Message:  "unknown suppression rule " + quote(id),
							Filename: ctx.File.Path,
							Range:    r,
						})
					}
				}
			}
			if d.Kind == suppress.KindDisable {
				if d.All {
					allDepth++
				}
				for _, id := range d.IDs {
					disableStack[id]++
				}
			}
		case suppress.KindEnable:
			if d.All {
				if allDepth <= 0 {
					ctx.Report(diagnostic.Diagnostic{
						RuleID:   "unknown-suppression",
						Message:  "unmatched `pawnlint-enable` for \"all\"",
						Filename: ctx.File.Path,
						Range:    r,
					})
				} else {
					allDepth--
				}
			}
			for _, id := range d.IDs {
				if disableStack[id] <= 0 {
					ctx.Report(diagnostic.Diagnostic{
						RuleID:   "unknown-suppression",
						Message:  "unmatched `pawnlint-enable` for " + quote(id),
						Filename: ctx.File.Path,
						Range:    r,
					})
				} else {
					disableStack[id]--
				}
			}
		}
	}
}

func quote(s string) string { return "\"" + strings.ReplaceAll(s, "\"", "\\\"") + "\"" }
