package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pawnkit/pawnlint/internal/config"
	"github.com/pawnkit/pawnlint/pkg/project"
)

type result struct {
	Files              int `json:"files"`
	Bytes              int `json:"bytes"`
	Includes           int `json:"includes"`
	UnresolvedIncludes int `json:"unresolvedIncludes"`
	UncertainIncludes  int `json:"uncertainIncludes"`
	BrokenFiles        int `json:"brokenFiles"`
	ErroneousRoots     int `json:"erroneousRoots"`
}

func main() {
	root := flag.String("root", "", "")
	entry := flag.String("entry", "", "")
	configPath := flag.String("config", "", "")
	flag.Parse()
	if *root == "" || *entry == "" || *configPath == "" {
		flag.Usage()
		os.Exit(2)
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(*root, *entry))
	if err != nil {
		fatal(err)
	}
	includePaths := make([]string, len(cfg.IncludePaths))
	for index, path := range cfg.IncludePaths {
		includePaths[index] = filepath.Join(*root, path)
	}
	model, err := project.Build([]project.Source{{Path: filepath.Join(*root, *entry), Content: content}}, project.Options{
		WorkingDir:   *root,
		IncludePaths: includePaths,
		Defines:      cfg.Defines,
	})
	if err != nil {
		fatal(err)
	}
	var output result
	output.Files = len(model.Files)
	for _, file := range model.Files {
		output.Bytes += len(file.Source)
		output.Includes += len(file.Includes)
		if file.Parsed.Broken {
			output.BrokenFiles++
		}
		if file.Parsed.Root != nil && file.Parsed.Root.HasError {
			output.ErroneousRoots++
		}
		for _, include := range file.Includes {
			if include.Uncertain {
				output.UncertainIncludes++
			} else if include.Resolved == nil && !include.Optional {
				output.UnresolvedIncludes++
			}
		}
	}
	if err := json.NewEncoder(os.Stdout).Encode(output); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
