package semantic

import (
	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
)

func (m *Model) ExpressionTags(node *parser.Node) []string {
	node = unwrap(node)
	if node == nil || node.HasError || m.Walk.Uncertain(node) {
		return nil
	}
	switch node.Kind {
	case parser.KindIdentifier:
		if _, ok := m.BooleanLiteral(node); ok {
			return []string{"bool"}
		}
		return symbolTags(m.Resolve(node))
	case parser.KindTaggedExpression:
		tag := node.Field("tag")
		if tag == nil {
			return nil
		}
		name := m.Walk.Text(tag)
		if name == "" {
			return nil
		}
		return []string{name}
	case parser.KindCallExpression:
		return symbolTags(m.Resolve(unwrap(node.Field("function"))))
	case parser.KindSubscriptExpression:
		return m.ExpressionTags(node.Field("array"))
	case parser.KindAssignmentExpression:
		return m.ExpressionTags(node.Field("left"))
	case parser.KindUpdateExpression:
		return m.ExpressionTags(node.Field("expression"))
	case parser.KindUnaryExpression:
		if node.Tok.Kind == token.Bang {
			return []string{"bool"}
		}
		return m.ExpressionTags(node.Field("expression"))
	case parser.KindBinaryExpression:
		switch node.Tok.Kind {
		case token.Eq, token.NotEq, token.Lt, token.Gt, token.LtEq, token.GtEq, token.AndAnd, token.OrOr:
			return []string{"bool"}
		case token.Dot:
			return nil
		}
		left := m.ExpressionTags(node.Field("left"))
		right := m.ExpressionTags(node.Field("right"))
		if equalTags(left, right) {
			return left
		}
	case parser.KindTernaryExpression:
		left := m.ExpressionTags(node.Field("consequence"))
		right := m.ExpressionTags(node.Field("alternative"))
		if equalTags(left, right) {
			return left
		}
	case parser.KindExpressionList:
		if len(node.Children) != 0 {
			return m.ExpressionTags(node.Children[len(node.Children)-1])
		}
	}
	return nil
}

func symbolTags(symbol *Symbol) []string {
	if symbol == nil || symbol.Ambiguous || len(symbol.Tags) == 0 {
		return nil
	}
	return append([]string(nil), symbol.Tags...)
}

func equalTags(left, right []string) bool {
	if len(left) == 0 || len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
