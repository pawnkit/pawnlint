package project

import (
	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/source"
	"github.com/pawnkit/pawnlint/internal/source/cst"
)

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

func includeSyntax(include *Include) cst.Node {
	if include == nil {
		return cst.Node{}
	}
	return include.syntax
}
