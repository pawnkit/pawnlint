// Package cli implements the pawnlint command.
package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/pawnkit/pawnlint/internal/config"
	"github.com/pawnkit/pawnlint/internal/output"
	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

const (
	exitOK       = 0
	exitFindings = 1
	exitUsage    = 2
	exitInternal = 3
)

type cli struct {
	Paths      []string         `arg:"" optional:"" name:"path" help:"files or directories to lint"`
	Config     string           `help:"path to a pawnlint.toml config file"`
	Profile    string           `help:"rule profile (recommended|strict|all)"`
	Target     string           `help:"target dialect (openmp|samp)"`
	Enable     []string         `help:"enable a rule by id (repeatable)"`
	Disable    []string         `help:"disable a rule by id (repeatable)"`
	Format     string           `default:"text" help:"output format (text|compact|json|jsonl|sarif|github)"`
	ListRules  bool             `help:"list all rules and exit"`
	Explain    string           `help:"print documentation for a rule and exit"`
	InitConfig bool             `help:"write a commented pawnlint.toml with defaults and exit"`
	Stdin      bool             `help:"read source from stdin"`
	StdinName  string           `name:"stdin-filename" help:"virtual filename for stdin input"`
	Fix        bool             `help:"apply machine-safe fixes"`
	FixSafe    bool             `help:"apply only machine-safe fixes"`
	Diff       bool             `help:"print a diff of available fixes"`
	Color      string           `default:"auto" help:"colour: auto|always|never"`
	Version    kong.VersionFlag `short:"V" help:"print version and exit"`
}

func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) (code int) {
	opts, code, done := parse(args, stdout, stderr)
	if done {
		return code
	}
	reg := rules.Default()
	if opts.ListRules {
		_, _ = fmt.Fprint(stdout, config.ListRulesText(reg))
		return exitOK
	}
	if opts.Explain != "" {
		m, ok := reg.Lookup(opts.Explain)
		if !ok {
			_, _ = fmt.Fprintf(stderr, "pawnlint: unknown rule %q\n", opts.Explain)
			return exitUsage
		}
		_, _ = fmt.Fprint(stdout, config.ExplainText(m))
		return exitOK
	}
	if opts.InitConfig {
		_, _ = fmt.Fprint(stdout, config.InitConfigText(reg))
		return exitOK
	}
	if !output.AllowedFormat(opts.Format) {
		_, _ = fmt.Fprintf(stderr, "pawnlint: unknown --format %q (allowed: %s)\n", opts.Format, strings.Join(output.AllFormats(), ", "))
		return exitUsage
	}
	if opts.Color != "auto" && opts.Color != "always" && opts.Color != "never" {
		_, _ = fmt.Fprintf(stderr, "pawnlint: unknown --color %q (allowed: auto, always, never)\n", opts.Color)
		return exitUsage
	}
	if opts.Diff && (opts.Fix || opts.FixSafe) {
		_, _ = fmt.Fprintln(stderr, "pawnlint: --diff cannot be combined with --fix or --fix-safe")
		return exitUsage
	}
	if opts.Stdin && (opts.Fix || opts.FixSafe) {
		_, _ = fmt.Fprintln(stderr, "pawnlint: --fix and --fix-safe cannot write stdin")
		return exitUsage
	}
	resolved, err := resolveConfig(opts, reg)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "pawnlint: %v\n", err)
		return exitUsage
	}
	if opts.Stdin {
		return runStdin(opts, stdin, stdout, stderr, reg, resolved)
	}
	if len(opts.Paths) == 0 {
		_, _ = fmt.Fprintln(stderr, "pawnlint: no input; pass file/directory paths or use --stdin")
		return exitUsage
	}
	return runFiles(opts, stdout, stderr, reg, resolved)
}

type kongEagerExit struct{ code int }

func parse(args []string, stdout, stderr io.Writer) (opts *cli, code int, done bool) {
	opts = &cli{}
	parser, err := kong.New(opts,
		kong.Name("pawnlint"),
		kong.Description("A static-analysis and linting tool for Pawn (SA-MP/open.mp)."),
		kong.Writers(stdout, stderr),
		kong.Exit(func(c int) { panic(kongEagerExit{code: c}) }),
		kong.Vars{"version": Version},
	)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "pawnlint: cli setup: %v\n", err)
		return nil, exitInternal, true
	}
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		if e, ok := r.(kongEagerExit); ok {
			code = e.code
			done = true
			return
		}
		panic(r)
	}()
	if _, err := parser.Parse(args); err != nil {
		_, _ = fmt.Fprintf(stderr, "pawnlint: %v\n", err)
		return opts, exitUsage, true
	}
	return opts, exitOK, false
}

func resolveConfig(opts *cli, reg *lint.Registrar) (*config.Resolved, error) {
	var f config.File
	var srcPath string
	if opts.Config != "" {
		loaded, err := config.Load(opts.Config)
		if err != nil {
			return nil, err
		}
		f = loaded
		srcPath = opts.Config
	} else {
		cwd, _ := os.Getwd()
		path, loaded, err := config.Discover(cwd)
		if err != nil {
			return nil, err
		}
		f = loaded
		srcPath = path
	}
	r, err := config.Resolve(f, srcPath, reg)
	if err != nil {
		return nil, err
	}
	if err := r.ApplyCLIOverrides(opts.Profile, opts.Target, opts.Enable, opts.Disable, reg); err != nil {
		return nil, err
	}
	return r, nil
}
