package semantic

import (
	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
)

func (m *Model) Equivalent(left, right *parser.Node) bool {
	left = unwrap(left)
	right = unwrap(right)
	if left == nil || right == nil || left.Kind != right.Kind || left.HasError || right.HasError {
		return false
	}
	if !m.certainSubtree(left) || !m.certainSubtree(right) {
		return false
	}
	switch left.Kind {
	case parser.KindIdentifier:
		leftSymbol := m.Resolve(left)
		return leftSymbol != nil && m.Resolve(right) == leftSymbol
	case parser.KindLiteral:
		return left.Tok.Kind == right.Tok.Kind && left.Tok.Text(m.File.Source) == right.Tok.Text(m.File.Source)
	}
	if left.Tok.Kind != right.Tok.Kind || len(left.Children) != len(right.Children) {
		return false
	}
	for i := range left.Children {
		if !m.Equivalent(left.Children[i], right.Children[i]) {
			return false
		}
	}
	return true
}

func (m *Model) certainSubtree(node *parser.Node) bool {
	if node == nil || node.HasError || m.Walk.Uncertain(node) {
		return false
	}
	for _, child := range node.Children {
		if !m.certainSubtree(child) {
			return false
		}
	}
	return true
}

func (m *Model) EquivalentSyntax(left, right *parser.Node) bool {
	if left == nil || right == nil || left.Kind != right.Kind || left.HasError || right.HasError {
		return false
	}
	if !m.certainSubtree(left) || !m.certainSubtree(right) {
		return false
	}
	leftTokens := m.tokensWithin(left)
	rightTokens := m.tokensWithin(right)
	if len(leftTokens) != len(rightTokens) {
		return false
	}
	for i := range leftTokens {
		if leftTokens[i].Kind != rightTokens[i].Kind || leftTokens[i].Text(m.File.Source) != rightTokens[i].Text(m.File.Source) {
			return false
		}
	}
	return len(leftTokens) != 0
}

func (m *Model) tokensWithin(node *parser.Node) []token.Token {
	var result []token.Token
	for _, tok := range m.File.Tokens {
		if tok.Start.Offset >= node.Start && tok.End.Offset <= node.End {
			result = append(result, tok)
		}
	}
	return result
}

func (m *Model) Pure(node *parser.Node) bool {
	node = unwrap(node)
	if node == nil || node.HasError || m.Walk.Uncertain(node) {
		return false
	}
	switch node.Kind {
	case parser.KindIdentifier, parser.KindLiteral:
		return true
	case parser.KindUnaryExpression:
		if node.Tok.Kind == token.PlusPlus || node.Tok.Kind == token.MinusMinus {
			return false
		}
	case parser.KindBinaryExpression, parser.KindTernaryExpression,
		parser.KindSubscriptExpression, parser.KindSizeofExpression,
		parser.KindTagofExpression, parser.KindTaggedExpression:
	case parser.KindParenthesizedExpression:
	default:
		return false
	}
	for _, child := range node.Children {
		if !m.Pure(child) {
			return false
		}
	}
	return true
}

func unwrap(node *parser.Node) *parser.Node {
	for node != nil && node.Kind == parser.KindParenthesizedExpression {
		node = node.Field("expression")
	}
	return node
}
