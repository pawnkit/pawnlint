package semantic

import (
	"strconv"
	"strings"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/syntax"
)

func (m *CompactModel) Eval(node syntax.NodeID) (int64, bool) {
	return m.compactEval(node, make(map[*CompactSymbol]bool), nil, nil)
}

func (m *CompactModel) EvalWithValues(node syntax.NodeID, values map[*CompactSymbol]int64) (int64, bool) {
	return m.compactEval(node, make(map[*CompactSymbol]bool), values, nil)
}

func (m *CompactModel) EvalWithResolver(node syntax.NodeID, resolver func(syntax.NodeID) (int64, bool)) (int64, bool) {
	return m.compactEval(node, make(map[*CompactSymbol]bool), nil, resolver)
}

func (m *CompactModel) ConstantValue(symbol *CompactSymbol) (int64, bool) {
	if m == nil || symbol == nil || !symbol.Constant || symbol.Ambiguous {
		return 0, false
	}
	if value, ok := m.constants[symbol]; ok {
		return value, true
	}
	if symbol.Value == syntax.NoNode {
		return 0, false
	}
	return m.Eval(symbol.Value)
}

func (m *CompactModel) compactEval(node syntax.NodeID, visiting map[*CompactSymbol]bool, values map[*CompactSymbol]int64, resolver func(syntax.NodeID) (int64, bool)) (int64, bool) {
	if !m.Walk.Tree.Valid(node) || m.Walk.Tree.HasError(node) || m.Walk.Uncertain(node) {
		return 0, false
	}
	switch m.Walk.Tree.Kind(node) {
	case parser.KindLiteral:
		return m.compactEvalLiteral(node)
	case parser.KindParenthesizedExpression:
		return m.compactEval(m.Walk.Tree.Field(node, "expression"), visiting, values, resolver)
	case parser.KindUnaryExpression:
		value, ok := m.compactEval(m.Walk.Tree.Field(node, "expression"), visiting, values, resolver)
		if !ok {
			return 0, false
		}
		switch m.Walk.Tree.TokenKind(node) {
		case token.Plus:
			return value, true
		case token.Minus:
			return cell(-value), true
		case token.Bang:
			return truth(value == 0), true
		case token.Tilde:
			return cell(^value), true
		default:
			return 0, false
		}
	case parser.KindBinaryExpression:
		left, leftOK := m.compactEval(m.Walk.Tree.Field(node, "left"), visiting, values, resolver)
		right, rightOK := m.compactEval(m.Walk.Tree.Field(node, "right"), visiting, values, resolver)
		if !leftOK || !rightOK {
			return 0, false
		}
		return evalBinary(m.Walk.Tree.TokenKind(node), left, right)
	case parser.KindTernaryExpression:
		condition, ok := m.compactEval(m.Walk.Tree.Field(node, "condition"), visiting, values, resolver)
		if !ok {
			return 0, false
		}
		if condition != 0 {
			return m.compactEval(m.Walk.Tree.Field(node, "consequence"), visiting, values, resolver)
		}
		return m.compactEval(m.Walk.Tree.Field(node, "alternative"), visiting, values, resolver)
	case parser.KindIdentifier:
		if value, ok := builtinValue(m.Walk.Text(node)); ok && m.Resolve(node) == nil {
			return value, true
		}
		symbol := m.Resolve(node)
		if symbol == nil && resolver != nil {
			return resolver(node)
		}
		if value, ok := values[symbol]; ok && symbol != nil {
			return value, true
		}
		if symbol == nil || !symbol.Constant {
			return 0, false
		}
		if value, ok := m.constants[symbol]; ok {
			return value, true
		}
		if symbol.Value == syntax.NoNode || visiting[symbol] {
			return 0, false
		}
		visiting[symbol] = true
		value, ok := m.compactEval(symbol.Value, visiting, values, resolver)
		delete(visiting, symbol)
		return value, ok
	default:
		return 0, false
	}
}

func (m *CompactModel) compactEvalLiteral(node syntax.NodeID) (int64, bool) {
	if m.Walk.Tree.TokenKind(node) == token.KwNull {
		return 0, true
	}
	if m.Walk.Tree.TokenKind(node) != token.IntLiteral {
		return 0, false
	}
	text := strings.ReplaceAll(m.Walk.Tree.TokenText(node), "_", "")
	base := 10
	if strings.HasPrefix(text, "0x") || strings.HasPrefix(text, "0X") || strings.HasPrefix(text, "0b") || strings.HasPrefix(text, "0B") {
		base = 0
	}
	value, err := strconv.ParseUint(text, base, 32)
	if err != nil {
		return 0, false
	}
	return int64(int32(uint32(value))), true
}
