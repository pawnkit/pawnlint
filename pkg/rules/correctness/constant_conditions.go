package correctness

import (
	"fmt"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type ConstantCondition struct{}

func (ConstantCondition) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "constant-condition",
		Name:            "Constant condition",
		Summary:         "Reports if and ternary conditions with a constant result",
		Explanation:     "A constant condition always selects the same branch. Loops are skipped because constant loop conditions are often intentional.",
		Category:        diagnostic.CategorySuspicious,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.ControlFlowAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"constants", "conditions", "control-flow"},
	}
}

func (ConstantCondition) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	ctx.Walk.Iter(func(node *parser.Node) {
		if node.Kind != parser.KindIfStatement && node.Kind != parser.KindTernaryExpression {
			return
		}
		condition := node.Field("condition")
		value, ok := ctx.Eval(condition)
		if !ok {
			return
		}
		result := "false"
		if value != 0 {
			result = "true"
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  "condition is always " + result,
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(condition),
		})
	})
}

type DuplicateSwitchCase struct{}

func (DuplicateSwitchCase) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "duplicate-switch-case",
		Name:            "Duplicate switch case",
		Summary:         "Reports repeated constant values in one switch statement",
		Explanation:     "Two case values with the same constant can never select different branches. Case ranges are skipped until range overlap analysis is available.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityError,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  true,
		Fixable:         false,
		Tags:            []string{"constants", "switch", "semantic"},
	}
}

func (DuplicateSwitchCase) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	ctx.Walk.IterKind(parser.KindSwitchStatement, func(node *parser.Node) {
		seen := make(map[int64]*parser.Node)
		for _, clause := range node.Children {
			if clause.Kind != parser.KindCaseClause || ctx.Walk.Uncertain(clause) {
				continue
			}
			values := clause.Field("values")
			if values == nil {
				continue
			}
			for _, valueNode := range values.Children {
				if valueNode.Kind == parser.KindCaseRange {
					continue
				}
				value, ok := ctx.Semantic.Eval(valueNode)
				if !ok {
					continue
				}
				first := seen[value]
				if first == nil {
					seen[value] = valueNode
					continue
				}
				ctx.Report(diagnostic.Diagnostic{
					Message:  fmt.Sprintf("switch case value %d is duplicated", value),
					Filename: ctx.File.Path,
					Range:    ctx.Walk.Range(valueNode),
					Notes: []diagnostic.RelatedLocation{{
						Range:   ctx.Walk.Range(first),
						Message: "first case is here",
					}},
				})
			}
		}
	})
}
