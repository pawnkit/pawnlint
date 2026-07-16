package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"

	"github.com/pawnkit/pawnlint/internal/config"
	"github.com/pawnkit/pawnlint/pkg/project"
)

type result struct {
	Files              int `json:"files"`
	Bytes              int `json:"bytes"`
	Tokens             int `json:"tokens"`
	Functions          int `json:"functions"`
	Calls              int `json:"calls"`
	DynamicCalls       int `json:"dynamicCalls"`
	TimerCalls         int `json:"timerCalls"`
	EntryPoints        int `json:"entryPoints"`
	ExpandedFiles      int `json:"expandedFiles"`
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
	heapProfile := flag.String("heap-profile", "", "")
	cpuProfile := flag.String("cpu-profile", "", "")
	releaseExpanded := flag.Bool("release-expanded", false, "")
	flag.Parse()
	if *root == "" || *entry == "" || *configPath == "" {
		flag.Usage()
		os.Exit(2)
	}
	if *cpuProfile != "" {
		file, err := os.Create(*cpuProfile)
		if err != nil {
			fatal(err)
		}
		if err := pprof.StartCPUProfile(file); err != nil {
			_ = file.Close()
			fatal(err)
		}
		defer func() {
			pprof.StopCPUProfile()
			_ = file.Close()
		}()
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
		WorkingDir:      *root,
		IncludePaths:    includePaths,
		Defines:         cfg.Defines,
		DefinesComplete: true,
		ReleaseExpanded: *releaseExpanded,
		ReleaseIncludes: *releaseExpanded,
	})
	if err != nil {
		fatal(err)
	}
	var output result
	output.Files = len(model.Files)
	for _, file := range model.Files {
		output.Bytes += len(file.Source)
		if file.Parsed != nil {
			output.Tokens += len(file.Parsed.Tokens)
		} else if file.CompactParsed != nil {
			output.Tokens += len(file.CompactParsed.Tokens)
		}
		output.Includes += len(file.Includes)
		if file.Parsed != nil && file.ExpandedParsed != file.Parsed {
			output.ExpandedFiles++
		}
		if file.Parsed != nil && file.Parsed.Broken || file.CompactParsed != nil && file.CompactParsed.Broken {
			output.BrokenFiles++
		}
		if file.Parsed != nil && file.Parsed.Root != nil && file.Parsed.Root.HasError || file.CompactParsed != nil && file.CompactParsed.Tree.Nodes[file.CompactParsed.Tree.Root].HasError {
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
	if model.CallGraph != nil {
		output.Functions = len(model.CallGraph.Functions)
		output.Calls = len(model.CallGraph.Calls)
		output.TimerCalls = len(model.CallGraph.AsyncCalls)
		output.EntryPoints = len(model.CallGraph.EntryPoints)
		for _, call := range model.CallGraph.Calls {
			if call.Kind == project.CallDynamic {
				output.DynamicCalls++
			}
		}
	}
	if err := json.NewEncoder(os.Stdout).Encode(output); err != nil {
		fatal(err)
	}
	if *heapProfile != "" {
		runtime.GC()
		file, err := os.Create(*heapProfile)
		if err != nil {
			fatal(err)
		}
		if err := pprof.WriteHeapProfile(file); err != nil {
			_ = file.Close()
			fatal(err)
		}
		if err := file.Close(); err != nil {
			fatal(err)
		}
		runtime.KeepAlive(model)
	}
}

func fatal(err error) {
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
