package semantic

import (
	"strconv"
	"strings"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
)

func (m *Model) Eval(node *parser.Node) (int64, bool) {
	return m.eval(node, make(map[*Symbol]bool), nil)
}

func (m *Model) EvalWithValues(node *parser.Node, values map[*Symbol]int64) (int64, bool) {
	return m.eval(node, make(map[*Symbol]bool), values)
}

func (m *Model) eval(node *parser.Node, visiting map[*Symbol]bool, values map[*Symbol]int64) (int64, bool) {
	if node == nil || node.HasError || m.Walk.Uncertain(node) {
		return 0, false
	}
	switch node.Kind {
	case parser.KindLiteral:
		return m.evalLiteral(node)
	case parser.KindParenthesizedExpression:
		return m.eval(node.Field("expression"), visiting, values)
	case parser.KindUnaryExpression:
		value, ok := m.eval(node.Field("expression"), visiting, values)
		if !ok {
			return 0, false
		}
		switch node.Tok.Kind {
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
		left, leftOK := m.eval(node.Field("left"), visiting, values)
		right, rightOK := m.eval(node.Field("right"), visiting, values)
		if !leftOK || !rightOK {
			return 0, false
		}
		return evalBinary(node.Tok.Kind, left, right)
	case parser.KindTernaryExpression:
		condition, ok := m.eval(node.Field("condition"), visiting, values)
		if !ok {
			return 0, false
		}
		if condition != 0 {
			return m.eval(node.Field("consequence"), visiting, values)
		}
		return m.eval(node.Field("alternative"), visiting, values)
	case parser.KindIdentifier:
		if value, ok := builtinValue(m.Walk.Text(node)); ok && m.Resolve(node) == nil {
			return value, true
		}
		symbol := m.Resolve(node)
		if value, ok := values[symbol]; ok && symbol != nil {
			return value, true
		}
		if symbol == nil || !symbol.Constant {
			return 0, false
		}
		if value, ok := m.constantValues[symbol]; ok {
			return value, true
		}
		if symbol.Value == nil || visiting[symbol] {
			return 0, false
		}
		visiting[symbol] = true
		value, ok := m.eval(symbol.Value, visiting, values)
		delete(visiting, symbol)
		return value, ok
	default:
		return 0, false
	}
}

func builtinValue(name string) (int64, bool) {
	switch name {
	case "false", "EOS", "charmin":
		return 0, true
	case "true":
		return 1, true
	case "cellbits":
		return 32, true
	case "cellmax":
		return 2147483647, true
	case "cellmin":
		return -2147483648, true
	case "charbits":
		return 8, true
	case "charmax":
		return 255, true
	case "ucharmax":
		return 16777215, true
	default:
		return 0, false
	}
}

func (m *Model) evalLiteral(node *parser.Node) (int64, bool) {
	if node.Tok.Kind == token.KwNull {
		return 0, true
	}
	if node.Tok.Kind != token.IntLiteral {
		return 0, false
	}
	text := strings.ReplaceAll(node.Tok.Text(m.File.Source), "_", "")
	base := 10
	if strings.HasPrefix(text, "0x") || strings.HasPrefix(text, "0X") || strings.HasPrefix(text, "0b") || strings.HasPrefix(text, "0B") {
		base = 0
	}
	value, err := strconv.ParseUint(text, base, 32)
	if err == nil {
		return int64(int32(uint32(value))), true
	}
	return 0, false
}

func evalBinary(kind token.Kind, left, right int64) (int64, bool) {
	switch kind {
	case token.Plus:
		return cell(left + right), true
	case token.Minus:
		return cell(left - right), true
	case token.Star:
		return cell(left * right), true
	case token.Slash:
		if right == 0 {
			return 0, false
		}
		return cell(left / right), true
	case token.Percent:
		if right == 0 {
			return 0, false
		}
		return cell(left % right), true
	case token.Shl, token.Shr, token.Ushr:
		if right < 0 || right >= 32 {
			return 0, false
		}
		switch kind {
		case token.Shl:
			return int64(int32(uint32(left) << uint32(right))), true
		case token.Shr:
			return int64(int32(left) >> uint32(right)), true
		default:
			return int64(int32(uint32(left) >> uint32(right))), true
		}
	case token.Amp:
		return cell(left & right), true
	case token.Pipe:
		return cell(left | right), true
	case token.Caret:
		return cell(left ^ right), true
	case token.AndAnd:
		return truth(left != 0 && right != 0), true
	case token.OrOr:
		return truth(left != 0 || right != 0), true
	case token.Eq:
		return truth(left == right), true
	case token.NotEq:
		return truth(left != right), true
	case token.Lt:
		return truth(left < right), true
	case token.Gt:
		return truth(left > right), true
	case token.LtEq:
		return truth(left <= right), true
	case token.GtEq:
		return truth(left >= right), true
	default:
		return 0, false
	}
}

func cell(value int64) int64 {
	return int64(int32(value))
}

func truth(value bool) int64 {
	if value {
		return 1
	}
	return 0
}
