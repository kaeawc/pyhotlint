// pyhotlint analyzes Python source files for inference-server hot-path
// hazards. The CLI takes file paths, directories, or shell-style globs;
// directories are walked recursively (skipping common venv / cache /
// build dirs). Findings are emitted as a JSON array on stdout.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kaeawc/pyhotlint/internal/output"
	_ "github.com/kaeawc/pyhotlint/internal/rules" // registers rules
	v2 "github.com/kaeawc/pyhotlint/internal/rules/v2"
	"github.com/kaeawc/pyhotlint/internal/scanner"
	"github.com/kaeawc/pyhotlint/internal/walker"
)

var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: pyhotlint <path> [path ...]")
		fmt.Fprintln(os.Stderr, "  paths may be files, directories (walked recursively), or shell globs")
	}
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		return
	}

	paths := flag.Args()
	if len(paths) == 0 {
		flag.Usage()
		os.Exit(2)
	}

	files, err := walker.FindFiles(paths)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pyhotlint: %v\n", err)
		os.Exit(2)
	}

	rules := v2.All()
	var all []v2.Finding
	exit := 0
	for _, p := range files {
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
		exit = 1
	}
	os.Exit(exit)
}
