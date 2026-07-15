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
	File     *File
	Report   func(diagnostic.Diagnostic)
	Level    AnalysisLevel
	Walk     *walk.Model
	Tokens   func(k token.Kind) []*token.Token
	Supp     []suppress.Directive
	Known    map[string]struct{}
	PerRule  map[string]map[string]any
	Semantic *semantic.Model
	Flow     *controlflow.Model
	Project  *project.Model
	Target   string
	API      *api.Metadata
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
	if ctx.Semantic != nil {
		return ctx.Semantic.Eval(node)
	}
	return 0, false
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
