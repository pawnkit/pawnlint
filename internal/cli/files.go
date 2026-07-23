package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/pawnkit/pawnlint/internal/config"
	"github.com/pawnkit/pawnlint/internal/external"
	"github.com/pawnkit/pawnlint/internal/fix"
	"github.com/pawnkit/pawnlint/internal/output"
	discovery "github.com/pawnkit/pawnlint/internal/project"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
	projectmodel "github.com/pawnkit/pawnlint/pkg/project"
	"golang.org/x/sync/errgroup"
)

// fixableDiagnostics drops fixes for rules marked UnsafeFix when safeOnly is set.
func fixableDiagnostics(diags []diagnostic.Diagnostic, reg *lint.Registrar, safeOnly bool) []diagnostic.Diagnostic {
	if !safeOnly {
		return diags
	}
	filtered := make([]diagnostic.Diagnostic, len(diags))
	copy(filtered, diags)
	for i, d := range filtered {
		if d.Fix == nil {
			continue
		}
		if m, ok := reg.Lookup(d.RuleID); ok && m.UnsafeFix {
			filtered[i].Fix = nil
		}
	}
	return filtered
}

func runFiles(opts *cli, stdout, stderr io.Writer, reg *lint.Registrar, r *config.Resolved, timings *runTimings) int {
	cwd, _ := os.Getwd()
	projectDir := cwd
	if r.SourcePath != "" {
		projectDir = filepath.Dir(r.SourcePath)
	}
	if len(opts.Paths) == 0 {
		return runConfiguredBuilds(opts, stdout, stderr, reg, r, projectDir, timings)
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
	if timings != nil {
		engine.ObserveTiming = timings.observeLint
	}
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
	canonicalStart := projectDir
	if len(files) != 0 {
		canonicalStart = files[0].Path
	}
	canonical, err := discovery.Canonical(canonicalStart, includePaths)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "pawnlint: load project: %v\n", err)
		return exitUsage
	}
	workingDir := cwd
	if canonical != nil {
		includePaths = discovery.IncludeRoots(canonical)
		workingDir = filepath.FromSlash(canonical.Root())
		projectDir = workingDir
	}

	variants := r.Source.Variants
	if len(variants) == 0 {
		variants = []config.Variant{{Defines: r.Source.Defines}}
	}

	sources := output.SourceSet{}
	perVariant := make([][]diagnostic.Diagnostic, 0, len(variants))
	cacheDirectory := configuredCacheDirectory(r, projectDir)
	projectFeatures := r.ProjectFeatures(reg)
	for variantIndex, variant := range variants {
		var started time.Time
		if timings != nil {
			started = time.Now()
		}
		model, err := projectmodel.Build(projectSources, projectmodel.Options{
			WorkingDir:      workingDir,
			IncludePaths:    includePaths,
			Defines:         variant.Defines,
			ReleaseExpanded: true,
			ReleaseIncludes: true,
			Features:        &projectFeatures,
			ObserveTiming:   projectTimingObserver(timings),
		})
		if timings != nil {
			timings.addProject(time.Since(started))
		}
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "pawnlint: build project: %v\n", err)
			return exitInternal
		}
		engine.Defines = variant.Defines
		engine.Project = model
		for _, file := range model.Files {
			sources[file.Path] = file.Source
		}
		targetPaths := make([]string, len(files))
		for index, file := range files {
			targetPaths[index] = file.Path
		}
		diags := cachedDiagnostics(cacheDirectory, "paths:"+projectDir, fmt.Sprintf("variant:%d:%s", variantIndex, variant.Name), r, string(r.Target), variant.Defines, r.API, model, targetPaths, func() []diagnostic.Diagnostic {
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
			var diagnostics []diagnostic.Diagnostic
			for _, fileDiagnostics := range perFile {
				diagnostics = append(diagnostics, fileDiagnostics...)
			}
			return diagnostics
		})
		externalDiagnostics, err := external.RunProject(context.Background(), r.Source.ExternalRules, projectDir, variant.Name, string(r.Target), variant.Defines, model, targetPaths)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "pawnlint: %v\n", err)
			return exitInternal
		}
		diags = append(diags, externalDiagnostics...)
		perVariant = append(perVariant, diags)
	}
	all := mergeVariantDiagnostics(perVariant)
	if opts.Diff || opts.Fix || opts.FixSafe {
		plan, err := fix.Build(map[string][]byte(sources), fixableDiagnostics(all, reg, opts.FixSafe && !opts.Fix))
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "pawnlint: build fixes: %v\n", err)
			return exitInternal
		}
		if opts.Diff && len(plan.Changes) != 0 {
			var started time.Time
			if timings != nil {
				started = time.Now()
			}
			_, _ = fmt.Fprint(stdout, fix.Diff(plan))
			if timings != nil {
				timings.addOutput(time.Since(started))
				timings.write(stderr)
			}
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
			return runFiles(&next, stdout, stderr, reg, r, timings)
		}
	}
	return emit(opts, stdout, stderr, all, sources, r, timings)
}

func runConfiguredBuilds(opts *cli, stdout, stderr io.Writer, reg *lint.Registrar, r *config.Resolved, projectDir string, timings *runTimings) int {
	sources := output.SourceSet{}
	perBuild := make([][]diagnostic.Diagnostic, 0, len(r.Source.Builds))
	cacheDirectory := configuredCacheDirectory(r, projectDir)
	projectFeatures := r.ProjectFeatures(reg)
	for _, build := range r.Source.Builds {
		workingDir := resolveBuildPath(projectDir, build.WorkingDirectory)
		entry := resolveBuildPath(workingDir, build.Entry)
		content, err := os.ReadFile(entry)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "pawnlint: build %q entry: %v\n", build.Name, err)
			return exitUsage
		}
		includePaths := resolveConfiguredPaths(projectDir, r.Source.IncludePaths)
		includePaths = appendUniquePaths(includePaths, resolveConfiguredPaths(workingDir, build.IncludePaths)...)
		canonical, err := discovery.Canonical(projectDir, includePaths)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "pawnlint: build %q project: %v\n", build.Name, err)
			return exitUsage
		}
		if canonical != nil {
			includePaths = discovery.IncludeRoots(canonical)
		}
		defines := appendUniqueStrings(r.Source.Defines, build.Defines...)
		target := r.Target
		if build.Target != "" {
			target = config.Target(build.Target)
		}
		metadata, err := r.APIForTarget(target)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "pawnlint: build %q: %v\n", build.Name, err)
			return exitUsage
		}
		var started time.Time
		if timings != nil {
			started = time.Now()
		}
		model, err := projectmodel.Build([]projectmodel.Source{{Path: entry, Content: content}}, projectmodel.Options{
			WorkingDir:      workingDir,
			IncludePaths:    includePaths,
			Defines:         defines,
			DefinesComplete: true,
			ReleaseExpanded: true,
			ReleaseIncludes: true,
			Features:        &projectFeatures,
			ObserveTiming:   projectTimingObserver(timings),
		})
		if timings != nil {
			timings.addProject(time.Since(started))
		}
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "pawnlint: build %q project: %v\n", build.Name, err)
			return exitInternal
		}
		engine := lint.NewEngine(reg)
		engine.Target = string(target)
		engine.API = metadata
		engine.Defines = defines
		engine.Project = model
		if timings != nil {
			engine.ObserveTiming = timings.observeLint
		}
		files := configuredBuildFiles(model, entry, workingDir, build.Files, build.Exclude)
		model.ReleaseIncludeTokens(files)
		targetPaths := make([]string, len(files))
		for index, file := range files {
			targetPaths[index] = file.Path
		}
		diagnostics := cachedDiagnostics(cacheDirectory, "build:"+projectDir+":"+build.Name, "build:"+build.Name, r, string(target), defines, metadata, model, targetPaths, func() []diagnostic.Diagnostic {
			perFile := make([][]diagnostic.Diagnostic, len(files))
			var eg errgroup.Group
			eg.SetLimit(max(runtime.GOMAXPROCS(0), 1))
			for i, file := range files {
				eg.Go(func() error {
					rel := discovery.RelPath(projectDir, file.Path)
					perFile[i] = engine.LintProjectFile(file, lint.ProjectAnalysis, r.EnabledForPath(rel), r.AllKnownRuleIDs, r.RuleConfigForPath(rel))
					return nil
				})
			}
			_ = eg.Wait()
			var findings []diagnostic.Diagnostic
			for _, fileDiagnostics := range perFile {
				findings = append(findings, fileDiagnostics...)
			}
			return findings
		})
		externalDiagnostics, err := external.RunProject(context.Background(), r.Source.ExternalRules, projectDir, build.Name, string(target), defines, model, targetPaths)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "pawnlint: build %q: %v\n", build.Name, err)
			return exitInternal
		}
		diagnostics = append(diagnostics, externalDiagnostics...)
		perBuild = append(perBuild, diagnostics)
		for _, file := range model.Files {
			sources[file.Path] = file.Source
		}
	}
	all := mergeBuildDiagnostics(perBuild)
	if opts.Diff || opts.Fix || opts.FixSafe {
		plan, err := fix.Build(map[string][]byte(sources), fixableDiagnostics(all, reg, opts.FixSafe && !opts.Fix))
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "pawnlint: build fixes: %v\n", err)
			return exitInternal
		}
		if opts.Diff && len(plan.Changes) != 0 {
			var started time.Time
			if timings != nil {
				started = time.Now()
			}
			_, _ = fmt.Fprint(stdout, fix.Diff(plan))
			if timings != nil {
				timings.addOutput(time.Since(started))
				timings.write(stderr)
			}
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
			return runFiles(&next, stdout, stderr, reg, r, timings)
		}
	}
	return emit(opts, stdout, stderr, all, sources, r, timings)
}

func projectTimingObserver(timings *runTimings) func(projectmodel.TimingEvent) {
	if timings == nil {
		return nil
	}
	return timings.observeProject
}

func checkBuildIncludes(r *config.Resolved, projectDir string, build config.Build, entry, workingDir string) error {
	content, err := os.ReadFile(entry)
	if err != nil {
		return fmt.Errorf("build %q entry: %w", build.Name, err)
	}
	includePaths := resolveConfiguredPaths(projectDir, r.Source.IncludePaths)
	includePaths = appendUniquePaths(includePaths, resolveConfiguredPaths(workingDir, build.IncludePaths)...)
	canonical, err := discovery.Canonical(projectDir, includePaths)
	if err != nil {
		return fmt.Errorf("build %q project: %w", build.Name, err)
	}
	if canonical != nil {
		includePaths = discovery.IncludeRoots(canonical)
	}
	defines := appendUniqueStrings(r.Source.Defines, build.Defines...)
	features := projectmodel.NewFeatures(projectmodel.FeatureIncludeIssues)
	model, err := projectmodel.Build([]projectmodel.Source{{Path: entry, Content: content}}, projectmodel.Options{
		WorkingDir:      workingDir,
		IncludePaths:    includePaths,
		Defines:         defines,
		DefinesComplete: true,
		Features:        &features,
	})
	if err != nil {
		return fmt.Errorf("build %q project: %w", build.Name, err)
	}
	if missing := model.MissingIncludes(); len(missing) != 0 {
		issue := missing[0]
		return fmt.Errorf("build %q: %s includes missing target %q", build.Name, issue.File.Path, issue.Include.Path)
	}
	return nil
}

func configuredBuildFiles(model *projectmodel.Model, entry, workingDir string, patterns, excludes []string) []*projectmodel.File {
	files := make([]*projectmodel.File, 0, len(model.Files))
	for _, file := range model.Files {
		isEntry := samePath(file.Path, entry)
		selected := isEntry
		rel := discovery.RelPath(workingDir, file.Path)
		if !selected && matchesBuildPatterns(patterns, rel) {
			selected = true
		}
		if !selected || !isEntry && matchesBuildPatterns(excludes, rel) {
			continue
		}
		files = append(files, file)
	}
	return files
}

func matchesBuildPatterns(patterns []string, path string) bool {
	for _, pattern := range patterns {
		if discovery.MatchGlob(pattern, path) {
			return true
		}
	}
	return false
}

func resolveBuildPath(base, path string) string {
	if path == "" {
		return filepath.Clean(base)
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(base, path)
	}
	return filepath.Clean(path)
}

func resolveConfiguredPaths(base string, paths []string) []string {
	resolved := make([]string, 0, len(paths))
	for _, path := range paths {
		resolved = append(resolved, resolveBuildPath(base, path))
	}
	return resolved
}

func appendUniquePaths(paths []string, additions ...string) []string {
	for _, addition := range additions {
		found := false
		for _, path := range paths {
			if samePath(path, addition) {
				found = true
				break
			}
		}
		if !found {
			paths = append(paths, addition)
		}
	}
	return paths
}

func appendUniqueStrings(values []string, additions ...string) []string {
	merged := append([]string(nil), values...)
	for _, addition := range additions {
		found := false
		for _, value := range merged {
			if value == addition {
				found = true
				break
			}
		}
		if !found {
			merged = append(merged, addition)
		}
	}
	return merged
}

func samePath(left, right string) bool {
	leftPath, leftErr := filepath.Abs(left)
	rightPath, rightErr := filepath.Abs(right)
	return leftErr == nil && rightErr == nil && filepath.Clean(leftPath) == filepath.Clean(rightPath)
}

func mergeBuildDiagnostics(perBuild [][]diagnostic.Diagnostic) []diagnostic.Diagnostic {
	var merged []diagnostic.Diagnostic
	seen := make(map[variantDiagKey]struct{})
	for _, diagnostics := range perBuild {
		for _, item := range diagnostics {
			key := variantDiagKeyFor(item)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			merged = append(merged, item)
		}
	}
	return merged
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
