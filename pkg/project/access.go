package project

import (
	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source"
	"github.com/pawnkit/pawnlint/internal/source/cst"
	"github.com/pawnkit/pawnlint/internal/source/walk"
)

type FunctionParameter struct {
	Tags     []string
	Variadic bool
	Known    bool
}

type NodeKey struct {
	file    *File
	pointer *parser.Node
	compact uint32
}

func (d Declaration) Valid() bool {
	return declarationSyntax(d).Valid()
}

func (d Declaration) Key() NodeKey {
	node := declarationSyntax(d)
	return NodeKey{file: d.File, pointer: node.Pointer(), compact: uint32(node.ID())}
}

func (d Declaration) NodeKind() parser.Kind {
	return declarationSyntax(d).Kind()
}

func (d Declaration) Start() int {
	return declarationSyntax(d).Start()
}

func (d Declaration) End() int {
	return declarationSyntax(d).End()
}

func (d Declaration) HasError() bool {
	return declarationSyntax(d).HasError()
}

func (d Declaration) Ambiguous() bool {
	return declarationSymbolAmbiguous(d)
}

func (d Declaration) Constant() bool {
	return declarationSymbolConstant(d)
}

func (d Declaration) Tags() []string {
	return append([]string(nil), declarationSymbolTags(d)...)
}

func (d Declaration) HasToken(kind token.Kind) bool {
	return declarationSyntax(d).HasChildToken(kind)
}

func (d Declaration) HasField(name string) bool {
	return declarationSyntax(d).Field(name).Valid()
}

func (d Declaration) FieldText(name string) string {
	return declarationSyntax(d).Field(name).Text()
}

func (d Declaration) Range() source.Range {
	if d.File == nil || d.File.Syntax == nil {
		return source.Range{}
	}
	return d.File.Syntax.Range(declarationSyntax(d))
}

func (d Declaration) FieldRange(name string) source.Range {
	if d.File == nil || d.File.Syntax == nil {
		return source.Range{}
	}
	return d.File.Syntax.Range(declarationSyntax(d).Field(name))
}

func (d Declaration) NameRange() source.Range {
	if d.File == nil || d.File.Syntax == nil {
		return source.Range{}
	}
	return d.File.Syntax.Range(declarationNameSyntax(d))
}

func (d Declaration) FunctionParameters() []FunctionParameter {
	list := declarationSyntax(d).Field("parameters")
	if !list.Valid() {
		return nil
	}
	var result []FunctionParameter
	for index := 0; index < list.ChildCount(); index++ {
		parameter := list.Child(index)
		if parameter.Kind() != parser.KindParameter {
			continue
		}
		if !parameter.Field("name").Valid() && parameter.TokenKind() == token.Ellipsis {
			result = append(result, FunctionParameter{Variadic: true, Known: true})
			continue
		}
		item := FunctionParameter{Tags: []string{""}}
		if d.Symbol != nil {
			for _, symbol := range d.File.Semantic.Symbols {
				if symbol.Kind != semantic.SymbolParameter || symbol.Decl != parameter.Pointer() {
					continue
				}
				if !symbol.Ambiguous {
					item.Known = true
					if len(symbol.Tags) != 0 {
						item.Tags = append([]string(nil), symbol.Tags...)
					}
				}
				break
			}
		} else if d.compactSymbol != nil {
			for _, symbol := range d.File.CompactSemantic.Symbols {
				if symbol.Kind != semantic.SymbolParameter || symbol.Decl != parameter.ID() {
					continue
				}
				if !symbol.Ambiguous {
					item.Known = true
					if len(symbol.Tags) != 0 {
						item.Tags = append([]string(nil), symbol.Tags...)
					}
				}
				break
			}
		}
		result = append(result, item)
	}
	return result
}

func (i *Include) Valid() bool {
	return i != nil && includeSyntax(i).Valid()
}

func (i *Include) Start() int {
	return includeSyntax(i).Start()
}

func (i *Include) Range() source.Range {
	if i == nil || !i.syntax.Valid() {
		return source.Range{}
	}
	return i.syntax.Range()
}

func (i *Include) PathRange() source.Range {
	if i == nil || !i.syntax.Valid() {
		return source.Range{}
	}
	return i.syntax.Field("path").Range()
}

func (f *File) LineTable() *source.LineTable {
	if f == nil || f.Syntax == nil {
		return nil
	}
	if f.Walk != nil {
		return f.Walk.LineTable
	}
	if f.CompactWalk != nil {
		return f.CompactWalk.LineTable
	}
	return nil
}

func (f *File) HasParseErrors() bool {
	return f == nil || fileHasParseErrors(f)
}

func (c Call) Valid() bool {
	return callSyntax(c).Valid()
}

func (c Call) Key() NodeKey {
	node := callSyntax(c)
	return NodeKey{file: c.File, pointer: node.Pointer(), compact: uint32(node.ID())}
}

func (c Call) FieldRange(name string) source.Range {
	return callSyntax(c).Field(name).Range()
}

func (d Declaration) PointerNode() *parser.Node {
	if d.Node != nil {
		return d.Node
	}
	return pointerNode(d.File, declarationSyntax(d))
}

func (c Call) PointerNode() *parser.Node {
	if c.Node != nil {
		return c.Node
	}
	return pointerNode(c.File, callSyntax(c))
}

func pointerNode(file *File, node cst.Node) *parser.Node {
	if file == nil || !node.Valid() {
		return nil
	}
	file.ensurePointerSyntax()
	return file.pointerNodes[nodeLocation{kind: node.Kind(), start: node.Start(), end: node.End()}]
}

func (f *File) ensurePointerSyntax() {
	if f == nil {
		return
	}
	f.pointerOnce.Do(func() {
		if f.Parsed == nil {
			f.Parsed = parser.Parse(f.Source)
		}
		if f.Walk == nil {
			lines := f.LineTable()
			f.syntaxIndex = walk.NewIndex(f.Parsed)
			f.Walk = walk.NewWithContext(f.Path, f.Parsed, f.defines.walk, f.snapshots, f.complete, lines, f.syntaxIndex)
		}
		if f.Semantic == nil {
			f.Semantic = semantic.Build(f.Parsed, f.Walk)
		}
		if f.CompactParsed != nil {
			f.compactNodes = make(map[nodeLocation]cst.Node)
			var compactIndex func(cst.Node)
			compactIndex = func(current cst.Node) {
				if !current.Valid() {
					return
				}
				f.compactNodes[nodeLocation{kind: current.Kind(), start: current.Start(), end: current.End()}] = current
				for index := 0; index < current.ChildCount(); index++ {
					compactIndex(current.Child(index))
				}
			}
			compactIndex(f.Syntax.Root())
		}
		f.pointerNodes = make(map[nodeLocation]*parser.Node)
		var index func(*parser.Node)
		index = func(current *parser.Node) {
			if current == nil {
				return
			}
			f.pointerNodes[nodeLocation{kind: current.Kind, start: current.Start, end: current.End}] = current
			for _, child := range current.Children {
				index(child)
			}
		}
		index(f.Parsed.Root)
	})
}

func (f *File) syntaxNode(node *parser.Node) cst.Node {
	if f == nil || node == nil || f.Syntax == nil {
		return cst.Node{}
	}
	if current := f.Syntax.PointerNode(node); current.Valid() {
		return current
	}
	f.ensurePointerSyntax()
	return f.compactNodes[nodeLocation{kind: node.Kind, start: node.Start, end: node.End}]
}

func includeSyntax(include *Include) cst.Node {
	if include == nil {
		return cst.Node{}
	}
	return include.syntax
}
