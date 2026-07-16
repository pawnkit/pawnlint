package walk

import (
	"sort"
	"strconv"
	"strings"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
)

func (m *Model) indexNodeStates() {
	var index func(*parser.Node, bool, bool, bool)
	index = func(node *parser.Node, conditionalUncertain, inactive, ancestorError bool) {
		if node.HasError || conditionalUncertain || ancestorError {
			m.uncertain[node] = true
		}
		if inactive {
			m.inactive[node] = true
		}
		childUncertain := conditionalUncertain
		childInactive := inactive
		childError := ancestorError || node.HasError
		if node.Kind == parser.KindSourceFile {
			childError = false
		}
		switch node.Kind {
		case parser.KindConditionalBranch:
			childUncertain = childUncertain || m.branches[node] != branchActive
			childInactive = childInactive || m.branches[node] == branchInactive
		case parser.KindSharedConditional, parser.KindConditionalFunction,
			parser.KindConditionalSplice:
			childUncertain = true
		}
		for _, child := range node.Children {
			index(child, childUncertain, childInactive, childError)
		}
	}
	index(m.File.Root, false, false, false)
}

func (m *Model) indexConditionalStates() {
	cursor := m.NewDefineCursor()
	for _, region := range m.byKind[parser.KindConditionalRegion] {
		reached := branchActive
		for _, branch := range region.Children {
			if branch.Kind != parser.KindConditionalBranch {
				continue
			}
			m.branches[branch] = branchUncertain
			directive := branch.Field("directive")
			if directive == nil || directive.Kind == parser.KindDirectiveEndif {
				continue
			}
			if reached == branchInactive {
				m.branches[branch] = branchInactive
				continue
			}
			if directive.Kind == parser.KindDirectiveElse {
				m.branches[branch] = reached
				reached = branchInactive
				continue
			}
			value, known := m.directiveValue(cursor, directive.Field("condition"), directive.Start)
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

func (m *Model) directiveValue(cursor *DefineCursor, node *parser.Node, offset int) (int64, bool) {
	if node == nil || node.HasError {
		return 0, false
	}
	switch node.Kind {
	case parser.KindParenthesizedExpression:
		return m.directiveValue(cursor, node.Field("expression"), offset)
	case parser.KindDefinedExpression:
		name := node.Field("name")
		if name == nil {
			return 0, false
		}
		known := cursor.definesAt(offset)
		index := sort.SearchStrings(known, m.Text(name))
		if index < len(known) && known[index] == m.Text(name) {
			return 1, true
		}
		if m.complete {
			return 0, true
		}
		return 0, false
	case parser.KindLiteral:
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
		unsigned, err := strconv.ParseUint(text, base, 32)
		return int64(int32(uint32(unsigned))), err == nil
	case parser.KindUnaryExpression:
		value, ok := m.directiveValue(cursor, node.Field("expression"), offset)
		if !ok {
			return 0, false
		}
		switch node.Tok.Kind {
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
	default:
		return 0, false
	}
}

type DefineCursor struct {
	model          *Model
	offset         int
	directiveIndex int
	snapshotIndex  int
	known          []string
}

func (m *Model) NewDefineCursor() *DefineCursor {
	cursor := &DefineCursor{model: m}
	cursor.reset()
	return cursor
}

func (c *DefineCursor) reset() {
	c.offset = -1
	c.directiveIndex = 0
	c.snapshotIndex = 0
	c.clearKnown(len(compilerDefines) + len(c.model.defines.names))
	for _, name := range compilerDefines {
		c.add(name)
	}
	for _, name := range c.model.defines.names {
		c.add(name)
	}
}

func (c *DefineCursor) clearKnown(capacity int) {
	if cap(c.known) < capacity {
		c.known = make([]string, 0, capacity)
		return
	}
	clear(c.known)
	c.known = c.known[:0]
}

func (c *DefineCursor) definesAt(offset int) []string {
	if offset < c.offset {
		c.reset()
	}
	m := c.model
	for {
		directiveReady := c.directiveIndex < len(m.directives) && m.directives[c.directiveIndex].Start < offset
		snapshotReady := c.snapshotIndex < len(m.snapshots) && m.snapshots[c.snapshotIndex].Offset < offset
		if !directiveReady && !snapshotReady {
			break
		}
		if snapshotReady && (!directiveReady || m.snapshots[c.snapshotIndex].Offset <= m.directives[c.directiveIndex].Start) {
			snapshot := m.snapshots[c.snapshotIndex]
			c.clearKnown(len(snapshot.Defines))
			for _, name := range snapshot.Defines {
				c.add(name)
			}
			c.snapshotIndex++
			continue
		}
		node := m.directives[c.directiveIndex]
		c.directiveIndex++
		if !m.directiveActive(node) {
			continue
		}
		name := m.directiveName(node)
		if name == "" {
			continue
		}
		if node.Kind == parser.KindDirectiveUndef {
			c.remove(name)
		} else {
			c.add(name)
		}
	}
	c.offset = offset
	return c.known
}

func (c *DefineCursor) add(name string) {
	if name == "" {
		return
	}
	index := sort.SearchStrings(c.known, name)
	if index < len(c.known) && c.known[index] == name {
		return
	}
	c.known = append(c.known, "")
	copy(c.known[index+1:], c.known[index:])
	c.known[index] = name
}

func (c *DefineCursor) remove(name string) {
	index := sort.SearchStrings(c.known, name)
	if index >= len(c.known) || c.known[index] != name {
		return
	}
	copy(c.known[index:], c.known[index+1:])
	c.known = c.known[:len(c.known)-1]
}

func (m *Model) KnownDefinesAt(offset int) []string {
	return m.NewDefineCursor().KnownDefinesAt(offset)
}

func (c *DefineCursor) KnownDefinesAt(offset int) []string {
	known := c.definesAt(offset)
	return append([]string(nil), known...)
}

func (m *Model) directiveActive(node *parser.Node) bool {
	for _, ancestor := range m.Ancestors(node) {
		switch ancestor.Kind {
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

func (m *Model) directiveName(node *parser.Node) string {
	if node.Kind == parser.KindDirectiveDefine {
		return m.Text(node.Field("name"))
	}
	if node.Kind != parser.KindDirectiveUndef || m.File == nil {
		return ""
	}
	seenDirective := false
	start := sort.Search(len(m.File.Tokens), func(index int) bool {
		return m.File.Tokens[index].End.Offset > node.Start
	})
	for index := start; index < len(m.File.Tokens); index++ {
		tok := m.File.Tokens[index]
		if tok.Start.Offset >= node.End {
			break
		}
		if tok.Start.Offset < node.Start || tok.End.Offset > node.End {
			continue
		}
		if tok.Kind != token.Identifier {
			continue
		}
		text := tok.Text(m.File.Source)
		if !seenDirective {
			seenDirective = text == "undef"
			continue
		}
		return text
	}
	return ""
}

func (m *Model) IsInsideConditionalBranch(n *parser.Node) bool {
	for _, a := range m.Ancestors(n) {
		switch a.Kind {
		case parser.KindConditionalRegion, parser.KindConditionalBranch,
			parser.KindSharedConditional, parser.KindConditionalFunction,
			parser.KindConditionalSplice:
			return true
		}
	}
	return false
}

func (m *Model) Uncertain(n *parser.Node) bool {
	if m == nil || n == nil {
		return false
	}
	return m.uncertain[n]
}

func (m *Model) Inactive(n *parser.Node) bool {
	if m == nil || n == nil {
		return false
	}
	return m.inactive[n]
}
