package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pawnkit/pawnlint/internal/baseline"
	"github.com/pawnkit/pawnlint/internal/config"
	"github.com/pawnkit/pawnlint/internal/output"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
)

func emit(opts *cli, stdout, stderr io.Writer, diags []diagnostic.Diagnostic, sources output.SourceSet, r *config.Resolved) int {
	baselinePath, projectDir := resolvedBaselinePath(opts, r)
	if baselinePath != "" {
		if opts.GenerateBaseline {
			generated := baseline.Generate(diags, map[string][]byte(sources), projectDir)
			if err := baseline.Write(baselinePath, generated); err != nil {
				_, _ = fmt.Fprintf(stderr, "pawnlint: %v\n", err)
				return exitInternal
			}
			_, _ = fmt.Fprintf(stderr, "pawnlint: wrote %d baseline entries to %s\n", len(generated.Entries), baselinePath)
			diags = baseline.Apply(generated, diags, map[string][]byte(sources), projectDir).Remaining
		} else {
			configured, err := baseline.Load(baselinePath)
			if err != nil {
				_, _ = fmt.Fprintf(stderr, "pawnlint: %v\n", err)
				return exitUsage
			}
			matched := baseline.Apply(configured, diags, map[string][]byte(sources), projectDir)
			diags = matched.Remaining
			if opts.PruneBaseline {
				if err := baseline.Write(baselinePath, matched.Current); err != nil {
					_, _ = fmt.Fprintf(stderr, "pawnlint: %v\n", err)
					return exitInternal
				}
				_, _ = fmt.Fprintf(stderr, "pawnlint: pruned %d stale baseline entries from %s\n", matched.Stale, baselinePath)
			}
		}
	}
	diagnostic.Sort(diags)
	fail := reachedThreshold(r, diags)
	if limit := r.Source.Lint.MaxDiagnostics; limit > 0 && len(diags) > limit {
		diags = diags[:limit]
	}
	useColor := opts.Color == "always" || opts.Color == "auto" && isTerminal(stdout)
	if err := output.Write(stdout, output.Format(opts.Format), diags, sources, useColor); err != nil {
		_, _ = fmt.Fprintf(stderr, "pawnlint: %v\n", err)
		return exitInternal
	}
	if fail {
		return exitFindings
	}
	return exitOK
}

func baselineSetting(opts *cli, r *config.Resolved) string {
	if opts.Baseline != "" {
		return opts.Baseline
	}
	return r.Source.Baseline
}

func resolvedBaselinePath(opts *cli, r *config.Resolved) (string, string) {
	cwd, _ := os.Getwd()
	projectDir := cwd
	if r.SourcePath != "" {
		projectDir = filepath.Dir(r.SourcePath)
		if !filepath.IsAbs(projectDir) {
			projectDir = filepath.Join(cwd, projectDir)
		}
	}
	path := baselineSetting(opts, r)
	if path == "" {
		return "", filepath.Clean(projectDir)
	}
	baseDir := projectDir
	if opts.Baseline != "" {
		baseDir = cwd
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	return filepath.Clean(path), filepath.Clean(projectDir)
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func reachedThreshold(r *config.Resolved, diags []diagnostic.Diagnostic) bool {
	if len(diags) == 0 {
		return false
	}
	for _, d := range diags {
		if d.Severity == diagnostic.SeverityError {
			return true
		}
		if r.Source.Lint.WarningsAsErrors && d.Severity == diagnostic.SeverityWarning {
			return true
		}
	}
	return false
}
