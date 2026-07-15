package maintainability

import (
	"fmt"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type NPathComplexity struct{}

const npathComplexityLimit int64 = 1_000_000_000

func (NPathComplexity) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "npath-complexity",
		Name:            "NPath complexity",
		Summary:         "Reports functions with too many acyclic execution paths",
		Explanation:     "Alternative paths add and sequential branching statements multiply. Loops add an exit path, while short-circuit operators and ternaries add expression alternatives. Counts saturate safely and ignore inactive or uncertain conditional-compilation branches.",
		Category:        diagnostic.CategoryMaintainability,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SyntaxAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"complexity", "control-flow", "maintainability"},
		Options: []lint.Option{{
			Name: "maximum", Summary: "Maximum permitted NPath complexity",
			Type: lint.OptionInteger, Default: int64(200), Minimum: 1, Maximum: npathComplexityLimit - 1, HasMinimum: true, HasMaximum: true,
		}},
	}
}

func (NPathComplexity) Run(ctx *lint.Context) {
	maximum := configuredNPathMaximum(ctx)
	for _, function := range ctx.Walk.OfKind(parser.KindFunctionDefinition) {
		if ctx.Walk.Inactive(function) || function.HasError {
			continue
		}
		complexity := npathNode(ctx, function.Field("body"))
		if complexity <= maximum {
			continue
		}
		name := function.Field("name")
		if name == nil {
			continue
		}
		message := fmt.Sprintf("function %q has NPath complexity %d, exceeding the maximum of %d", ctx.Walk.Text(name), complexity, maximum)
		if complexity == npathComplexityLimit {
			message = fmt.Sprintf("function %q has NPath complexity of at least %d, exceeding the maximum of %d", ctx.Walk.Text(name), complexity, maximum)
		}
		ctx.Report(diagnostic.Diagnostic{Message: message, Filename: ctx.File.Path, Range: ctx.Walk.Range(name)})
	}
}

func configuredNPathMaximum(ctx *lint.Context) int64 {
	if ctx.PerRule != nil && ctx.PerRule["npath-complexity"] != nil {
		if value, ok := ctx.PerRule["npath-complexity"]["maximum"].(int64); ok && value > 0 {
			return value
		}
	}
	return 200
}

func npathNode(ctx *lint.Context, node *parser.Node) int64 {
	if node == nil || ctx.Walk.Inactive(node) || ctx.Walk.Uncertain(node) {
		return 1
	}
	switch node.Kind {
	case parser.KindBlock, parser.KindConditionalRegion, parser.KindConditionalBranch:
		return npathSequence(ctx, node.Children)
	case parser.KindIfStatement:
		thenPaths := npathNode(ctx, node.Field("consequence"))
		elsePaths := npathNode(ctx, node.Field("alternative"))
		paths := npathAdd(thenPaths, elsePaths)
		return npathAdd(paths, npathExpressionDecisions(ctx, node.Field("condition")))
	case parser.KindWhileStatement, parser.KindDoWhileStatement:
		paths := npathAdd(npathNode(ctx, node.Field("body")), 1)
		return npathAdd(paths, npathExpressionDecisions(ctx, node.Field("condition")))
	case parser.KindForStatement:
		paths := npathAdd(npathNode(ctx, node.Field("body")), 1)
		for _, field := range []string{"init", "condition", "increment"} {
			paths = npathAdd(paths, npathExpressionDecisions(ctx, node.Field(field)))
		}
		return paths
	case parser.KindSwitchStatement:
		return npathSwitch(ctx, node)
	case parser.KindCaseClause, parser.KindDefaultClause:
		return npathNode(ctx, node.Field("body"))
	case parser.KindMacroInvocationBlock:
		return npathNode(ctx, node.Field("body"))
	default:
		return npathAdd(1, npathExpressionDecisions(ctx, node))
	}
}

func npathSequence(ctx *lint.Context, nodes []*parser.Node) int64 {
	paths := int64(1)
	for _, child := range nodes {
		paths = npathMultiply(paths, npathNode(ctx, child))
	}
	return paths
}

func npathSwitch(ctx *lint.Context, node *parser.Node) int64 {
	paths := int64(0)
	hasDefault := false
	for _, child := range node.Children {
		if ctx.Walk.Inactive(child) || ctx.Walk.Uncertain(child) {
			continue
		}
		switch child.Kind {
		case parser.KindCaseClause:
			paths = npathAdd(paths, npathNode(ctx, child))
		case parser.KindDefaultClause:
			hasDefault = true
			paths = npathAdd(paths, npathNode(ctx, child))
		}
	}
	if !hasDefault {
		paths = npathAdd(paths, 1)
	}
	paths = npathAdd(paths, npathExpressionDecisions(ctx, node.Field("condition")))
	if paths == 0 {
		return 1
	}
	return paths
}

func npathExpressionDecisions(ctx *lint.Context, node *parser.Node) int64 {
	if node == nil || ctx.Walk.Inactive(node) || ctx.Walk.Uncertain(node) {
		return 0
	}
	decisions := int64(0)
	if node.Kind == parser.KindTernaryExpression || node.Kind == parser.KindBinaryExpression && (node.Tok.Kind == token.AndAnd || node.Tok.Kind == token.OrOr) {
		decisions = 1
	}
	for _, child := range node.Children {
		decisions = npathAdd(decisions, npathExpressionDecisions(ctx, child))
	}
	return decisions
}

func npathAdd(left, right int64) int64 {
	if left >= npathComplexityLimit-right {
		return npathComplexityLimit
	}
	return left + right
}

func npathMultiply(left, right int64) int64 {
	if left == 0 || right == 0 {
		return 0
	}
	if left >= npathComplexityLimit/right {
		return npathComplexityLimit
	}
	return left * right
}
