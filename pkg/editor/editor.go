// Package editor lints open buffers for editor integrations.
package editor

import (
	"bytes"
	"context"
	"path/filepath"

	analysis "github.com/pawnkit/pawn-analysis"
	"github.com/pawnkit/pawn-parser/token"
	coresource "github.com/pawnkit/pawnkit-core/source"
	"github.com/pawnkit/pawnlint/internal/config"
	projectcontext "github.com/pawnkit/pawnlint/internal/project"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/project"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

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
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	reg := rules.Default()

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
	if cache != nil && shared != nil && shared.Preprocess != nil {
		prepared := make([]project.PreparedSource, 0, len(shared.Preprocess.Files))
		for _, file := range shared.Preprocess.Files {
			uri := coresource.URI(file.URI)
			filename, err := uri.Filename()
			if err != nil {
				continue
			}
			canonical, err := filepath.Abs(filename)
			if err != nil {
				continue
			}
			prepared = append(prepared, project.PreparedSource{
				Path: filepath.Clean(canonical), Content: file.Content, Tokens: file.Tokens,
				DiscardTrivia: !features.Has(project.FeatureTrivia) && !bytes.Contains(file.Content, []byte("pawnlint-")),
			})
		}
		if err := cache.PrepareContext(ctx, prepared); err != nil {
			return nil, err
		}
	}
	model, err := project.BuildContext(
		ctx,
		[]project.Source{{Path: path, Content: content}},
		project.Options{
			WorkingDir: workingDir, IncludePaths: includePaths, Defines: resolved.Source.Defines,
			ParseCache: cache, Features: &features, RootTokens: rootTokens, ObserveTiming: observe,
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

	diagnostics := engine.LintFile(path, content, lint.ProjectAnalysis, resolved.Enabled, resolved.AllKnownRuleIDs, resolved.RuleConfig)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	diagnostic.Sort(diagnostics)
	return diagnostics, nil
}
