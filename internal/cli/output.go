package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/pawnkit/pawnlint/internal/config"
	"github.com/pawnkit/pawnlint/internal/output"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
)

func emit(opts *cli, stdout, stderr io.Writer, diags []diagnostic.Diagnostic, sources output.SourceSet, r *config.Resolved) int {
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
