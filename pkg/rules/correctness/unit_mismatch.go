package correctness

import (
	"fmt"
	"strings"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type UnitMismatch struct{}

func (UnitMismatch) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "unit-mismatch",
		Name:            "Unit mismatch",
		Summary:         "Reports operations between incompatible configured unit tags",
		Explanation:     "Configured unit groups map equivalent Pawn tags to the same unit. Assignments, initializers, returns, addition, subtraction, comparisons, and conditional branches must use the same configured unit. Untagged, unknown, union, macro-derived, and uncertain expressions are ignored.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"units", "tags", "conversions", "semantic"},
		Options: []lint.Option{{
			Name: "units", Summary: "Unit objects with name and equivalent tags", Type: lint.OptionObjectList,
			Default: []map[string]any{}, Validate: validateUnitGroups,
			Fields: []lint.Option{
				{Name: "name", Type: lint.OptionString, Required: true},
				{Name: "tags", Type: lint.OptionStringList, Required: true},
			},
		}},
	}
}

func validateUnitGroups(value any) error {
	groups, _ := value.([]map[string]any)
	names := make(map[string]bool)
	tags := make(map[string]bool)
	for index, group := range groups {
		name, _ := group["name"].(string)
		name = strings.TrimSpace(name)
		if name == "" {
			return fmt.Errorf("entry %d requires a non-empty name", index+1)
		}
		if names[name] {
			return fmt.Errorf("entry %d repeats unit name %q", index+1, name)
		}
		names[name] = true
		unitTags, _ := group["tags"].([]string)
		if len(unitTags) == 0 {
			return fmt.Errorf("entry %d requires at least one tag", index+1)
		}
		for _, tag := range unitTags {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				return fmt.Errorf("entry %d contains an empty tag", index+1)
			}
			if tags[tag] {
				return fmt.Errorf("entry %d repeats tag %q", index+1, tag)
			}
			tags[tag] = true
		}
	}
	return nil
}

func (UnitMismatch) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	units := configuredUnits(ctx)
	if len(units) == 0 {
		return
	}
	for _, symbol := range ctx.Semantic.Symbols {
		if !unitSymbolCandidate(ctx, symbol) {
			continue
		}
		value := unitSymbolValue(symbol)
		if value == nil || unitMismatchSkipped(ctx, value) {
			continue
		}
		unitMismatchReport(ctx, value, symbol.Tags[0], unitExpressionTag(ctx, value), units)
	}
	for _, node := range ctx.Walk.OfKind(parser.KindAssignmentExpression) {
		if !unitAssignmentOperator(node.Tok.Kind) || unitMismatchSkipped(ctx, node) {
			continue
		}
		unitMismatchReport(ctx, node, unitExpressionTag(ctx, node.Field("left")), unitExpressionTag(ctx, node.Field("right")), units)
	}
	for _, node := range ctx.Walk.OfKind(parser.KindBinaryExpression) {
		if !unitBinaryOperator(node.Tok.Kind) || unitMismatchSkipped(ctx, node) {
			continue
		}
		unitMismatchReport(ctx, node, unitExpressionTag(ctx, node.Field("left")), unitExpressionTag(ctx, node.Field("right")), units)
	}
	for _, node := range ctx.Walk.OfKind(parser.KindTernaryExpression) {
		if unitMismatchSkipped(ctx, node) {
			continue
		}
		unitMismatchReport(ctx, node, unitExpressionTag(ctx, node.Field("consequence")), unitExpressionTag(ctx, node.Field("alternative")), units)
	}
	functionUnits := make(map[*parser.Node]string)
	for _, symbol := range ctx.Semantic.Symbols {
		if symbol != nil && symbol.Kind == semantic.SymbolFunction && !symbol.Ambiguous && len(symbol.Tags) == 1 {
			functionUnits[symbol.Decl] = symbol.Tags[0]
		}
	}
	for _, node := range ctx.Walk.OfKind(parser.KindReturnStatement) {
		value := node.Field("value")
		if value == nil || unitMismatchSkipped(ctx, node) {
			continue
		}
		unitMismatchReport(ctx, value, functionUnits[ctx.Walk.EnclosingFunction(node)], unitExpressionTag(ctx, value), units)
	}
}

func configuredUnits(ctx *lint.Context) map[string]string {
	result := make(map[string]string)
	if ctx.PerRule == nil || ctx.PerRule["unit-mismatch"] == nil {
		return result
	}
	groups, _ := ctx.PerRule["unit-mismatch"]["units"].([]map[string]any)
	for _, group := range groups {
		name, _ := group["name"].(string)
		tags, _ := group["tags"].([]string)
		for _, tag := range tags {
			result[tag] = name
		}
	}
	return result
}

func unitSymbolCandidate(ctx *lint.Context, symbol *semantic.Symbol) bool {
	return symbol != nil && !symbol.Ambiguous && len(symbol.Tags) == 1 && symbol.Decl != nil && !ctx.Walk.Inactive(symbol.Decl) && !ctx.Walk.Uncertain(symbol.Decl)
}

func unitSymbolValue(symbol *semantic.Symbol) *parser.Node {
	switch symbol.Kind {
	case semantic.SymbolGlobal, semantic.SymbolLocal:
		return symbol.Decl.Field("initializer")
	case semantic.SymbolParameter:
		return symbol.Decl.Field("default")
	default:
		return nil
	}
}

func unitExpressionTag(ctx *lint.Context, node *parser.Node) string {
	if node == nil || unitNodeHasError(node) || node.Kind == parser.KindTaggedExpression && node.Field("expression") == nil {
		return ""
	}
	tag, ok := ctx.Semantic.ExpressionTag(node)
	if !ok {
		return ""
	}
	return tag
}

func unitMismatchReport(ctx *lint.Context, node *parser.Node, leftTag, rightTag string, units map[string]string) {
	leftUnit, leftOK := units[leftTag]
	rightUnit, rightOK := units[rightTag]
	if !leftOK || !rightOK || leftUnit == rightUnit {
		return
	}
	ctx.Report(diagnostic.Diagnostic{
		Message:  fmt.Sprintf("unit tags %q and %q are incompatible", leftTag, rightTag),
		Filename: ctx.File.Path,
		Range:    ctx.Walk.Range(node),
	})
}

func unitAssignmentOperator(kind token.Kind) bool {
	switch kind {
	case token.Assign, token.PlusAssign, token.MinusAssign:
		return true
	default:
		return false
	}
}

func unitBinaryOperator(kind token.Kind) bool {
	switch kind {
	case token.Plus, token.Minus, token.Eq, token.NotEq, token.Lt, token.Gt, token.LtEq, token.GtEq:
		return true
	default:
		return false
	}
}

func unitMismatchSkipped(ctx *lint.Context, node *parser.Node) bool {
	if node == nil || unitNodeHasError(node) || node.Tok.Origin != nil || ctx.Walk.Inactive(node) || ctx.Walk.Uncertain(node) {
		return true
	}
	for current := node; current != nil; current = ctx.Walk.Parent(current) {
		switch current.Kind {
		case parser.KindDirectiveDefine, parser.KindMacroBody, parser.KindMacroInvocation, parser.KindMacroInvocationBlock,
			parser.KindSharedConditional, parser.KindConditionalFunction, parser.KindConditionalSplice:
			return true
		}
	}
	return false
}

func unitNodeHasError(node *parser.Node) bool {
	if node == nil {
		return false
	}
	if node.HasError {
		return true
	}
	for _, child := range node.Children {
		if unitNodeHasError(child) {
			return true
		}
	}
	return false
}
