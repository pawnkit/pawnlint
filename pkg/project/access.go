package project

import (
	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source"
	"github.com/pawnkit/pawnlint/internal/source/cst"
)

type FunctionParameter struct {
	Tags     []string
	Variadic bool
	Known    bool
}

func (d Declaration) Valid() bool {
	return declarationSyntax(d).Valid()
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
		if d.File.Semantic != nil {
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
		} else if d.File.CompactSemantic != nil {
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

func includeSyntax(include *Include) cst.Node {
	if include == nil {
		return cst.Node{}
	}
	return include.syntax
}
