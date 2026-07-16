package lint

import (
	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/api"
	"github.com/pawnkit/pawnlint/internal/controlflow"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint/suppress"
	"github.com/pawnkit/pawnlint/pkg/project"
)

type AnalysisLevel uint8

const (
	SyntaxAnalysis AnalysisLevel = iota

	SemanticAnalysis

	ControlFlowAnalysis

	ProjectAnalysis
)

type Stability uint8

const (
	StabilityStable Stability = iota
	StabilityPreview
)

func (s Stability) String() string {
	if s == StabilityPreview {
		return "preview"
	}
	return "stable"
}

type Metadata struct {
	ID              string
	Name            string
	Summary         string
	Explanation     string
	Category        diagnostic.Category
	DefaultSeverity diagnostic.Severity
	AnalysisLevel   AnalysisLevel
	Stability       Stability
	DefaultEnabled  bool
	Fixable         bool
	Tags            []string
	Options         []Option
	ConfigExample   string
}

type File struct {
	Path      string
	Source    []byte
	Parsed    *parser.File
	LineTable *source.LineTable
}

type Context struct {
	File        *File
	Report      func(diagnostic.Diagnostic)
	Level       AnalysisLevel
	Walk        *walk.Model
	Tokens      func(k token.Kind) []*token.Token
	Supp        []suppress.Directive
	Known       map[string]struct{}
	PerRule     map[string]map[string]any
	Semantic    *semantic.Model
	Flow        *controlflow.Model
	Project     *project.Model
	ProjectFile *project.File
	Target      string
	API         *api.Metadata
}

func (ctx *Context) Eval(node *parser.Node) (int64, bool) {
	if ctx == nil {
		return 0, false
	}
	if ctx.Flow != nil && ctx.Level >= ControlFlowAnalysis {
		if value, ok := ctx.Flow.Eval(node); ok {
			return value, true
		}
	}
	return ctx.Constant(node)
}

func (ctx *Context) Constant(node *parser.Node) (int64, bool) {
	if ctx == nil {
		return 0, false
	}
	if ctx.Project != nil && ctx.ProjectFile != nil {
		if value, ok := ctx.Project.Eval(ctx.ProjectFile, node); ok {
			return value, true
		}
	}
	if ctx.Semantic != nil {
		return ctx.Semantic.Eval(node)
	}
	return 0, false
}

func (ctx *Context) ExpressionTags(node *parser.Node) []string {
	if ctx == nil {
		return nil
	}
	if ctx.Project != nil && ctx.ProjectFile != nil {
		if tags := ctx.Project.ExpressionTags(ctx.ProjectFile, node); len(tags) != 0 {
			return tags
		}
	}
	if ctx.Semantic != nil {
		return ctx.Semantic.ExpressionTags(node)
	}
	return nil
}

func (ctx *Context) ExpressionTag(node *parser.Node) (string, bool) {
	tags := ctx.ExpressionTags(node)
	if len(tags) != 1 || tags[0] == "" {
		return "", false
	}
	return tags[0], true
}

func (ctx *Context) Pure(node *parser.Node) bool {
	if ctx == nil || ctx.Semantic == nil || node == nil || node.HasError || ctx.Walk.Uncertain(node) {
		return false
	}
	if ctx.Semantic.Pure(node) {
		return true
	}
	for node.Kind == parser.KindParenthesizedExpression {
		node = node.Field("expression")
		if node == nil {
			return false
		}
	}
	switch node.Kind {
	case parser.KindCallExpression:
		if _, pure := ctx.PureCall(node); !pure {
			return false
		}
		arguments := node.Field("arguments")
		if arguments == nil {
			return true
		}
		for _, argument := range arguments.Children {
			if !ctx.Pure(argument) {
				return false
			}
		}
		return true
	case parser.KindUnaryExpression:
		if node.Tok.Kind == token.PlusPlus || node.Tok.Kind == token.MinusMinus {
			return false
		}
	case parser.KindBinaryExpression, parser.KindTernaryExpression, parser.KindSubscriptExpression,
		parser.KindSizeofExpression, parser.KindTagofExpression, parser.KindTaggedExpression:
	default:
		return false
	}
	for _, child := range node.Children {
		if !ctx.Pure(child) {
			return false
		}
	}
	return true
}

func (ctx *Context) PureCall(call *parser.Node) (string, bool) {
	if ctx == nil || call == nil || call.Kind != parser.KindCallExpression {
		return "", false
	}
	callee := call.Field("function")
	if callee == nil || callee.Kind != parser.KindIdentifier || callee.Tok.Origin != nil {
		return "", false
	}
	name := ctx.Walk.Text(callee)
	if ctx.Project != nil && ctx.ProjectFile != nil {
		variants := ctx.Project.FunctionVariants(ctx.ProjectFile, callee)
		if len(variants) != 0 {
			projectFunction := false
			for _, declaration := range variants {
				if declaration.Kind != semantic.SymbolFunction || !declaration.Valid() {
					return "", false
				}
				if declaration.HasToken(token.KwNative) {
					continue
				}
				projectFunction = true
				effects, known := ctx.Project.FunctionEffects(declaration)
				if !known || !effects.Complete || !effects.Pure {
					return "", false
				}
			}
			if projectFunction {
				return name, true
			}
		}
		for _, declaration := range ctx.Project.Declarations[name] {
			if declaration.Kind == semantic.SymbolFunction && len(variants) == 0 {
				return "", false
			}
		}
	}
	if native, ok := ctx.Natives()[name]; ok && native.Pure {
		return name, true
	}
	if function, ok := ctx.Functions()[name]; ok && function.Pure {
		return name, true
	}
	return "", false
}

func (ctx *Context) Natives() map[string]api.Native {
	if ctx != nil && ctx.API != nil {
		return ctx.API.Natives
	}
	target := ""
	if ctx != nil {
		target = ctx.Target
	}
	return api.Natives(target)
}

func (ctx *Context) Callbacks() map[string]api.Callback {
	if ctx != nil && ctx.API != nil {
		return ctx.API.Callbacks
	}
	target := ""
	if ctx != nil {
		target = ctx.Target
	}
	return api.Callbacks(target)
}

func (ctx *Context) Functions() map[string]api.Function {
	if ctx != nil && ctx.API != nil {
		return ctx.API.Functions
	}
	return nil
}

func (ctx *Context) Constants() map[string]api.Constant {
	if ctx != nil && ctx.API != nil {
		return ctx.API.Constants
	}
	target := ""
	if ctx != nil {
		target = ctx.Target
	}
	return api.Constants(target)
}
