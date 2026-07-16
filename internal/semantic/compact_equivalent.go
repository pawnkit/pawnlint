package semantic

import (
	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/lexer"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/syntax"
)

func (m *CompactModel) Equivalent(left, right syntax.NodeID) bool {
	left = m.compactUnwrap(left)
	right = m.compactUnwrap(right)
	if !m.Walk.Tree.Valid(left) || !m.Walk.Tree.Valid(right) || m.Walk.Tree.Kind(left) != m.Walk.Tree.Kind(right) || m.Walk.Tree.HasError(left) || m.Walk.Tree.HasError(right) {
		return false
	}
	if !m.compactCertainSubtree(left) || !m.compactCertainSubtree(right) {
		return false
	}
	switch m.Walk.Tree.Kind(left) {
	case parser.KindIdentifier:
		leftSymbol := m.Resolve(left)
		return leftSymbol != nil && m.Resolve(right) == leftSymbol
	case parser.KindLiteral:
		return m.Walk.Tree.TokenKind(left) == m.Walk.Tree.TokenKind(right) && m.Walk.Tree.TokenText(left) == m.Walk.Tree.TokenText(right)
	}
	if m.Walk.Tree.TokenKind(left) != m.Walk.Tree.TokenKind(right) || m.Walk.Tree.ChildCount(left) != m.Walk.Tree.ChildCount(right) {
		return false
	}
	for index := 0; index < m.Walk.Tree.ChildCount(left); index++ {
		if !m.Equivalent(m.Walk.Tree.Child(left, index), m.Walk.Tree.Child(right, index)) {
			return false
		}
	}
	return true
}

func (m *CompactModel) compactCertainSubtree(node syntax.NodeID) bool {
	if !m.Walk.Tree.Valid(node) || m.Walk.Tree.HasError(node) || m.Walk.Uncertain(node) {
		return false
	}
	for index := 0; index < m.Walk.Tree.ChildCount(node); index++ {
		if !m.compactCertainSubtree(m.Walk.Tree.Child(node, index)) {
			return false
		}
	}
	return true
}

func (m *CompactModel) EquivalentSyntax(left, right syntax.NodeID) bool {
	if !m.Walk.Tree.Valid(left) || !m.Walk.Tree.Valid(right) || m.Walk.Tree.Kind(left) != m.Walk.Tree.Kind(right) || m.Walk.Tree.HasError(left) || m.Walk.Tree.HasError(right) {
		return false
	}
	if !m.compactCertainSubtree(left) || !m.compactCertainSubtree(right) {
		return false
	}
	leftSource := []byte(m.Walk.Text(left))
	rightSource := []byte(m.Walk.Text(right))
	leftTokens := lexer.Tokenize(leftSource)
	rightTokens := lexer.Tokenize(rightSource)
	if len(leftTokens) != len(rightTokens) {
		return false
	}
	for index := range leftTokens {
		if leftTokens[index].Kind != rightTokens[index].Kind || leftTokens[index].Text(leftSource) != rightTokens[index].Text(rightSource) {
			return false
		}
	}
	return len(leftTokens) > 1 || len(leftTokens) == 1 && leftTokens[0].Kind != token.EOF
}

func (m *CompactModel) Pure(node syntax.NodeID) bool {
	node = m.compactUnwrap(node)
	if !m.Walk.Tree.Valid(node) || m.Walk.Tree.HasError(node) || m.Walk.Uncertain(node) {
		return false
	}
	switch m.Walk.Tree.Kind(node) {
	case parser.KindIdentifier, parser.KindLiteral:
		return true
	case parser.KindUnaryExpression:
		if m.Walk.Tree.TokenKind(node) == token.PlusPlus || m.Walk.Tree.TokenKind(node) == token.MinusMinus {
			return false
		}
	case parser.KindBinaryExpression, parser.KindTernaryExpression,
		parser.KindSubscriptExpression, parser.KindSizeofExpression,
		parser.KindTagofExpression, parser.KindTaggedExpression,
		parser.KindParenthesizedExpression:
	default:
		return false
	}
	for index := 0; index < m.Walk.Tree.ChildCount(node); index++ {
		if !m.Pure(m.Walk.Tree.Child(node, index)) {
			return false
		}
	}
	return true
}
