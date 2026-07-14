package correctness

import (
	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type SelfAssignment struct{}

func (SelfAssignment) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "self-assignment",
		Name:            "Self assignment",
		Summary:         "Reports assignments that store a symbol back into itself",
		Explanation:     explanationSelfAssignment,
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  true,
		Fixable:         true,
		Tags:            []string{"assignment", "semantic"},
	}
}

const explanationSelfAssignment = `An assignment such as ` + "`value = value`" + ` does not change the value and is
usually a typo. The rule reports only direct identifiers that resolve to the
same symbol. The safe fix removes the redundant assignment while preserving its
value when it is part of a larger expression.`

func (SelfAssignment) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	ctx.Walk.IterKind(parser.KindAssignmentExpression, func(node *parser.Node) {
		if node.Tok.Kind != token.Assign || ctx.Walk.Uncertain(node) {
			return
		}
		left := node.Field("left")
		right := node.Field("right")
		if left == nil || right == nil || left.Kind != parser.KindIdentifier || right.Kind != parser.KindIdentifier {
			return
		}
		leftSymbol := ctx.Semantic.Resolve(left)
		if leftSymbol == nil || ctx.Semantic.Resolve(right) != leftSymbol {
			return
		}
		if leftSymbol.Kind == semantic.SymbolFunction {
			return
		}
		fixRange := ctx.Walk.Range(node)
		newText := ctx.Walk.Text(right)
		statement := ctx.Walk.Parent(node)
		if statement != nil && statement.Kind == parser.KindExpressionStatement && statement.Field("expression") == node {
			newText = ""
			if block := ctx.Walk.Parent(statement); block != nil && block.Kind == parser.KindBlock {
				fixRange = ctx.Walk.Range(statement)
			}
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  "symbol is assigned to itself",
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(node),
			Fix: &diagnostic.Fix{
				Description: "remove the redundant self-assignment",
				Edits: []diagnostic.Edit{{
					Range:   fixRange,
					NewText: newText,
				}},
			},
		})
	})
}
