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
	if ctx.Flow != nil {
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
