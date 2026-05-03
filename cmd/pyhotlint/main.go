// pyhotlint analyzes Python source files for inference-server hot-path
// hazards. MVP: takes one or more file paths, parses them with
// tree-sitter, runs the registered rules, and emits findings as JSON.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kaeawc/pyhotlint/internal/output"
	_ "github.com/kaeawc/pyhotlint/internal/rules" // registers rules
	v2 "github.com/kaeawc/pyhotlint/internal/rules/v2"
	"github.com/kaeawc/pyhotlint/internal/scanner"
)

var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		return
	}

	paths := flag.Args()
	if len(paths) == 0 {
		fmt.Fprintln(os.Stderr, "usage: pyhotlint <file.py> [file.py ...]")
		os.Exit(2)
	}

	rules := v2.All()
	var all []v2.Finding
	exit := 0
	for _, p := range paths {
		pf, err := scanner.ParseFile(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "pyhotlint: %v\n", err)
			exit = 1
			continue
		}
		findings := v2.Run(rules, pf.Path, pf.Source, pf.Tree.RootNode())
		all = append(all, findings...)
		pf.Close()
	}

	if err := output.WriteJSON(os.Stdout, all); err != nil {
		fmt.Fprintf(os.Stderr, "pyhotlint: %v\n", err)
		os.Exit(1)
	}
	if len(all) > 0 && exit == 0 {
		exit = 1 // nonzero when findings exist, like ruff/mypy
	}
	os.Exit(exit)
}
