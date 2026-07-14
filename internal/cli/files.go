package cli

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	"github.com/pawnkit/pawnlint/internal/config"
	"github.com/pawnkit/pawnlint/internal/fix"
	"github.com/pawnkit/pawnlint/internal/output"
	discovery "github.com/pawnkit/pawnlint/internal/project"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
	projectmodel "github.com/pawnkit/pawnlint/pkg/project"
	"golang.org/x/sync/errgroup"
)

func runFiles(opts *cli, stdout, stderr io.Writer, reg *lint.Registrar, r *config.Resolved) int {
	cwd, _ := os.Getwd()
	projectDir := cwd
	if r.SourcePath != "" {
		projectDir = filepath.Dir(r.SourcePath)
	}
	files, err := discovery.Discover(discovery.Options{
		Roots:      opts.Paths,
		Include:    r.Source.Include,
		Exclude:    r.Source.Exclude,
		WorkingDir: projectDir,
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "pawnlint: %v\n", err)
		if !errors.Is(err, fs.ErrNotExist) {
			return exitInternal
		}
		return exitUsage
	}
	engine := lint.NewEngine(reg)
	engine.Target = string(r.Target)
	engine.API = r.API
	projectSources := make([]projectmodel.Source, 0, len(files))
	for _, file := range files {
		projectSources = append(projectSources, projectmodel.Source{Path: file.Path, Content: file.Content})
	}
	includePaths := make([]string, len(r.Source.IncludePaths))
	for i, path := range r.Source.IncludePaths {
		if !filepath.IsAbs(path) {
			path = filepath.Join(projectDir, path)
		}
		includePaths[i] = path
	}

	variants := r.Source.Variants
	if len(variants) == 0 {
		variants = []config.Variant{{Defines: r.Source.Defines}}
	}

	sources := output.SourceSet{}
	perVariant := make([][]diagnostic.Diagnostic, 0, len(variants))
	for _, variant := range variants {
		model, err := projectmodel.Build(projectSources, projectmodel.Options{
			WorkingDir:   cwd,
			IncludePaths: includePaths,
			Defines:      variant.Defines,
		})
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "pawnlint: build project: %v\n", err)
			return exitInternal
		}
		engine.Defines = variant.Defines
		engine.Project = model
		for _, file := range model.Files {
			sources[file.Path] = file.Source
		}
		perFile := make([][]diagnostic.Diagnostic, len(files))
		var eg errgroup.Group
		eg.SetLimit(max(runtime.GOMAXPROCS(0), 1))
		for i, f := range files {
			eg.Go(func() error {
				rel := discovery.RelPath(projectDir, f.Path)
				perFile[i] = engine.LintFile(f.Path, f.Content, lint.ProjectAnalysis, r.EnabledForPath(rel), r.AllKnownRuleIDs, r.RuleConfigForPath(rel))
				return nil
			})
		}
		_ = eg.Wait()
		var diags []diagnostic.Diagnostic
		for _, fd := range perFile {
			diags = append(diags, fd...)
		}
		perVariant = append(perVariant, diags)
	}
	all := mergeVariantDiagnostics(perVariant)
	if opts.Diff || opts.Fix || opts.FixSafe {
		plan, err := fix.Build(map[string][]byte(sources), all)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "pawnlint: build fixes: %v\n", err)
			return exitInternal
		}
		if opts.Diff && len(plan.Changes) != 0 {
			_, _ = fmt.Fprint(stdout, fix.Diff(plan))
			return exitFindings
		}
		if (opts.Fix || opts.FixSafe) && len(plan.Changes) != 0 {
			if err := fix.Write(plan); err != nil {
				_, _ = fmt.Fprintf(stderr, "pawnlint: write fixes: %v\n", err)
				return exitInternal
			}
			next := *opts
			next.Fix = false
			next.FixSafe = false
			return runFiles(&next, stdout, stderr, reg, r)
		}
	}
	return emit(opts, stdout, stderr, all, sources, r)
}

type variantDiagKey struct {
	rule  string
	file  string
	start int
	end   int
}

func variantDiagKeyFor(d diagnostic.Diagnostic) variantDiagKey {
	return variantDiagKey{d.RuleID, d.Filename, d.Range.Start.Offset, d.Range.End.Offset}
}

func mergeVariantDiagnostics(perVariant [][]diagnostic.Diagnostic) []diagnostic.Diagnostic {
	if len(perVariant) == 1 {
		return perVariant[0]
	}
	var merged []diagnostic.Diagnostic
	seen := make(map[variantDiagKey]struct{})
	suppressionFirst := make(map[variantDiagKey]diagnostic.Diagnostic)
	suppressionCount := make(map[variantDiagKey]int)
	for _, diags := range perVariant {
		for _, d := range diags {
			key := variantDiagKeyFor(d)
			if d.RuleID == lint.SuppressionID {
				suppressionCount[key]++
				if _, ok := suppressionFirst[key]; !ok {
					suppressionFirst[key] = d
				}
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			merged = append(merged, d)
		}
	}
	for key, count := range suppressionCount {
		if count == len(perVariant) {
			merged = append(merged, suppressionFirst[key])
		}
	}
	return merged
}
