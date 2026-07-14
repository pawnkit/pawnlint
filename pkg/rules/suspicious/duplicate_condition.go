package suspicious

import (
	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type DuplicateCondition struct{}

func (DuplicateCondition) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "duplicate-condition",
		Name:            "Duplicate condition",
		Summary:         "Reports repeated pure conditions in an if and else-if chain",
		Explanation:     "A repeated pure condition in an else-if chain can never become true after the first copy was false. Calls and other expressions with side effects are skipped.",
		Category:        diagnostic.CategorySuspicious,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"conditions", "branches", "semantic"},
	}
}

func (DuplicateCondition) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	ctx.Walk.IterKind(parser.KindIfStatement, func(node *parser.Node) {
		parent := ctx.Walk.Parent(node)
		if parent != nil && parent.Kind == parser.KindIfStatement && parent.Field("alternative") == node {
			return
		}
		var seen []*parser.Node
		for current := node; current != nil && current.Kind == parser.KindIfStatement; {
			condition := current.Field("condition")
			if ctx.Semantic.Pure(condition) {
				for _, first := range seen {
					if !ctx.Semantic.Equivalent(first, condition) {
						continue
					}
					ctx.Report(diagnostic.Diagnostic{
						Message:  "condition duplicates an earlier branch",
						Filename: ctx.File.Path,
						Range:    ctx.Walk.Range(condition),
						Notes: []diagnostic.RelatedLocation{{
							Range:   ctx.Walk.Range(first),
							Message: "first condition is here",
						}},
					})
					break
				}
				seen = append(seen, condition)
			}
			next := current.Field("alternative")
			if next == nil || next.Kind != parser.KindIfStatement {
				break
			}
			current = next
		}
	})
}
