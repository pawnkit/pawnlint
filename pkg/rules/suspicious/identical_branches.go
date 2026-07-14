package suspicious

import (
	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type IdenticalBranches struct{}

func (IdenticalBranches) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "identical-branches",
		Name:            "Identical branches",
		Summary:         "Reports if and ternary branches with identical code",
		Explanation:     "Identical alternatives make the condition ineffective and often indicate a copy-and-paste mistake. Branches must have the same parsed tokens; whitespace and comments are ignored.",
		Category:        diagnostic.CategorySuspicious,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"branches", "conditionals", "semantic"},
	}
}

func (IdenticalBranches) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	ctx.Walk.IterKind(parser.KindIfStatement, func(node *parser.Node) {
		left := node.Field("consequence")
		right := node.Field("alternative")
		if right == nil || !ctx.Semantic.EquivalentSyntax(left, right) {
			return
		}
		reportIdenticalBranches(ctx, left, right)
	})
	ctx.Walk.IterKind(parser.KindTernaryExpression, func(node *parser.Node) {
		left := node.Field("consequence")
		right := node.Field("alternative")
		if !ctx.Semantic.EquivalentSyntax(left, right) {
			return
		}
		reportIdenticalBranches(ctx, left, right)
	})
}

func reportIdenticalBranches(ctx *lint.Context, first, second *parser.Node) {
	ctx.Report(diagnostic.Diagnostic{
		Message:  "conditional branches are identical",
		Filename: ctx.File.Path,
		Range:    ctx.Walk.Range(second),
		Notes: []diagnostic.RelatedLocation{{
			Range:   ctx.Walk.Range(first),
			Message: "matching branch is here",
		}},
	})
}
