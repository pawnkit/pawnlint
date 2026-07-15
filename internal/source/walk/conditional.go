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
			value, known := m.directiveValue(directive.Field("condition"), directive.Start)
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

func (m *Model) directiveValue(node *parser.Node, offset int) (int64, bool) {
	if node == nil || node.HasError {
		return 0, false
	}
	switch node.Kind {
	case parser.KindParenthesizedExpression:
		return m.directiveValue(node.Field("expression"), offset)
	case parser.KindDefinedExpression:
		name := node.Field("name")
		if name == nil {
			return 0, false
		}
		_, known := m.knownDefinesAt(offset)[m.Text(name)]
		if known {
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
		value, ok := m.directiveValue(node.Field("expression"), offset)
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

func (m *Model) knownDefinesAt(offset int) map[string]struct{} {
	known := make(map[string]struct{}, len(m.defines))
	for _, name := range m.defines {
		if name != "" {
			known[name] = struct{}{}
		}
	}
	directiveIndex := 0
	snapshotIndex := 0
	for {
		for directiveIndex < len(m.directives) && m.directives[directiveIndex].Start >= offset {
			directiveIndex++
		}
		for snapshotIndex < len(m.snapshots) && m.snapshots[snapshotIndex].Offset >= offset {
			snapshotIndex++
		}
		if directiveIndex >= len(m.directives) && snapshotIndex >= len(m.snapshots) {
			break
		}
		if snapshotIndex < len(m.snapshots) && (directiveIndex >= len(m.directives) || m.snapshots[snapshotIndex].Offset <= m.directives[directiveIndex].Start) {
			clear(known)
			for _, name := range m.snapshots[snapshotIndex].Defines {
				if name != "" {
					known[name] = struct{}{}
				}
			}
			snapshotIndex++
			continue
		}
		node := m.directives[directiveIndex]
		directiveIndex++
		if !m.directiveActive(node) {
			continue
		}
		name := m.directiveName(node)
		if name == "" {
			continue
		}
		if node.Kind == parser.KindDirectiveUndef {
			delete(known, name)
		} else {
			known[name] = struct{}{}
		}
	}
	return known
}

func (m *Model) KnownDefinesAt(offset int) []string {
	known := m.knownDefinesAt(offset)
	names := make([]string, 0, len(known))
	for name := range known {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
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
