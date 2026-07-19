package lint

import (
	"bytes"
	"fmt"
	"time"

	parser "github.com/pawnkit/pawn-parser"
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

type Engine struct {
	Reg           *Registrar
	Defines       []string
	Target        string
	Project       *project.Model
	API           *api.Metadata
	ObserveTiming func(TimingEvent)
}

type TimingStage string

const (
	TimingParse       TimingStage = "parse"
	TimingSemantic    TimingStage = "semantic"
	TimingControlFlow TimingStage = "control-flow"
	TimingRule        TimingStage = "rule"
)

type TimingEvent struct {
	Stage    TimingStage
	RuleID   string
	Duration time.Duration
}

func NewEngine(reg *Registrar) *Engine {
	return &Engine{Reg: reg}
}

const SuppressionID = "unknown-suppression"

const ParseErrorID = "parse-error"

const InternalErrorID = "internal-error"

const DeprecatedRuleID = "deprecated-rule-id"

func levelAllowed(l AnalysisLevel, max AnalysisLevel) bool {
	return l <= max
}

func (e *Engine) LintFile(path string, src []byte, maxLevel AnalysisLevel, ruleSet map[string]diagnostic.Severity, known map[string]struct{}, perRule map[string]map[string]any) []diagnostic.Diagnostic {
	return e.lintFileSafe(path, src, nil, maxLevel, ruleSet, known, perRule)
}

func (e *Engine) LintProjectFile(projectFile *project.File, maxLevel AnalysisLevel, ruleSet map[string]diagnostic.Severity, known map[string]struct{}, perRule map[string]map[string]any) []diagnostic.Diagnostic {
	if projectFile == nil {
		return nil
	}
	return e.lintFileSafe(projectFile.Path, projectFile.Source, projectFile, maxLevel, ruleSet, known, perRule)
}

// lintFileSafe keeps one bad file from stopping a multi-file run.
func (e *Engine) lintFileSafe(path string, src []byte, contextFile *project.File, maxLevel AnalysisLevel, ruleSet map[string]diagnostic.Severity, known map[string]struct{}, perRule map[string]map[string]any) (diagnostics []diagnostic.Diagnostic) {
	defer func() {
		if failed := recover(); failed != nil {
			diagnostics = []diagnostic.Diagnostic{{
				RuleID:   InternalErrorID,
				Severity: diagnostic.SeverityError,
				Category: diagnostic.CategoryCorrectness,
				Message:  fmt.Sprintf("analysis failed: %v", failed),
				Filename: path,
				Range:    source.NewLineTable(src).Range(0, min(1, len(src))),
			}}
		}
	}()
	return e.lintFile(path, src, contextFile, maxLevel, ruleSet, known, perRule)
}

func (e *Engine) lintFile(path string, src []byte, contextFile *project.File, maxLevel AnalysisLevel, ruleSet map[string]diagnostic.Severity, known map[string]struct{}, perRule map[string]map[string]any) []diagnostic.Diagnostic {
	var pf *parser.File
	var m *walk.Model
	var semantics *semantic.Model
	if contextFile != nil && bytes.Equal(contextFile.Source, src) {
		pf = contextFile.Parsed
		m = contextFile.Walk
		semantics = contextFile.Semantic
	} else if e.Project != nil {
		if projectFile := e.Project.File(path); projectFile != nil && bytes.Equal(projectFile.Source, src) {
			contextFile = projectFile
			pf = projectFile.Parsed
			m = projectFile.Walk
			semantics = projectFile.Semantic
		}
	}
	if pf == nil {
		if e.ObserveTiming == nil {
			pf = parser.Parse(src)
		} else {
			started := time.Now()
			pf = parser.Parse(src)
			e.observe(TimingEvent{Stage: TimingParse, Duration: time.Since(started)})
		}
	}
	if pf == nil {
		return []diagnostic.Diagnostic{{
			RuleID:   ParseErrorID,
			Severity: diagnostic.SeverityError,
			Category: diagnostic.CategoryCorrectness,
			Message:  "source could not be parsed",
			Filename: path,
			Range:    source.NewLineTable(src).Range(0, min(1, len(src))),
		}}
	}
	if m == nil {
		if contextFile != nil {
			m = contextFile.PointerWalk(pf)
		} else {
			m = walk.NewWithDefines(path, pf, e.Defines)
		}
	}
	lt := m.LineTable

	supps := suppress.FromFile(path, src, pf)
	supps, ruleMigrations := e.normalizeSuppressionAliases(supps, m.LineTable)
	matcher := suppress.NewMatcher(supps)
	used := make([]bool, len(supps))

	var raw []diagnostic.Diagnostic
	parseErrors := parseErrorDiagnostics(path, pf, lt)
	var internalErrors []diagnostic.Diagnostic
	file := &File{Path: path, Source: src, Parsed: pf, LineTable: lt}
	var flow *controlflow.Model
	needSemantics := false
	needFlow := false
	for _, id := range e.Reg.IDs() {
		if _, enabled := ruleSet[id]; !enabled {
			continue
		}
		meta, ok := e.Reg.Lookup(id)
		if !ok || !levelAllowed(meta.AnalysisLevel, maxLevel) {
			continue
		}
		needSemantics = needSemantics || meta.AnalysisLevel >= SemanticAnalysis
		needFlow = needFlow || meta.AnalysisLevel >= ControlFlowAnalysis
	}
	if needSemantics && semantics == nil {
		if e.ObserveTiming == nil {
			semantics = semantic.Build(pf, m)
		} else {
			started := time.Now()
			semantics = semantic.Build(pf, m)
			e.observe(TimingEvent{Stage: TimingSemantic, Duration: time.Since(started)})
		}
	}
	if needFlow {
		if e.ObserveTiming == nil {
			flow = controlflow.BuildWithOptions(m, semantics, e.controlFlowOptions(contextFile, m))
		} else {
			started := time.Now()
			flow = controlflow.BuildWithOptions(m, semantics, e.controlFlowOptions(contextFile, m))
			e.observe(TimingEvent{Stage: TimingControlFlow, Duration: time.Since(started)})
		}
	}

	tokensByKind := func(k token.Kind) []*token.Token {
		var out []*token.Token
		for i := range pf.Tokens {
			if pf.Tokens[i].Kind == k {
				out = append(out, &pf.Tokens[i])
			}
		}
		return out
	}

	for _, id := range e.Reg.IDs() {
		sev, enabled := ruleSet[id]
		if !enabled || sev == diagnostic.SeverityOff {
			continue
		}
		rl, ok := e.Reg.Rule(id)
		if !ok {
			continue
		}
		meta, ok := e.Reg.Lookup(id)
		if !ok || !levelAllowed(meta.AnalysisLevel, maxLevel) {
			continue
		}
		ctx := &Context{
			File:        file,
			Level:       meta.AnalysisLevel,
			Walk:        m,
			Tokens:      tokensByKind,
			Supp:        supps,
			Known:       known,
			PerRule:     perRule,
			Semantic:    semantics,
			Flow:        flow,
			Project:     e.Project,
			ProjectFile: contextFile,
			Target:      e.Target,
			API:         e.API,
		}
		collected := make([]diagnostic.Diagnostic, 0, 8)
		ctx.Report = func(d diagnostic.Diagnostic) {
			if d.RuleID == "" {
				d.RuleID = id
			}
			if d.Severity == 0 {
				d.Severity = sev
			}
			if d.Category == 0 {
				d.Category = meta.Category
			}
			if d.Filename == "" {
				d.Filename = path
			}
			if d.Range.Start.Offset == d.Range.End.Offset && d.Range.Start.Offset == 0 {
				d.Range = lt.Range(0, 0)
			}
			collected = append(collected, d)
		}
		var failed any
		func() {
			if e.ObserveTiming == nil {
				defer func() { failed = recover() }()
				rl.Run(ctx)
				return
			}
			started := time.Now()
			defer func() {
				e.observe(TimingEvent{Stage: TimingRule, RuleID: id, Duration: time.Since(started)})
				failed = recover()
			}()
			rl.Run(ctx)
		}()
		if failed != nil {
			internalErrors = append(internalErrors, diagnostic.Diagnostic{
				RuleID:   InternalErrorID,
				Severity: diagnostic.SeverityError,
				Category: diagnostic.CategoryCorrectness,
				Message:  fmt.Sprintf("rule %q failed: %v", id, failed),
				Filename: path,
				Range:    lt.Range(0, 0),
			})
			continue
		}
		for _, d := range collected {
			if d.RuleID != id && d.RuleID != "" {
				d.RuleID = id
			}
			raw = append(raw, d)
		}
	}
	if maxLevel >= SemanticAnalysis {
		raw = appendSharedDiagnostics(raw, path, src)
	}

	var out []diagnostic.Diagnostic
	for _, d := range raw {
		if d.Filename != path && e.Project != nil {
			projectFile := e.Project.File(d.Filename)
			if projectFile != nil {
				directives := suppress.FromFile(projectFile.Path, projectFile.Source, projectFile.Parsed)
				if projectFile.Parsed == nil {
					directives = suppress.FromCompact(projectFile.Path, projectFile.Source, projectFile.CompactParsed)
				}
				lines := projectFile.LineTable()
				directives, _ = e.normalizeSuppressionAliases(directives, lines)
				projectMatcher := suppress.NewMatcher(directives)
				line := d.Range.Start.Line
				if line == 0 && lines != nil {
					line = lines.Lookup(d.Range.Start.Offset).Line
				}
				if projectMatcher.IsSuppressed(nil, d.RuleID, line) {
					continue
				}
				out = append(out, d)
				continue
			}
		}
		ln := d.Range.Start.Line
		if ln == 0 {
			ln = lt.Lookup(d.Range.Start.Offset).Line
		}
		if matcher.IsSuppressed(used, d.RuleID, ln) {
			continue
		}
		out = append(out, d)
	}

	if _, enabled := ruleSet[SuppressionID]; enabled {
		out = append(out, e.unusedSuppressionDiagnostics(path, supps, used, lt)...)
	}
	out = append(out, parseErrors...)
	out = append(out, internalErrors...)
	out = append(out, ruleMigrations...)

	diagnostic.Sort(out)
	return out
}

func (e *Engine) observe(event TimingEvent) {
	if e.ObserveTiming != nil {
		e.ObserveTiming(event)
	}
}
