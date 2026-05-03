// pyhotlint analyzes Python source files for inference-server hot-path
// hazards. The CLI takes file paths, directories, or shell-style globs;
// directories are walked recursively (skipping common venv / cache /
// build dirs). Findings are emitted as a JSON array on stdout.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kaeawc/pyhotlint/internal/config"
	"github.com/kaeawc/pyhotlint/internal/output"
	_ "github.com/kaeawc/pyhotlint/internal/rules" // registers rules
	v2 "github.com/kaeawc/pyhotlint/internal/rules/v2"
	"github.com/kaeawc/pyhotlint/internal/scanner"
	"github.com/kaeawc/pyhotlint/internal/walker"
)

var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	configPath := flag.String("config", "", "path to pyhotlint.yml; auto-discovered when empty")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: pyhotlint [--config FILE] <path> [path ...]")
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

	rules, err := resolveRules(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pyhotlint: %v\n", err)
		os.Exit(2)
	}

	files, err := walker.FindFiles(paths)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pyhotlint: %v\n", err)
		os.Exit(2)
	}

	findings, parseFailed := analyzeFiles(rules, files)
	if err := output.WriteJSON(os.Stdout, findings); err != nil {
		fmt.Fprintf(os.Stderr, "pyhotlint: %v\n", err)
		os.Exit(1)
	}
	exit := 0
	if parseFailed || len(findings) > 0 {
		exit = 1
	}
	os.Exit(exit)
}

// resolveRules loads the config (explicit path or auto-discovered),
// applies its overrides to the registry, and prints a banner + warnings
// to stderr. Returns the filtered rule set.
func resolveRules(explicit string) ([]*v2.Rule, error) {
	rules := v2.All()
	cfg, cfgPath, err := loadConfig(explicit)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return rules, nil
	}
	cfg.WarnUnknownRules(os.Stderr, rules)
	if cfgPath != "" && os.Getenv("PYHOTLINT_QUIET") == "" {
		fmt.Fprintf(os.Stderr, "pyhotlint: using config %s\n", cfgPath)
	}
	return cfg.Apply(rules), nil
}

// loadConfig resolves the config file. An explicit --config errors when
// the path is missing; auto-discovery silently returns nil when no
// config is found anywhere up the directory tree.
func loadConfig(explicit string) (*config.Config, string, error) {
	if explicit != "" {
		cfg, err := config.Load(explicit)
		return cfg, explicit, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, "", err
	}
	return config.Find(cwd)
}

// analyzeFiles runs every rule across every file and concatenates the
// results. Returns (findings, parseFailed); parseFailed is true if any
// file could not be parsed (a hard error reported to stderr).
func analyzeFiles(rules []*v2.Rule, files []string) ([]v2.Finding, bool) {
	var all []v2.Finding
	parseFailed := false
	for _, p := range files {
		pf, err := scanner.ParseFile(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "pyhotlint: %v\n", err)
			parseFailed = true
			continue
		}
		findings := v2.Run(rules, pf.Path, pf.Source, pf.Tree.RootNode())
		all = append(all, findings...)
		pf.Close()
	}
	return all, parseFailed
}
