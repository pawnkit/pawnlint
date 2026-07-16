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
			m.states[node] |= nodeUncertain
		}
		if inactive {
			m.states[node] |= nodeInactive
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
	for _, region := range m.index.byKind[parser.KindConditionalRegion] {
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
		if cursor.containsAt(offset, m.Text(name)) {
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
	values         defineValues
}

type defineValues struct {
	base         []string
	added        []string
	removed      []string
	materialized []string
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
	c.values.reset(c.model.defines.names)
}

func (v *defineValues) reset(names []string) {
	v.base = names
	v.added = v.added[:0]
	v.removed = v.removed[:0]
	v.materialized = nil
}

func (c *DefineCursor) containsAt(offset int, name string) bool {
	c.advance(offset)
	return c.values.contains(name)
}

func (c *DefineCursor) advance(offset int) {
	if offset < c.offset {
		c.reset()
	}
	m := c.model
	for {
		directiveReady := c.directiveIndex < len(m.index.directives) && m.index.directives[c.directiveIndex].Start < offset
		snapshotReady := c.snapshotIndex < len(m.snapshots) && m.snapshots[c.snapshotIndex].Offset < offset
		if !directiveReady && !snapshotReady {
			break
		}
		if snapshotReady && (!directiveReady || m.snapshots[c.snapshotIndex].Offset <= m.index.directives[c.directiveIndex].Start) {
			snapshot := m.snapshots[c.snapshotIndex]
			c.values.reset(snapshot.Defines)
			c.snapshotIndex++
			continue
		}
		node := m.index.directives[c.directiveIndex]
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
}

func (c *DefineCursor) add(name string) {
	c.values.add(name)
}

func (c *DefineCursor) remove(name string) {
	c.values.remove(name)
}

func (v *defineValues) contains(name string) bool {
	if containsSorted(v.removed, name) {
		return false
	}
	return containsSorted(v.added, name) || containsSorted(v.base, name)
}

func (v *defineValues) add(name string) {
	if name == "" {
		return
	}
	if removeSorted(&v.removed, name) {
		v.materialized = nil
		return
	}
	if containsSorted(v.base, name) || containsSorted(v.added, name) {
		return
	}
	insertSorted(&v.added, name)
	v.materialized = nil
}

func (v *defineValues) remove(name string) {
	if removeSorted(&v.added, name) {
		v.materialized = nil
		return
	}
	if !containsSorted(v.base, name) || containsSorted(v.removed, name) {
		return
	}
	insertSorted(&v.removed, name)
	v.materialized = nil
}

func (v *defineValues) view() []string {
	if len(v.added) == 0 && len(v.removed) == 0 {
		return v.base
	}
	if v.materialized != nil {
		return v.materialized
	}
	known := make([]string, 0, len(v.base)+len(v.added)-len(v.removed))
	baseIndex := 0
	addedIndex := 0
	removedIndex := 0
	for baseIndex < len(v.base) || addedIndex < len(v.added) {
		for baseIndex < len(v.base) {
			for removedIndex < len(v.removed) && v.removed[removedIndex] < v.base[baseIndex] {
				removedIndex++
			}
			if removedIndex >= len(v.removed) || v.removed[removedIndex] != v.base[baseIndex] {
				break
			}
			baseIndex++
		}
		if baseIndex < len(v.base) && (addedIndex >= len(v.added) || v.base[baseIndex] < v.added[addedIndex]) {
			known = append(known, v.base[baseIndex])
			baseIndex++
		} else if addedIndex < len(v.added) {
			known = append(known, v.added[addedIndex])
			addedIndex++
		}
	}
	v.materialized = known
	return v.materialized
}

func containsSorted(values []string, name string) bool {
	index := sort.SearchStrings(values, name)
	return index < len(values) && values[index] == name
}

func insertSorted(values *[]string, name string) {
	index := sort.SearchStrings(*values, name)
	*values = append(*values, "")
	copy((*values)[index+1:], (*values)[index:])
	(*values)[index] = name
}

func removeSorted(values *[]string, name string) bool {
	index := sort.SearchStrings(*values, name)
	if index >= len(*values) || (*values)[index] != name {
		return false
	}
	copy((*values)[index:], (*values)[index+1:])
	*values = (*values)[:len(*values)-1]
	return true
}

func (m *Model) KnownDefinesAt(offset int) []string {
	return m.NewDefineCursor().KnownDefinesAt(offset)
}

func (c *DefineCursor) KnownDefinesAt(offset int) []string {
	c.advance(offset)
	return append([]string(nil), c.values.view()...)
}

func (c *DefineCursor) KnownDefinesViewAt(offset int) []string {
	c.advance(offset)
	return c.values.view()
}

func (m *Model) directiveActive(node *parser.Node) bool {
	for ancestor := m.Parent(node); ancestor != nil; ancestor = m.Parent(ancestor) {
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
	for ancestor := m.Parent(n); ancestor != nil; ancestor = m.Parent(ancestor) {
		switch ancestor.Kind {
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
	return m.states[n]&nodeUncertain != 0
}

func (m *Model) Inactive(n *parser.Node) bool {
	if m == nil || n == nil {
		return false
	}
	return m.states[n]&nodeInactive != 0
}
