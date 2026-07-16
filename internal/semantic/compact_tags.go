package semantic

import (
	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/syntax"
)

func (m *CompactModel) ExpressionTags(node syntax.NodeID) []string {
	node = m.compactUnwrap(node)
	if !m.Walk.Tree.Valid(node) || m.Walk.Tree.HasError(node) || m.Walk.Uncertain(node) {
		return nil
	}
	switch m.Walk.Tree.Kind(node) {
	case parser.KindIdentifier:
		if _, ok := m.BooleanLiteral(node); ok {
			return []string{"bool"}
		}
		return compactSymbolTags(m.Resolve(node))
	case parser.KindTaggedExpression:
		tag := m.Walk.Tree.Field(node, "tag")
		if tag == syntax.NoNode {
			return nil
		}
		name := m.Walk.Text(tag)
		if name != "" {
			return []string{name}
		}
	case parser.KindCallExpression:
		return compactSymbolTags(m.Resolve(m.compactUnwrap(m.Walk.Tree.Field(node, "function"))))
	case parser.KindSubscriptExpression:
		return m.ExpressionTags(m.Walk.Tree.Field(node, "array"))
	case parser.KindAssignmentExpression:
		return m.ExpressionTags(m.Walk.Tree.Field(node, "left"))
	case parser.KindUpdateExpression:
		return m.ExpressionTags(m.Walk.Tree.Field(node, "expression"))
	case parser.KindUnaryExpression:
		if m.Walk.Tree.TokenKind(node) == token.Bang {
			return []string{"bool"}
		}
		return m.ExpressionTags(m.Walk.Tree.Field(node, "expression"))
	case parser.KindBinaryExpression:
		switch m.Walk.Tree.TokenKind(node) {
		case token.Eq, token.NotEq, token.Lt, token.Gt, token.LtEq, token.GtEq, token.AndAnd, token.OrOr:
			return []string{"bool"}
		case token.Dot:
			return nil
		}
		left := m.ExpressionTags(m.Walk.Tree.Field(node, "left"))
		right := m.ExpressionTags(m.Walk.Tree.Field(node, "right"))
		if equalTags(left, right) {
			return left
		}
	case parser.KindTernaryExpression:
		left := m.ExpressionTags(m.Walk.Tree.Field(node, "consequence"))
		right := m.ExpressionTags(m.Walk.Tree.Field(node, "alternative"))
		if equalTags(left, right) {
			return left
		}
	case parser.KindExpressionList:
		count := m.Walk.Tree.ChildCount(node)
		if count != 0 {
			return m.ExpressionTags(m.Walk.Tree.Child(node, count-1))
		}
	}
	return nil
}

func (m *CompactModel) ExpressionTag(node syntax.NodeID) (string, bool) {
	tags := m.ExpressionTags(node)
	if len(tags) != 1 || tags[0] == "" {
		return "", false
	}
	return tags[0], true
}

func (m *CompactModel) BooleanLiteral(node syntax.NodeID) (bool, bool) {
	node = m.compactUnwrap(node)
	if !m.Walk.Tree.Valid(node) || m.Walk.Tree.Kind(node) != parser.KindIdentifier || m.Resolve(node) != nil || m.Walk.Uncertain(node) {
		return false, false
	}
	switch m.Walk.Text(node) {
	case "true":
		return true, true
	case "false":
		return false, true
	default:
		return false, false
	}
}

func (m *CompactModel) Boolean(node syntax.NodeID) bool {
	node = m.compactUnwrap(node)
	if !m.Walk.Tree.Valid(node) || m.Walk.Tree.HasError(node) || m.Walk.Uncertain(node) {
		return false
	}
	if _, ok := m.BooleanLiteral(node); ok {
		return true
	}
	if tags := m.ExpressionTags(node); len(tags) == 1 && tags[0] == "bool" {
		return true
	}
	if m.Walk.Tree.Kind(node) == parser.KindUnaryExpression {
		return m.Walk.Tree.TokenKind(node) == token.Bang
	}
	if m.Walk.Tree.Kind(node) != parser.KindBinaryExpression {
		return false
	}
	switch m.Walk.Tree.TokenKind(node) {
	case token.Eq, token.NotEq, token.Lt, token.Gt, token.LtEq, token.GtEq, token.AndAnd, token.OrOr:
		return true
	default:
		return false
	}
}

func (m *CompactModel) compactUnwrap(node syntax.NodeID) syntax.NodeID {
	for m.Walk.Tree.Valid(node) && m.Walk.Tree.Kind(node) == parser.KindParenthesizedExpression {
		node = m.Walk.Tree.Field(node, "expression")
	}
	return node
}

func compactSymbolTags(symbol *CompactSymbol) []string {
	if symbol == nil || symbol.Ambiguous || len(symbol.Tags) == 0 {
		return nil
	}
	return append([]string(nil), symbol.Tags...)
}
