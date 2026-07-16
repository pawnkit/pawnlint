package project

import (
	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/source/cst"
	"github.com/pawnkit/pawnlint/internal/syntax"
)

func (m *Model) Eval(file *File, node *parser.Node) (int64, bool) {
	if m == nil || file == nil || file.Semantic == nil || node == nil {
		return 0, false
	}
	return m.evalSyntax(file, file.syntaxNode(node), make(map[declarationID]bool))
}

func (m *Model) evalSyntax(file *File, node cst.Node, visiting map[declarationID]bool) (int64, bool) {
	resolve := func(identifier cst.Node) (int64, bool) {
		declaration, ok := m.resolveSyntax(file, identifier)
		if !ok || !declarationSymbolConstant(declaration) || declarationSymbolAmbiguous(declaration) {
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
	}
	if node.Pointer() != nil && file.Semantic != nil {
		return file.Semantic.EvalWithResolver(node.Pointer(), func(identifier *parser.Node) (int64, bool) {
			return resolve(file.syntaxNode(identifier))
		})
	}
	return file.CompactSemantic.EvalWithResolver(node.ID(), func(identifier syntax.NodeID) (int64, bool) {
		return resolve(file.Syntax.CompactNode(identifier))
	})
}

func (m *Model) declarationValue(declaration Declaration, visiting map[declarationID]bool) (int64, bool) {
	if declaration.Symbol != nil {
		if value, ok := declaration.File.Semantic.ConstantValue(declaration.Symbol); ok {
			return value, true
		}
		if declaration.Symbol.Value != nil {
			return m.evalSyntax(declaration.File, declaration.File.Syntax.PointerNode(declaration.Symbol.Value), visiting)
		}
	} else if declaration.compactSymbol != nil {
		if value, ok := declaration.File.CompactSemantic.ConstantValue(declaration.compactSymbol); ok {
			return value, true
		}
		if declaration.compactSymbol.Value != syntax.NoNode {
			return m.evalSyntax(declaration.File, declaration.File.Syntax.CompactNode(declaration.compactSymbol.Value), visiting)
		}
	}
	if declarationSyntax(declaration).Kind() == parser.KindEnumEntry {
		return m.enumEntryValue(declaration, visiting)
	}
	return 0, false
}

func enclosingEnumBody(file *File, entry cst.Node) (body, declaration cst.Node) {
	node := entry
	for {
		parent := file.Syntax.Parent(node)
		if !parent.Valid() {
			return cst.Node{}, cst.Node{}
		}
		switch parent.Kind() {
		case parser.KindConditionalRegion, parser.KindConditionalBranch:
			node = parent
			continue
		case parser.KindBlock:
			enum := file.Syntax.Parent(parent)
			if enum.Valid() && enum.Kind() == parser.KindEnumDeclaration {
				return parent, enum
			}
			return cst.Node{}, cst.Node{}
		default:
			return cst.Node{}, cst.Node{}
		}
	}
}

func enumEntryList(body cst.Node) []cst.Node {
	var entries []cst.Node
	var collect func(cst.Node)
	collect = func(node cst.Node) {
		switch node.Kind() {
		case parser.KindConditionalRegion, parser.KindConditionalBranch:
			for index := 0; index < node.ChildCount(); index++ {
				collect(node.Child(index))
			}
		case parser.KindEnumEntry:
			entries = append(entries, node)
		}
	}
	for index := 0; index < body.ChildCount(); index++ {
		collect(body.Child(index))
	}
	return entries
}

func (m *Model) enumEntryValue(target Declaration, visiting map[declarationID]bool) (int64, bool) {
	body, declaration := enclosingEnumBody(target.File, declarationSyntax(target))
	if !body.Valid() || !declaration.Valid() || declaration.Field("increment").Valid() || target.File.Syntax.Uncertain(declaration) {
		return 0, false
	}
	current := int64(0)
	for _, entry := range enumEntryList(body) {
		if target.File.Syntax.Inactive(entry) {
			continue
		}
		if entry.HasError() || target.File.Syntax.Uncertain(entry) {
			return 0, false
		}
		if explicit := entry.Field("value"); explicit.Valid() {
			value, ok := m.evalSyntax(target.File, explicit, visiting)
			if !ok {
				return 0, false
			}
			current = value
		}
		if entry.Same(declarationSyntax(target)) {
			return current, true
		}
		width := int64(1)
		for index := 0; index < entry.ChildCount(); index++ {
			child := entry.Child(index)
			if child.Kind() != parser.KindDimension {
				continue
			}
			size, ok := m.evalSyntax(target.File, child.Field("size"), visiting)
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
	if ok && !declarationSymbolAmbiguous(declaration) && len(declarationSymbolTags(declaration)) != 0 {
		return append([]string(nil), declarationSymbolTags(declaration)...)
	}
	variants := m.FunctionVariants(file, node)
	if len(variants) == 0 || len(declarationSymbolTags(variants[0])) == 0 {
		return nil
	}
	tags := declarationSymbolTags(variants[0])
	for _, variant := range variants[1:] {
		if !sameProjectTags(tags, declarationSymbolTags(variant)) {
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
