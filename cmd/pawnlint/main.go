package main

import (
	"os"

	"github.com/pawnkit/pawnlint/internal/cli"
	"go.uber.org/automaxprocs/maxprocs"
)

func main() {
	_, _ = maxprocs.Set(maxprocs.Logger(func(string, ...any) {}))
	os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
