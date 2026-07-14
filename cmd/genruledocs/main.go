package main

import (
	"os"

	"github.com/pawnkit/pawnlint/internal/docgen"
)

func main() {
	os.Exit(docgen.Run(os.Args[1:]))
}
