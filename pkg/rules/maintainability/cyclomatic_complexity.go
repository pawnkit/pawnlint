package maintainability

import (
	"fmt"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type CyclomaticComplexity struct{}

func (CyclomaticComplexity) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "cyclomatic-complexity",
		Name:            "Cyclomatic complexity",
		Summary:         "Reports functions with too many independent control-flow paths",
		Explanation:     "Complexity starts at one and increases for conditionals, loops, non-default switch cases, ternary expressions, and short-circuit boolean operators. Inactive and uncertain conditional-compilation branches are ignored.",
		Category:        diagnostic.CategoryMaintainability,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SyntaxAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"complexity", "control-flow", "maintainability"},
		Options: []lint.Option{{
			Name: "maximum", Summary: "Maximum permitted complexity",
			Type: lint.OptionInteger, Default: int64(10), Minimum: 1, Maximum: 10000, HasMinimum: true, HasMaximum: true,
		}},
	}
}

func (CyclomaticComplexity) Run(ctx *lint.Context) {
	maximum := configuredCyclomaticMaximum(ctx)
	for _, function := range ctx.Walk.OfKind(parser.KindFunctionDefinition) {
		if ctx.Walk.Inactive(function) || function.HasError {
			continue
		}
		complexity := 1 + cyclomaticDecisionCount(ctx, function.Field("body"))
		if complexity <= maximum {
			continue
		}
		name := function.Field("name")
		if name == nil {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("function %q has cyclomatic complexity %d, exceeding the maximum of %d", ctx.Walk.Text(name), complexity, maximum),
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(name),
		})
	}
}

func configuredCyclomaticMaximum(ctx *lint.Context) int {
	if ctx.PerRule != nil && ctx.PerRule["cyclomatic-complexity"] != nil {
		if value, ok := ctx.PerRule["cyclomatic-complexity"]["maximum"].(int64); ok && value > 0 {
			return int(value)
		}
	}
	return 10
}

func cyclomaticDecisionCount(ctx *lint.Context, node *parser.Node) int {
	if node == nil || ctx.Walk.Inactive(node) || ctx.Walk.Uncertain(node) {
		return 0
	}
	count := 0
	switch node.Kind {
	case parser.KindIfStatement, parser.KindWhileStatement, parser.KindDoWhileStatement,
		parser.KindForStatement, parser.KindCaseClause, parser.KindTernaryExpression:
		count++
	case parser.KindBinaryExpression:
		if node.Tok.Kind == token.AndAnd || node.Tok.Kind == token.OrOr {
			count++
		}
	}
	for _, child := range node.Children {
		count += cyclomaticDecisionCount(ctx, child)
	}
	return count
}
