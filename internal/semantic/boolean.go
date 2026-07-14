package semantic

import (
	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
)

func (m *Model) BooleanLiteral(node *parser.Node) (bool, bool) {
	node = unwrap(node)
	if node == nil || node.Kind != parser.KindIdentifier || m.Resolve(node) != nil || m.Walk.Uncertain(node) {
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

func (m *Model) Boolean(node *parser.Node) bool {
	node = unwrap(node)
	if node == nil || node.HasError || m.Walk.Uncertain(node) {
		return false
	}
	if _, ok := m.BooleanLiteral(node); ok {
		return true
	}
	if tags := m.ExpressionTags(node); len(tags) == 1 && tags[0] == "bool" {
		return true
	}
	if node.Kind == parser.KindUnaryExpression {
		return node.Tok.Kind == token.Bang
	}
	if node.Kind != parser.KindBinaryExpression {
		return false
	}
	switch node.Tok.Kind {
	case token.Eq, token.NotEq, token.Lt, token.Gt, token.LtEq, token.GtEq, token.AndAnd, token.OrOr:
		return true
	default:
		return false
	}
}
