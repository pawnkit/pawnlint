package cst

import (
	"bytes"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/source"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/internal/syntax"
)

type Model struct {
	pointer        *walk.Model
	compact        *walk.CompactModel
	compactOrigins []*token.Origin
}

type Node struct {
	model   *Model
	pointer *parser.Node
	id      syntax.NodeID
}

type Token struct {
	model *Model
	index int
}

type DefineCursor struct {
	pointer *walk.DefineCursor
	compact *walk.CompactDefineCursor
}

func Pointer(model *walk.Model) *Model {
	return &Model{pointer: model}
}

func Compact(model *walk.CompactModel) *Model {
	return &Model{compact: model}
}

func (m *Model) Source() []byte {
	if m == nil {
		return nil
	}
	if m.pointer != nil {
		return m.pointer.Source()
	}
	if m.compact != nil {
		return m.compact.Source()
	}
	return nil
}

func (m *Model) Root() Node {
	if m == nil {
		return Node{}
	}
	if m.pointer != nil {
		return Node{model: m, pointer: m.pointer.Root(), id: syntax.NoNode}
	}
	if m.compact != nil {
		return Node{model: m, id: m.compact.Root()}
	}
	return Node{}
}

func (m *Model) PointerNode(node *parser.Node) Node {
	if m == nil || m.pointer == nil || node == nil {
		return Node{}
	}
	return Node{model: m, pointer: node, id: syntax.NoNode}
}

func (m *Model) CompactNode(node syntax.NodeID) Node {
	if m == nil || m.compact == nil || !m.compact.Tree.Valid(node) {
		return Node{}
	}
	return Node{model: m, id: node}
}

func (m *Model) TokenCount() int {
	if m == nil {
		return 0
	}
	if m.pointer != nil && m.pointer.File != nil {
		return len(m.pointer.File.Tokens)
	}
	if m.compact != nil && m.compact.Tree.File() != nil {
		return len(m.compact.Tree.File().Tokens)
	}
	return 0
}

func (m *Model) Token(index int) Token {
	if m == nil || index < 0 || index >= m.TokenCount() {
		return Token{}
	}
	return Token{model: m, index: index}
}

func (m *Model) NewDefineCursor() *DefineCursor {
	if m == nil {
		return &DefineCursor{}
	}
	if m.pointer != nil {
		return &DefineCursor{pointer: m.pointer.NewDefineCursor()}
	}
	if m.compact != nil {
		return &DefineCursor{compact: m.compact.NewCompactDefineCursor()}
	}
	return &DefineCursor{}
}

func (c *DefineCursor) KnownDefinesAt(offset int) []string {
	if c == nil {
		return nil
	}
	if c.pointer != nil {
		return c.pointer.KnownDefinesAt(offset)
	}
	if c.compact != nil {
		return c.compact.KnownDefinesAt(offset)
	}
	return nil
}

func (c *DefineCursor) KnownDefinesViewAt(offset int) []string {
	if c == nil {
		return nil
	}
	if c.pointer != nil {
		return c.pointer.KnownDefinesViewAt(offset)
	}
	if c.compact != nil {
		return c.compact.KnownDefinesViewAt(offset)
	}
	return nil
}

func (m *Model) OfKind(kind parser.Kind) []Node {
	if m == nil {
		return nil
	}
	if m.pointer != nil {
		items := m.pointer.OfKind(kind)
		result := make([]Node, len(items))
		for index, item := range items {
			result[index] = Node{model: m, pointer: item, id: syntax.NoNode}
		}
		return result
	}
	if m.compact != nil {
		items := m.compact.OfKind(kind)
		result := make([]Node, len(items))
		for index, item := range items {
			result[index] = Node{model: m, id: item}
		}
		return result
	}
	return nil
}

func (m *Model) Parent(node Node) Node {
	if !node.Valid() || node.model != m {
		return Node{}
	}
	if m.pointer != nil {
		return Node{model: m, pointer: m.pointer.Parent(node.pointer), id: syntax.NoNode}
	}
	return Node{model: m, id: m.compact.Parent(node.id)}
}

func (m *Model) EnclosingFunction(node Node) Node {
	if !node.Valid() || node.model != m {
		return Node{}
	}
	if m.pointer != nil {
		return Node{model: m, pointer: m.pointer.EnclosingFunction(node.pointer), id: syntax.NoNode}
	}
	return Node{model: m, id: m.compact.EnclosingFunction(node.id)}
}

func (m *Model) Inactive(node Node) bool {
	if !node.Valid() || node.model != m {
		return false
	}
	if m.pointer != nil {
		return m.pointer.Inactive(node.pointer)
	}
	return m.compact.Inactive(node.id)
}

func (m *Model) Uncertain(node Node) bool {
	if !node.Valid() || node.model != m {
		return false
	}
	if m.pointer != nil {
		return m.pointer.Uncertain(node.pointer)
	}
	return m.compact.Uncertain(node.id)
}

func (m *Model) Range(node Node) source.Range {
	if !node.Valid() || node.model != m {
		return source.Range{}
	}
	if m.pointer != nil {
		return m.pointer.Range(node.pointer)
	}
	return m.compact.Range(node.id)
}

func (n Node) Valid() bool {
	if n.model == nil {
		return false
	}
	if n.model.pointer != nil {
		return n.pointer != nil
	}
	return n.model.compact != nil && n.model.compact.Tree.Valid(n.id)
}

func (n Node) Pointer() *parser.Node {
	return n.pointer
}

func (n Node) ID() syntax.NodeID {
	if n.model == nil || n.model.compact == nil {
		return syntax.NoNode
	}
	return n.id
}

func (n Node) Kind() parser.Kind {
	if !n.Valid() {
		return parser.KindInvalid
	}
	if n.pointer != nil {
		return n.pointer.Kind
	}
	return n.model.compact.Tree.Kind(n.id)
}

func (n Node) Start() int {
	if !n.Valid() {
		return 0
	}
	if n.pointer != nil {
		return n.pointer.Start
	}
	return n.model.compact.Tree.Start(n.id)
}

func (n Node) End() int {
	if !n.Valid() {
		return 0
	}
	if n.pointer != nil {
		return n.pointer.End
	}
	return n.model.compact.Tree.End(n.id)
}

func (n Node) HasError() bool {
	if !n.Valid() {
		return false
	}
	if n.pointer != nil {
		return n.pointer.HasError
	}
	return n.model.compact.Tree.HasError(n.id)
}

func (n Node) MissingSemi() bool {
	if !n.Valid() {
		return false
	}
	if n.pointer != nil {
		return n.pointer.MissingSemi
	}
	return n.model.compact.Tree.MissingSemi(n.id)
}

func (n Node) TokenKind() token.Kind {
	if !n.Valid() {
		return token.Invalid
	}
	if n.pointer != nil {
		return n.pointer.Tok.Kind
	}
	return n.model.compact.Tree.TokenKind(n.id)
}

func (n Node) TokenText() string {
	if !n.Valid() {
		return ""
	}
	if n.pointer != nil {
		return n.pointer.Tok.Text(n.model.pointer.Source())
	}
	return n.model.compact.Tree.TokenText(n.id)
}

func (n Node) Text() string {
	if !n.Valid() {
		return ""
	}
	if n.pointer != nil {
		return n.model.pointer.Text(n.pointer)
	}
	return n.model.compact.Text(n.id)
}

func (n Node) ChildCount() int {
	if !n.Valid() {
		return 0
	}
	if n.pointer != nil {
		return len(n.pointer.Children)
	}
	return n.model.compact.Tree.ChildCount(n.id)
}

func (n Node) Child(index int) Node {
	if !n.Valid() || index < 0 || index >= n.ChildCount() {
		return Node{}
	}
	if n.pointer != nil {
		return Node{model: n.model, pointer: n.pointer.Children[index], id: syntax.NoNode}
	}
	return Node{model: n.model, id: n.model.compact.Tree.Child(n.id, index)}
}

func (n Node) Field(name string) Node {
	if !n.Valid() {
		return Node{}
	}
	if n.pointer != nil {
		return Node{model: n.model, pointer: n.pointer.Field(name), id: syntax.NoNode}
	}
	return Node{model: n.model, id: n.model.compact.Tree.Field(n.id, name)}
}

func (n Node) Same(other Node) bool {
	if !n.Valid() || !other.Valid() || n.model != other.model {
		return false
	}
	if n.pointer != nil {
		return n.pointer == other.pointer
	}
	return n.id == other.id
}

func (n Node) HasChildToken(kind token.Kind) bool {
	for index := 0; index < n.ChildCount(); index++ {
		if n.Child(index).TokenKind() == kind {
			return true
		}
	}
	return false
}

func (n Node) Range() source.Range {
	if !n.Valid() {
		return source.Range{}
	}
	return n.model.Range(n)
}

func (t Token) Valid() bool {
	return t.model != nil && t.index >= 0 && t.index < t.model.TokenCount()
}

func (t Token) Kind() token.Kind {
	if !t.Valid() {
		return token.Invalid
	}
	if t.model.pointer != nil {
		return t.model.pointer.File.Tokens[t.index].Kind
	}
	return t.model.compact.Tree.File().Tokens[t.index].Kind
}

func (t Token) Start() int {
	return t.StartPosition().Offset
}

func (t Token) StartPosition() token.Position {
	if !t.Valid() {
		return token.Position{}
	}
	if t.model.pointer != nil {
		return t.model.pointer.File.Tokens[t.index].Start
	}
	position := t.model.compact.Tree.File().Tokens[t.index].Start
	return token.Position{Offset: int(position.Offset), Line: int(position.Line), Col: int(position.Col)}
}

func (t Token) End() int {
	return t.EndPosition().Offset
}

func (t Token) EndPosition() token.Position {
	if !t.Valid() {
		return token.Position{}
	}
	if t.model.pointer != nil {
		return t.model.pointer.File.Tokens[t.index].End
	}
	position := t.model.compact.Tree.File().Tokens[t.index].End
	return token.Position{Offset: int(position.Offset), Line: int(position.Line), Col: int(position.Col)}
}

func (t Token) Text() string {
	if !t.Valid() {
		return ""
	}
	if t.model.pointer != nil {
		return t.model.pointer.File.Tokens[t.index].Text(t.model.Source())
	}
	start, end := t.Start(), t.End()
	if start < 0 || end > len(t.model.Source()) || start > end {
		return ""
	}
	return string(t.model.Source()[start:end])
}

func (t Token) EndsLine() bool {
	if !t.Valid() {
		return false
	}
	if t.model.pointer != nil {
		for _, trivia := range t.model.pointer.File.Tokens[t.index].TrailingTrivia {
			if trivia.Kind == token.Newline {
				return true
			}
		}
	} else {
		file := t.model.compact.Tree.File()
		current := file.Tokens[t.index]
		end := current.TrailingStart + current.TrailingCount
		if end >= current.TrailingStart && end <= uint32(len(file.Trivia)) {
			for _, trivia := range file.Trivia[current.TrailingStart:end] {
				if trivia.Kind == token.Newline {
					return true
				}
			}
		}
	}
	start := t.End()
	end := len(t.model.Source())
	if t.index+1 < t.model.TokenCount() {
		end = t.model.Token(t.index + 1).Start()
	}
	if start < 0 || end < start || end > len(t.model.Source()) {
		return false
	}
	return bytes.ContainsAny(t.model.Source()[start:end], "\r\n")
}

func (t Token) Origin() *token.Origin {
	if !t.Valid() {
		return nil
	}
	if t.model.pointer != nil {
		return t.model.pointer.File.Tokens[t.index].Origin
	}
	file := t.model.compact.Tree.File()
	id := file.Tokens[t.index].Origin
	if id == 0 || id >= uint32(len(file.Origins)) {
		return nil
	}
	if t.model.compactOrigins == nil {
		t.model.compactOrigins = make([]*token.Origin, len(file.Origins))
	}
	var expand func(uint32) *token.Origin
	expand = func(current uint32) *token.Origin {
		if current == 0 || current >= uint32(len(file.Origins)) {
			return nil
		}
		if t.model.compactOrigins[current] != nil {
			return t.model.compactOrigins[current]
		}
		value := file.Origins[current]
		origin := &token.Origin{Span: token.Span{
			File:  value.File,
			Start: token.Position{Offset: int(value.Start.Offset), Line: int(value.Start.Line), Col: int(value.Start.Col)},
			End:   token.Position{Offset: int(value.End.Offset), Line: int(value.End.Line), Col: int(value.End.Col)},
		}}
		if value.Macro < uint32(len(file.MacroNames)) {
			origin.Macro = file.MacroNames[value.Macro]
		}
		t.model.compactOrigins[current] = origin
		origin.Parent = expand(value.Parent)
		return origin
	}
	return expand(id)
}
