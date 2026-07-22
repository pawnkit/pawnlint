package walk

import (
	"strconv"
	"strings"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/lexer"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/syntax"
)

func (m *CompactModel) indexCompactNodeStates() {
	var index func(syntax.NodeID, bool, bool, bool)
	index = func(node syntax.NodeID, conditionalUncertain, inactive, ancestorError bool) {
		if m.Tree.Kind(node) != parser.KindSourceFile && m.endinput != 0 && m.Tree.Start(node) >= m.endinput {
			inactive = true
		}
		if m.Tree.HasError(node) || conditionalUncertain || ancestorError {
			m.states[node] |= compactUncertain
		}
		if inactive {
			m.states[node] |= compactInactive
		}
		childUncertain := conditionalUncertain
		childInactive := inactive
		childError := ancestorError || m.Tree.HasError(node)
		if m.Tree.Kind(node) == parser.KindSourceFile {
			childError = false
		}
		switch m.Tree.Kind(node) {
		case parser.KindConditionalBranch:
			childUncertain = childUncertain || m.branches[node] != branchActive
			childInactive = childInactive || m.branches[node] == branchInactive
		case parser.KindSharedConditional, parser.KindConditionalFunction, parser.KindConditionalSplice:
			childUncertain = true
		}
		for child := 0; child < m.Tree.ChildCount(node); child++ {
			index(m.Tree.Child(node, child), childUncertain, childInactive, childError)
		}
	}
	index(m.Root(), false, false, false)
}

func (m *CompactModel) indexCompactEndinput() {
	for _, node := range m.Tree.OfKind(parser.KindDirectiveEndinput) {
		if m.compactDirectiveActive(node) && (m.endinput == 0 || m.Tree.End(node) < m.endinput) {
			m.endinput = m.Tree.End(node)
		}
	}
}

func (m *CompactModel) indexCompactConditionalStates() {
	cursor := m.NewCompactDefineCursor()
	for _, region := range m.OfKind(parser.KindConditionalRegion) {
		reached := branchActive
		for child := 0; child < m.Tree.ChildCount(region); child++ {
			branch := m.Tree.Child(region, child)
			if m.Tree.Kind(branch) != parser.KindConditionalBranch {
				continue
			}
			m.branches[branch] = branchUncertain
			directive := m.Tree.Field(branch, "directive")
			if directive == syntax.NoNode || m.Tree.Kind(directive) == parser.KindDirectiveEndif {
				continue
			}
			if reached == branchInactive {
				m.branches[branch] = branchInactive
				continue
			}
			if m.Tree.Kind(directive) == parser.KindDirectiveElse {
				m.branches[branch] = reached
				reached = branchInactive
				continue
			}
			value, known := m.compactDirectiveValue(cursor, m.Tree.Field(directive, "condition"), m.Tree.Start(directive))
			if !known {
				m.branches[branch] = branchUncertain
				reached = branchUncertain
				continue
			}
			if value == 0 {
				m.branches[branch] = branchInactive
				continue
			}
			m.branches[branch] = reached
			reached = branchInactive
		}
	}
}

func (m *CompactModel) compactDirectiveValue(cursor *CompactDefineCursor, node syntax.NodeID, offset int) (int64, bool) {
	if !m.Tree.Valid(node) || m.Tree.HasError(node) {
		return 0, false
	}
	switch m.Tree.Kind(node) {
	case parser.KindIdentifier:
		return compilerConstant(m.Text(node))
	case parser.KindParenthesizedExpression:
		return m.compactDirectiveValue(cursor, m.Tree.Field(node, "expression"), offset)
	case parser.KindDefinedExpression:
		name := m.Tree.Field(node, "name")
		if name == syntax.NoNode {
			return 0, false
		}
		text := m.Text(name)
		if cursor.containsAt(offset, text) {
			return 1, true
		}
		if m.complete {
			return 0, true
		}
		return 0, false
	case parser.KindLiteral:
		if m.Tree.TokenKind(node) == token.KwNull {
			return 0, true
		}
		if m.Tree.TokenKind(node) != token.IntLiteral {
			return 0, false
		}
		text := strings.ReplaceAll(m.Tree.TokenText(node), "_", "")
		base := 10
		if strings.HasPrefix(text, "0x") || strings.HasPrefix(text, "0X") || strings.HasPrefix(text, "0b") || strings.HasPrefix(text, "0B") {
			base = 0
		}
		unsigned, err := strconv.ParseUint(text, base, 32)
		return int64(int32(uint32(unsigned))), err == nil
	case parser.KindUnaryExpression:
		value, ok := m.compactDirectiveValue(cursor, m.Tree.Field(node, "expression"), offset)
		if !ok {
			return 0, false
		}
		switch m.Tree.TokenKind(node) {
		case token.Bang:
			if value == 0 {
				return 1, true
			}
			return 0, true
		case token.Plus:
			return value, true
		case token.Minus:
			return -value, true
		default:
			return 0, false
		}
	case parser.KindBinaryExpression:
		left, leftOK := m.compactDirectiveValue(cursor, m.Tree.Field(node, "left"), offset)
		right, rightOK := m.compactDirectiveValue(cursor, m.Tree.Field(node, "right"), offset)
		if !leftOK || !rightOK {
			return 0, false
		}
		return directiveBinaryValue(m.Tree.TokenKind(node), left, right)
	default:
		return 0, false
	}
}

type CompactDefineCursor struct {
	model          *CompactModel
	offset         int
	directiveIndex int
	snapshotIndex  int
	values         defineValues
}

func (m *CompactModel) NewCompactDefineCursor() *CompactDefineCursor {
	cursor := &CompactDefineCursor{model: m}
	cursor.reset()
	return cursor
}

func (c *CompactDefineCursor) reset() {
	c.offset = -1
	c.directiveIndex = 0
	c.snapshotIndex = 0
	c.values.reset(c.model.defines.names)
}

func (c *CompactDefineCursor) containsAt(offset int, name string) bool {
	c.advance(offset)
	return c.values.contains(name)
}

func (c *CompactDefineCursor) advance(offset int) {
	if offset < c.offset {
		c.reset()
	}
	model := c.model
	for {
		directiveReady := c.directiveIndex < len(model.directives) && model.Tree.Start(model.directives[c.directiveIndex]) < offset && (model.endinput == 0 || model.Tree.Start(model.directives[c.directiveIndex]) < model.endinput)
		snapshotReady := c.snapshotIndex < len(model.snapshots) && model.snapshots[c.snapshotIndex].Offset < offset && (model.endinput == 0 || model.snapshots[c.snapshotIndex].Offset < model.endinput)
		if !directiveReady && !snapshotReady {
			break
		}
		if snapshotReady && (!directiveReady || model.snapshots[c.snapshotIndex].Offset <= model.Tree.Start(model.directives[c.directiveIndex])) {
			snapshot := model.snapshots[c.snapshotIndex]
			c.values.reset(snapshot.Defines)
			c.snapshotIndex++
			continue
		}
		node := model.directives[c.directiveIndex]
		c.directiveIndex++
		if !model.compactDirectiveActive(node) {
			continue
		}
		name := model.compactDirectiveName(node)
		if name == "" {
			continue
		}
		if model.Tree.Kind(node) == parser.KindDirectiveUndef {
			c.remove(name)
		} else {
			c.add(name)
		}
	}
	c.offset = offset
}

func (c *CompactDefineCursor) add(name string) {
	c.values.add(name)
}

func (c *CompactDefineCursor) remove(name string) {
	c.values.remove(name)
}

func (m *CompactModel) KnownDefinesAt(offset int) []string {
	return m.NewCompactDefineCursor().KnownDefinesAt(offset)
}

func (c *CompactDefineCursor) KnownDefinesAt(offset int) []string {
	c.advance(offset)
	return append([]string(nil), c.values.view()...)
}

func (c *CompactDefineCursor) KnownDefinesViewAt(offset int) []string {
	c.advance(offset)
	return c.values.view()
}

func (m *CompactModel) compactDirectiveActive(node syntax.NodeID) bool {
	for ancestor := m.Parent(node); ancestor != syntax.NoNode; ancestor = m.Parent(ancestor) {
		switch m.Tree.Kind(ancestor) {
		case parser.KindConditionalBranch:
			if m.branches[ancestor] != branchActive {
				return false
			}
		case parser.KindSharedConditional, parser.KindConditionalFunction, parser.KindConditionalSplice:
			return false
		}
	}
	return true
}

func (m *CompactModel) compactDirectiveName(node syntax.NodeID) string {
	if m.Tree.Kind(node) == parser.KindDirectiveDefine {
		return m.Text(m.Tree.Field(node, "name"))
	}
	if m.Tree.Kind(node) != parser.KindDirectiveUndef {
		return ""
	}
	seenDirective := false
	source := []byte(m.Text(node))
	for _, tok := range lexer.Tokenize(source) {
		if tok.Kind != token.Identifier {
			continue
		}
		text := tok.Text(source)
		if !seenDirective {
			seenDirective = text == "undef"
			continue
		}
		return text
	}
	return ""
}

func (m *CompactModel) IsInsideConditionalBranch(node syntax.NodeID) bool {
	for ancestor := m.Parent(node); ancestor != syntax.NoNode; ancestor = m.Parent(ancestor) {
		switch m.Tree.Kind(ancestor) {
		case parser.KindConditionalRegion, parser.KindConditionalBranch,
			parser.KindSharedConditional, parser.KindConditionalFunction,
			parser.KindConditionalSplice:
			return true
		}
	}
	return false
}

func (m *CompactModel) Uncertain(node syntax.NodeID) bool {
	return m != nil && m.Tree != nil && m.Tree.Valid(node) && m.states[node]&compactUncertain != 0
}

func (m *CompactModel) Inactive(node syntax.NodeID) bool {
	return m != nil && m.Tree != nil && m.Tree.Valid(node) && m.states[node]&compactInactive != 0
}
