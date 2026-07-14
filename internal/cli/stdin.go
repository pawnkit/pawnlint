package cli

import (
	"fmt"
	"io"

	"github.com/pawnkit/pawnlint/internal/config"
	"github.com/pawnkit/pawnlint/internal/fix"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

func runStdin(opts *cli, stdin io.Reader, stdout, stderr io.Writer, reg *lint.Registrar, r *config.Resolved) int {
	src, err := io.ReadAll(stdin)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "pawnlint: read stdin: %v\n", err)
		return exitInternal
	}
	name := opts.StdinName
	if name == "" {
		name = "stdin.pwn"
	}
	engine := lint.NewEngine(reg)
	engine.Defines = r.Source.Defines
	engine.Target = string(r.Target)
	engine.API = r.API
	diags := engine.LintFile(name, src, lint.ControlFlowAnalysis, r.Enabled, r.AllKnownRuleIDs, r.RuleConfig)
	if opts.Diff {
		plan, err := fix.Build(map[string][]byte{name: src}, diags)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "pawnlint: build fixes: %v\n", err)
			return exitInternal
		}
		if len(plan.Changes) != 0 {
			_, _ = fmt.Fprint(stdout, fix.Diff(plan))
			return exitFindings
		}
	}
	return emit(opts, stdout, stderr, diags, map[string][]byte{name: src}, r)
}
