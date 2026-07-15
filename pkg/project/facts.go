package project

import (
	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/source/walk"
)

func (m *Model) Eval(file *File, node *parser.Node) (int64, bool) {
	if m == nil || file == nil || file.Semantic == nil || node == nil {
		return 0, false
	}
	return m.eval(file, node, make(map[declarationID]bool))
}

func (m *Model) eval(file *File, node *parser.Node, visiting map[declarationID]bool) (int64, bool) {
	return file.Semantic.EvalWithResolver(node, func(identifier *parser.Node) (int64, bool) {
		declaration, ok := m.Resolve(file, identifier)
		if !ok || declaration.Symbol == nil || !declaration.Symbol.Constant || declaration.Symbol.Ambiguous {
			return 0, false
		}
		key := declarationKey(declaration)
		if visiting[key] {
			return 0, false
		}
		visiting[key] = true
		value, known := m.declarationValue(declaration, visiting)
		delete(visiting, key)
		return value, known
	})
}

func (m *Model) declarationValue(declaration Declaration, visiting map[declarationID]bool) (int64, bool) {
	if value, ok := declaration.File.Semantic.ConstantValue(declaration.Symbol); ok {
		return value, true
	}
	if declaration.Symbol.Value != nil {
		return m.eval(declaration.File, declaration.Symbol.Value, visiting)
	}
	if declaration.Node.Kind == parser.KindEnumEntry {
		return m.enumEntryValue(declaration, visiting)
	}
	return 0, false
}

func enclosingEnumBody(w *walk.Model, entry *parser.Node) (body, declaration *parser.Node) {
	node := entry
	for {
		parent := w.Parent(node)
		if parent == nil {
			return nil, nil
		}
		switch parent.Kind {
		case parser.KindConditionalRegion, parser.KindConditionalBranch:
			node = parent
			continue
		case parser.KindBlock:
			enum := w.Parent(parent)
			if enum != nil && enum.Kind == parser.KindEnumDeclaration {
				return parent, enum
			}
			return nil, nil
		default:
			return nil, nil
		}
	}
}

func enumEntryList(body *parser.Node) []*parser.Node {
	var entries []*parser.Node
	var collect func(*parser.Node)
	collect = func(node *parser.Node) {
		switch node.Kind {
		case parser.KindConditionalRegion, parser.KindConditionalBranch:
			for _, child := range node.Children {
				collect(child)
			}
		case parser.KindEnumEntry:
			entries = append(entries, node)
		}
	}
	for _, child := range body.Children {
		collect(child)
	}
	return entries
}

func (m *Model) enumEntryValue(target Declaration, visiting map[declarationID]bool) (int64, bool) {
	body, declaration := enclosingEnumBody(target.File.Walk, target.Node)
	if body == nil || declaration == nil || declaration.Field("increment") != nil || target.File.Walk.Uncertain(declaration) {
		return 0, false
	}
	current := int64(0)
	for _, entry := range enumEntryList(body) {
		if target.File.Walk.Inactive(entry) {
			continue
		}
		if entry.HasError || target.File.Walk.Uncertain(entry) {
			return 0, false
		}
		if explicit := entry.Field("value"); explicit != nil {
			value, ok := m.eval(target.File, explicit, visiting)
			if !ok {
				return 0, false
			}
			current = value
		}
		if entry == target.Node {
			return current, true
		}
		width := int64(1)
		for _, child := range entry.Children {
			if child.Kind != parser.KindDimension {
				continue
			}
			size, ok := m.eval(target.File, child.Field("size"), visiting)
			if !ok || size <= 0 {
				return 0, false
			}
			width = projectCell(width * size)
			if width <= 0 {
				return 0, false
			}
		}
		current = projectCell(current + width)
	}
	return 0, false
}

func (m *Model) ExpressionTags(file *File, node *parser.Node) []string {
	if m == nil || file == nil || file.Semantic == nil {
		return nil
	}
	node = unwrapProjectExpression(node)
	if node == nil || node.HasError || file.Walk.Uncertain(node) {
		return nil
	}
	if tags := file.Semantic.ExpressionTags(node); len(tags) != 0 {
		return tags
	}
	switch node.Kind {
	case parser.KindIdentifier:
		return m.resolvedTags(file, node)
	case parser.KindCallExpression:
		return m.resolvedTags(file, unwrapProjectExpression(node.Field("function")))
	case parser.KindSubscriptExpression:
		return m.ExpressionTags(file, node.Field("array"))
	case parser.KindAssignmentExpression:
		return m.ExpressionTags(file, node.Field("left"))
	case parser.KindUpdateExpression:
		return m.ExpressionTags(file, node.Field("expression"))
	case parser.KindUnaryExpression:
		if node.Tok.Kind == token.Bang {
			return []string{"bool"}
		}
		return m.ExpressionTags(file, node.Field("expression"))
	case parser.KindBinaryExpression:
		switch node.Tok.Kind {
		case token.Eq, token.NotEq, token.Lt, token.Gt, token.LtEq, token.GtEq, token.AndAnd, token.OrOr:
			return []string{"bool"}
		case token.Dot:
			return nil
		}
		left := m.ExpressionTags(file, node.Field("left"))
		right := m.ExpressionTags(file, node.Field("right"))
		if sameProjectTags(left, right) {
			return left
		}
	case parser.KindTernaryExpression:
		left := m.ExpressionTags(file, node.Field("consequence"))
		right := m.ExpressionTags(file, node.Field("alternative"))
		if sameProjectTags(left, right) {
			return left
		}
	case parser.KindExpressionList:
		if len(node.Children) != 0 {
			return m.ExpressionTags(file, node.Children[len(node.Children)-1])
		}
	}
	return nil
}

func (m *Model) ExpressionTag(file *File, node *parser.Node) (string, bool) {
	tags := m.ExpressionTags(file, node)
	if len(tags) != 1 || tags[0] == "" {
		return "", false
	}
	return tags[0], true
}

func (m *Model) resolvedTags(file *File, node *parser.Node) []string {
	declaration, ok := m.Resolve(file, node)
	if ok && declaration.Symbol != nil && !declaration.Symbol.Ambiguous && len(declaration.Symbol.Tags) != 0 {
		return append([]string(nil), declaration.Symbol.Tags...)
	}
	variants := m.FunctionVariants(file, node)
	if len(variants) == 0 || variants[0].Symbol == nil || len(variants[0].Symbol.Tags) == 0 {
		return nil
	}
	tags := variants[0].Symbol.Tags
	for _, variant := range variants[1:] {
		if variant.Symbol == nil || !sameProjectTags(tags, variant.Symbol.Tags) {
			return nil
		}
	}
	return append([]string(nil), tags...)
}

func unwrapProjectExpression(node *parser.Node) *parser.Node {
	for node != nil && node.Kind == parser.KindParenthesizedExpression {
		node = node.Field("expression")
	}
	return node
}

func sameProjectTags(left, right []string) bool {
	if len(left) == 0 || len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func projectCell(value int64) int64 {
	return int64(int32(value))
}
