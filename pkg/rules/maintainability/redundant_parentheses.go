package maintainability

import (
	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type RedundantParentheses struct{}

func (RedundantParentheses) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "redundant-parentheses",
		Name:            "Redundant parentheses",
		Summary:         "Reports expression parentheses that do not affect parsing",
		Explanation:     "Parentheses are redundant when removing them preserves Pawn precedence, associativity, argument boundaries, statement syntax, and assignment-condition intent. Macro, uncertain, and malformed syntax is ignored.",
		Category:        diagnostic.CategoryStyle,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SyntaxAnalysis,
		DefaultEnabled:  false,
		Fixable:         true,
		Tags:            []string{"expressions", "parentheses", "style"},
	}
}

func (RedundantParentheses) Run(ctx *lint.Context) {
	for _, node := range ctx.Walk.OfKind(parser.KindParenthesizedExpression) {
		inner := node.Field("expression")
		if inner == nil || node.HasError || inner.HasError || ctx.Walk.Inactive(node) || ctx.Walk.Uncertain(node) || redundantParenthesesInMacro(ctx, node) {
			continue
		}
		if parent := ctx.Walk.Parent(node); parent != nil && parent.Kind == parser.KindParenthesizedExpression && !redundantParenthesesRequired(ctx, parent) {
			continue
		}
		if redundantParenthesesRequired(ctx, node) {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  "parentheses do not affect this expression",
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(node),
			Fix:      redundantParenthesesFix(ctx, node),
		})
	}
}

func redundantParenthesesInMacro(ctx *lint.Context, node *parser.Node) bool {
	for current := node; current != nil; current = ctx.Walk.Parent(current) {
		switch current.Kind {
		case parser.KindDirectiveDefine, parser.KindMacroBody, parser.KindMacroInvocation, parser.KindMacroInvocationBlock:
			return true
		case parser.KindFunctionDefinition:
			return false
		}
	}
	return false
}

func redundantParenthesesRequired(ctx *lint.Context, node *parser.Node) bool {
	inner := node.Field("expression")
	parent := ctx.Walk.Parent(node)
	if inner == nil || parent == nil {
		return false
	}
	if redundantParenthesesRequiredCondition(parent, node) {
		return true
	}
	if parent.Kind == parser.KindParenthesizedExpression {
		grandparent := ctx.Walk.Parent(parent)
		if redundantParenthesesRequiredCondition(grandparent, parent) && inner.Kind == parser.KindAssignmentExpression {
			return true
		}
		return false
	}
	innerPrecedence := redundantParenthesesPrecedence(inner)
	switch parent.Kind {
	case parser.KindBinaryExpression:
		parentPrecedence := redundantParenthesesPrecedence(parent)
		if innerPrecedence != parentPrecedence {
			return innerPrecedence < parentPrecedence
		}
		if parentPrecedence == redundantPrecedenceRelational {
			return true
		}
		return parent.Field("right") == node
	case parser.KindAssignmentExpression:
		if innerPrecedence != redundantPrecedenceAssignment {
			return innerPrecedence < redundantPrecedenceAssignment
		}
		return parent.Field("left") == node
	case parser.KindTernaryExpression:
		switch {
		case parent.Field("condition") == node:
			return innerPrecedence <= redundantPrecedenceTernary
		case parent.Field("consequence") == node, parent.Field("alternative") == node:
			return innerPrecedence == redundantPrecedenceComma
		}
	case parser.KindUnaryExpression, parser.KindTaggedExpression:
		if parent.Kind == parser.KindUnaryExpression && !redundantParenthesesBuiltinUnary(parent.Tok.Kind) {
			return true
		}
		return innerPrecedence < redundantPrecedenceUnary
	case parser.KindCallExpression:
		if parent.Field("function") == node {
			return innerPrecedence < redundantPrecedencePostfix
		}
	case parser.KindSubscriptExpression:
		if parent.Field("array") == node {
			return innerPrecedence < redundantPrecedencePostfix
		}
	case parser.KindUpdateExpression:
		return innerPrecedence < redundantPrecedencePostfix
	case parser.KindArgumentList, parser.KindArrayLiteral:
		return innerPrecedence == redundantPrecedenceComma
	case parser.KindVariableDeclarator:
		if parent.Field("initializer") == node {
			return innerPrecedence == redundantPrecedenceComma
		}
	case parser.KindParameter:
		if parent.Field("default_value") == node {
			return innerPrecedence == redundantPrecedenceComma
		}
	case parser.KindEnumEntry:
		if parent.Field("value") == node {
			return innerPrecedence <= redundantPrecedenceAssignment
		}
	case parser.KindCaseValueList, parser.KindCaseRange:
		return innerPrecedence <= redundantPrecedenceAssignment
	case parser.KindExpressionStatement:
		return parent.Field("expression") == node && inner.Kind == parser.KindTaggedExpression
	}
	return false
}

func redundantParenthesesRequiredCondition(parent, node *parser.Node) bool {
	if parent == nil || parent.Field("condition") != node {
		return false
	}
	switch parent.Kind {
	case parser.KindIfStatement, parser.KindWhileStatement, parser.KindDoWhileStatement, parser.KindSwitchStatement:
		return true
	default:
		return false
	}
}

const (
	redundantPrecedenceComma = iota + 1
	redundantPrecedenceAssignment
	redundantPrecedenceTernary
	redundantPrecedenceLogicalOr
	redundantPrecedenceLogicalAnd
	redundantPrecedenceBitOr
	redundantPrecedenceBitXor
	redundantPrecedenceBitAnd
	redundantPrecedenceEquality
	redundantPrecedenceRelational
	redundantPrecedenceShift
	redundantPrecedenceAdditive
	redundantPrecedenceMultiplicative
	redundantPrecedenceUnary
	redundantPrecedencePostfix
	redundantPrecedencePrimary
)

func redundantParenthesesPrecedence(node *parser.Node) int {
	if node == nil {
		return redundantPrecedenceComma
	}
	switch node.Kind {
	case parser.KindExpressionList:
		return redundantPrecedenceComma
	case parser.KindAssignmentExpression:
		return redundantPrecedenceAssignment
	case parser.KindTernaryExpression:
		return redundantPrecedenceTernary
	case parser.KindBinaryExpression:
		switch node.Tok.Kind {
		case token.OrOr:
			return redundantPrecedenceLogicalOr
		case token.AndAnd:
			return redundantPrecedenceLogicalAnd
		case token.Pipe:
			return redundantPrecedenceBitOr
		case token.Caret:
			return redundantPrecedenceBitXor
		case token.Amp:
			return redundantPrecedenceBitAnd
		case token.Eq, token.NotEq:
			return redundantPrecedenceEquality
		case token.Lt, token.Gt, token.LtEq, token.GtEq:
			return redundantPrecedenceRelational
		case token.Shl, token.Shr, token.Ushr:
			return redundantPrecedenceShift
		case token.Plus, token.Minus:
			return redundantPrecedenceAdditive
		case token.Star, token.Slash, token.Percent:
			return redundantPrecedenceMultiplicative
		case token.Dot, token.ColonColon:
			return redundantPrecedencePostfix
		}
	case parser.KindUnaryExpression, parser.KindTaggedExpression, parser.KindSizeofExpression, parser.KindTagofExpression, parser.KindDefinedExpression:
		return redundantPrecedenceUnary
	case parser.KindCallExpression, parser.KindSubscriptExpression, parser.KindUpdateExpression, parser.KindMacroBody:
		return redundantPrecedencePostfix
	case parser.KindParenthesizedExpression:
		return redundantParenthesesPrecedence(node.Field("expression"))
	}
	return redundantPrecedencePrimary
}

func redundantParenthesesBuiltinUnary(kind token.Kind) bool {
	switch kind {
	case token.Bang, token.Tilde, token.Minus, token.Plus, token.PlusPlus, token.MinusMinus:
		return true
	default:
		return false
	}
}

func redundantParenthesesFix(ctx *lint.Context, node *parser.Node) *diagnostic.Fix {
	nodes := []*parser.Node{node}
	if !redundantParenthesesAssignmentOptOut(ctx, node) {
		for inner := node.Field("expression"); inner != nil && inner.Kind == parser.KindParenthesizedExpression; inner = inner.Field("expression") {
			nodes = append(nodes, inner)
		}
	}
	edits := make([]diagnostic.Edit, 0, len(nodes)*2)
	for _, current := range nodes {
		if current.Start < 0 || current.End > len(ctx.File.Source) || current.End-current.Start < 2 || ctx.File.Source[current.Start] != '(' || ctx.File.Source[current.End-1] != ')' {
			return nil
		}
		edits = append(edits, diagnostic.Edit{Range: ctx.File.LineTable.Range(current.Start, current.Start+1), NewText: ""})
	}
	for index := len(nodes) - 1; index >= 0; index-- {
		current := nodes[index]
		edits = append(edits, diagnostic.Edit{Range: ctx.File.LineTable.Range(current.End-1, current.End), NewText: ""})
	}
	return &diagnostic.Fix{
		Description: "remove the redundant parentheses",
		Edits:       edits,
	}
}

func redundantParenthesesAssignmentOptOut(ctx *lint.Context, node *parser.Node) bool {
	current := node
	for inner := current.Field("expression"); inner != nil && inner.Kind == parser.KindParenthesizedExpression; inner = current.Field("expression") {
		current = inner
	}
	inner := current.Field("expression")
	if inner == nil || inner.Kind != parser.KindAssignmentExpression {
		return false
	}
	top := node
	for parent := ctx.Walk.Parent(top); parent != nil && parent.Kind == parser.KindParenthesizedExpression; parent = ctx.Walk.Parent(top) {
		top = parent
	}
	return redundantParenthesesRequiredCondition(ctx.Walk.Parent(top), top)
}
