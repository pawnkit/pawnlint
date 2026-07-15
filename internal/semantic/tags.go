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

func (m *Model) ExpressionTag(node *parser.Node) (string, bool) {
	node = unwrap(node)
	if node == nil || node.HasError || m.Walk.Uncertain(node) {
		return "", false
	}
	switch node.Kind {
	case parser.KindIdentifier:
		if _, ok := m.BooleanLiteral(node); ok {
			return "bool", true
		}
		return singleSymbolTag(m.Resolve(node))
	case parser.KindTaggedExpression:
		tag := node.Field("tag")
		if tag == nil {
			return "", false
		}
		name := m.Walk.Text(tag)
		return name, name != ""
	case parser.KindCallExpression:
		return singleSymbolTag(m.Resolve(unwrap(node.Field("function"))))
	case parser.KindSubscriptExpression:
		return m.ExpressionTag(node.Field("array"))
	case parser.KindAssignmentExpression:
		return m.ExpressionTag(node.Field("left"))
	case parser.KindUpdateExpression:
		return m.ExpressionTag(node.Field("expression"))
	case parser.KindUnaryExpression:
		if node.Tok.Kind == token.Bang {
			return "bool", true
		}
		return m.ExpressionTag(node.Field("expression"))
	case parser.KindBinaryExpression:
		switch node.Tok.Kind {
		case token.Eq, token.NotEq, token.Lt, token.Gt, token.LtEq, token.GtEq, token.AndAnd, token.OrOr:
			return "bool", true
		case token.Dot:
			return "", false
		}
		left, leftOK := m.ExpressionTag(node.Field("left"))
		right, rightOK := m.ExpressionTag(node.Field("right"))
		return left, leftOK && rightOK && left == right
	case parser.KindTernaryExpression:
		left, leftOK := m.ExpressionTag(node.Field("consequence"))
		right, rightOK := m.ExpressionTag(node.Field("alternative"))
		return left, leftOK && rightOK && left == right
	case parser.KindExpressionList:
		if len(node.Children) != 0 {
			return m.ExpressionTag(node.Children[len(node.Children)-1])
		}
	}
	return "", false
}

func singleSymbolTag(symbol *Symbol) (string, bool) {
	if symbol == nil || symbol.Ambiguous || len(symbol.Tags) != 1 || symbol.Tags[0] == "" {
		return "", false
	}
	return symbol.Tags[0], true
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
