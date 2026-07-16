package main

import (
	"os"
	"runtime/debug"

	"github.com/pawnkit/pawnlint/internal/cli"
	"go.uber.org/automaxprocs/maxprocs"
)

func main() {
	_, _ = maxprocs.Set(maxprocs.Logger(func(string, ...any) {}))
	if os.Getenv("GOMEMLIMIT") == "" {
		debug.SetMemoryLimit(900 << 20)
	}
	os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
