// Package editor lints open buffers for editor integrations.
package editor

import (
	"bytes"
	"context"
	"path/filepath"
	"sync"

	analysis "github.com/pawnkit/pawn-analysis"
	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	coresource "github.com/pawnkit/pawnkit-core/source"
	"github.com/pawnkit/pawnlint/internal/config"
	projectcontext "github.com/pawnkit/pawnlint/internal/project"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/project"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

var (
	defaultRegistryOnce sync.Once
	defaultRegistry     *lint.Registrar
)

func defaultRules() *lint.Registrar {
	defaultRegistryOnce.Do(func() { defaultRegistry = rules.Default() })
	return defaultRegistry
}

// Diagnose lints content as path using configuration found from workingDir.
func Diagnose(path string, content []byte, workingDir string) ([]diagnostic.Diagnostic, error) {
	return DiagnoseWithCache(path, content, workingDir, nil, nil)
}

// DiagnoseWithCache lints content as path using configuration found from
// workingDir. It reuses cached includes and shared analysis facts when
// supplied. Nil arguments behave like [Diagnose].
func DiagnoseWithCache(path string, content []byte, workingDir string, cache *project.ParseCache, shared *analysis.Result) ([]diagnostic.Diagnostic, error) {
	return DiagnoseContextWithCache(context.Background(), path, content, workingDir, cache, shared)
}

// DiagnoseContextWithCache stops before lint rules when ctx is cancelled.
func DiagnoseContextWithCache(
	ctx context.Context,
	path string,
	content []byte,
	workingDir string,
	cache *project.ParseCache,
	shared *analysis.Result,
) ([]diagnostic.Diagnostic, error) {
	return diagnoseContext(ctx, path, content, workingDir, cache, shared, nil)
}

func diagnoseContext(
	ctx context.Context,
	path string,
	content []byte,
	workingDir string,
	cache *project.ParseCache,
	shared *analysis.Result,
	observe func(project.TimingEvent),
) ([]diagnostic.Diagnostic, error) {
	return diagnoseContextWithTimings(ctx, path, content, workingDir, cache, shared, observe, nil)
}

func diagnoseContextWithTimings(
	ctx context.Context,
	path string,
	content []byte,
	workingDir string,
	cache *project.ParseCache,
	shared *analysis.Result,
	observeProject func(project.TimingEvent),
	observeLint func(lint.TimingEvent),
) ([]diagnostic.Diagnostic, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	reg := defaultRules()

	configPath, file, err := config.Discover(workingDir)
	if err != nil {
		return nil, err
	}

	resolved, err := config.Resolve(file, configPath, reg)
	if err != nil {
		return nil, err
	}

	base := workingDir
	if configPath != "" {
		base = filepath.Dir(configPath)
	}

	includePaths := make([]string, len(resolved.Source.IncludePaths))
	for i, p := range resolved.Source.IncludePaths {
		if !filepath.IsAbs(p) {
			p = filepath.Join(base, p)
		}
		includePaths[i] = p
	}
	canonical, err := projectcontext.Canonical(workingDir, includePaths)
	if err != nil {
		return nil, err
	}
	if canonical != nil {
		includePaths = projectcontext.IncludeRoots(canonical)
		workingDir = filepath.FromSlash(canonical.Root())
	}

	var rootTokens []token.Token
	var rootParsed *parser.File
	if shared != nil && shared.Preprocess != nil {
		rootTokens = shared.Preprocess.OriginalTokens
	}
	delegated := make(map[string]struct{})
	if shared != nil {
		for _, id := range reg.IDs() {
			if lint.DelegatesToShared(id) {
				delegated[id] = struct{}{}
			}
		}
	}
	features := resolved.ProjectFeaturesExcluding(delegated)
	if shared != nil && shared.Parse != nil && bytes.Equal(shared.Parse.Source, content) {
		if cache != nil {
			rootParsed = cache.ExpandRoot(path, shared.Parse, rootTokens)
		} else {
			rootParsed = shared.Parse.ExpandTokensWithOptions(rootTokens, parser.ParseOptions{})
		}
	}
	var sharedSources []project.Source
	if shared != nil && shared.Preprocess != nil {
		prepared := make([]project.PreparedSource, 0, len(shared.Preprocess.Files))
		for i, file := range shared.Preprocess.Files {
			uri := coresource.URI(file.URI)
			filename, err := uri.Filename()
			if err != nil {
				continue
			}
			canonical, err := filepath.Abs(filename)
			if err != nil {
				continue
			}
			path := filepath.Clean(canonical)
			var compactSyntax *parser.CompactFile
			if i == 0 && shared.Parse != nil && bytes.Equal(shared.Parse.Source, file.Content) {
				compactSyntax = shared.Parse
			}
			discardTrivia := compactSyntax == nil && !features.Has(project.FeatureTrivia) &&
				!bytes.Contains(file.Content, []byte("pawnlint-"))
			sharedSources = append(sharedSources, project.Source{Path: path, Content: file.Content})
			if compactSyntax != nil && rootParsed != nil {
				continue
			}
			prepared = append(prepared, project.PreparedSource{
				Path: path, Content: file.Content, Tokens: file.Tokens,
				CompactSyntax: compactSyntax,
				DiscardTrivia: discardTrivia,
			})
		}
		if cache != nil {
			if err := cache.PrepareContext(ctx, prepared); err != nil {
				return nil, err
			}
		}
	}
	model, err := project.BuildContext(
		ctx,
		[]project.Source{{Path: path, Content: content}},
		project.Options{
			WorkingDir: workingDir, IncludePaths: includePaths, Defines: resolved.Source.Defines,
			ParseCache: cache, Features: &features, RootTokens: rootTokens, RootParsed: rootParsed, ObserveTiming: observeProject,
			IncludeSources: sharedSources,
		},
	)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	engine := lint.NewEngine(reg)
	engine.Defines = resolved.Source.Defines
	engine.Target = string(resolved.Target)
	engine.Project = model
	engine.API = resolved.API
	engine.SharedAnalysis = shared
	engine.Context = ctx
	engine.ObserveTiming = observeLint

	diagnostics := engine.LintFile(path, content, lint.ProjectAnalysis, resolved.Enabled, resolved.AllKnownRuleIDs, resolved.RuleConfig)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	diagnostic.Sort(diagnostics)
	return diagnostics, nil
}
