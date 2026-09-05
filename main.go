package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func resolveEffectiveBase(base, head string) (string, error) {
	out, err := exec.Command("git", "rev-list", "--parents", "-n", "1", head).Output()
	if err != nil {
		return "", err
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) <= 2 {
		return base, nil
	}

	firstParent := fields[1]
	secondParent := fields[2]

	mergeBaseOut, err := exec.Command("git", "merge-base", firstParent, secondParent).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(mergeBaseOut)), nil
}

func getChangedFiles(base, head string) ([]string, error) {
	out, err := exec.Command("git", "diff", "--name-status", base, head).Output()
	if err != nil {
		return nil, err
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return []string{}, nil
	}

	paths := []string{}
	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Split(line, "\t")
		status := fields[0]
		if strings.HasPrefix(status, "R") {
			oldPath, newPath := fields[1], fields[2]
			paths = append(paths, oldPath, newPath)
			continue
		}
		paths = append(paths, fields[1])
	}
	return paths, nil
}

func getPackage(filePath string) string {
	parts := strings.Split(filepath.ToSlash(filePath), "/")
	if len(parts) < 2 {
		return "root"
	}
	return parts[1]
}

func readModulePath(gomodPath string) (string, error) {
	data, err := os.ReadFile(gomodPath)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "module ")), nil
		}
	}
	return "", fmt.Errorf("no module directive found in %s", gomodPath)
}

func parseImports(filePath, modulePrefix string) ([]string, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}

	deps := []string{}
	for _, imp := range node.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if strings.HasPrefix(path, modulePrefix) {
			dep := strings.TrimPrefix(path, modulePrefix)
			deps = append(deps, dep)
		}
	}
	return deps, nil
}

func parseCargoDeps(filePath string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	deps := []string{}
	inDeps := false
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			inDeps = trimmed == "[dependencies]"
			continue
		}
		if !inDeps || trimmed == "" {
			continue
		}
		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		deps = append(deps, name)
	}
	return deps, nil
}

type dubConfig struct {
	Dependencies map[string]interface{} `json:"dependencies"`
}

func parseDubDeps(filePath string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var cfg dubConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	deps := []string{}
	for name := range cfg.Dependencies {
		deps = append(deps, name)
	}
	sort.Strings(deps)
	return deps, nil
}

func loadGraph(dir, modulePrefix string) (map[string][]string, error) {
	graph := map[string][]string{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pkgName := e.Name()
		pkgDir := filepath.Join(dir, pkgName)

		files, err := os.ReadDir(pkgDir)
		if err != nil {
			continue
		}

		depSet := map[string]bool{}
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			name := f.Name()

			if name == "dub.json" {
				deps, err := parseDubDeps(filepath.Join(pkgDir, name))
				if err != nil {
					continue
				}
				for _, d := range deps {
					depSet[d] = true
				}
				continue
			}

			if name == "Cargo.toml" {
				deps, err := parseCargoDeps(filepath.Join(pkgDir, name))
				if err != nil {
					continue
				}
				for _, d := range deps {
					depSet[d] = true
				}
				continue
			}

			if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			match, err := build.Default.MatchFile(pkgDir, name)
			if err != nil || !match {
				continue
			}
			deps, err := parseImports(filepath.Join(pkgDir, name), modulePrefix)
			if err != nil {
				continue
			}
			for _, d := range deps {
				depSet[d] = true
			}
		}

		deps := []string{}
		for d := range depSet {
			deps = append(deps, d)
		}
		sort.Strings(deps)
		graph[pkgName] = deps
	}
	return graph, nil
}

func findAffected(changed []string, graph map[string][]string) []string {
	reverseDeps := map[string][]string{}
	for pkg, deps := range graph {
		for _, dep := range deps {
			reverseDeps[dep] = append(reverseDeps[dep], pkg)
		}
	}

	affected := map[string]bool{}
	queue := []string{}
	for _, pkg := range changed {
		if _, ok := graph[pkg]; !ok {
			fmt.Fprintf(os.Stderr, "warning: %q in changed set but not found in graph, skipping\n", pkg)
			continue
		}
		affected[pkg] = true
		queue = append(queue, pkg)
	}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		for _, dependent := range reverseDeps[curr] {
			if !affected[dependent] {
				affected[dependent] = true
				queue = append(queue, dependent)
			}
		}
	}

	result := []string{}
	for pkg := range affected {
		result = append(result, pkg)
	}
	return result
}

func formatGithubMatrix(affected []string) (string, error) {
	out, err := json.Marshal(affected)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func main() {
	base := flag.String("base", "HEAD~1", "base ref to diff against (e.g. main, HEAD~1, a commit SHA)")
	head := flag.String("head", "HEAD", "head ref (default: current HEAD)")
	format := flag.String("format", "plain", "output format: plain | json | github-matrix")
	pkgsDir := flag.String("packages-dir", "packages", "directory containing packages, relative to repo root")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "cascade, affected package detector for monorepos\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n  cascade [flags]\n\nFlags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  cascade --base=main\n")
		fmt.Fprintf(os.Stderr, "  cascade --base=main --format=json\n")
	fmt.Fprintf(os.Stderr, "  cascade --base=main --format=github-matrix\n")
	}

	flag.Parse()

	if out, err := exec.Command("git", "rev-parse", "--is-shallow-repository").Output(); err == nil {
		if strings.TrimSpace(string(out)) == "true" {
			if _, err := exec.Command("git", "fetch", "--unshallow", "-q").CombinedOutput(); err != nil {
				fmt.Fprintf(os.Stderr, "error: repo is a shallow clone and could not fetch full history: %v\n", err)
				os.Exit(1)
			}
		}
	}

	effectiveBase := *base
	if *base == "HEAD~1" {
		resolved, err := resolveEffectiveBase(*base, *head)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: couldn't resolve merge base: %v\n", err)
			os.Exit(1)
		}
		effectiveBase = resolved
	}

	files, err := getChangedFiles(effectiveBase, *head)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: couldn't get changed files: %v\n", err)
		os.Exit(1)
	}

	modulePath, err := readModulePath("go.mod")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: couldn't read go.mod: %v\n", err)
		os.Exit(1)
	}
	modulePrefix := modulePath + "/" + *pkgsDir + "/"

	graph, err := loadGraph(*pkgsDir, modulePrefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: couldn't load packages: %v\n", err)
		os.Exit(1)
	}

	start := time.Now()

	changed := []string{}
	seen := map[string]bool{}
	for _, f := range files {
		pkg := getPackage(f)
		if !seen[pkg] && pkg != "root" {
			seen[pkg] = true
			changed = append(changed, pkg)
		}
	}

	affected := findAffected(changed, graph)
	elapsed := time.Since(start)
	skipped := len(graph) - len(affected)

	sort.Strings(affected)

	switch *format {
	case "json":
		out, _ := json.MarshalIndent(map[string][]string{"affected": affected}, "", "  ")
		fmt.Println(string(out))
	case "github-matrix":
		out, err := formatGithubMatrix(affected)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: couldn't build github-matrix output: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(out)
	default:
		cyan := "\033[36m"
		green := "\033[32m"
		yellow := "\033[33m"
		gray := "\033[90m"
		reset := "\033[0m"

		fmt.Printf("\n%s cascade results%s\n", cyan, reset)
		fmt.Printf("%s─────────────────%s\n", gray, reset)

		if len(affected) == 0 {
			fmt.Printf("%s✔ nothing affected%s\n", green, reset)
		} else {
			for _, pkg := range affected {
				fmt.Printf("  %s▸%s %s\n", yellow, reset, pkg)
			}
		}

		fmt.Printf("%s─────────────────%s\n", gray, reset)
		fmt.Printf("  %s✔ %d affected  •  %d skipped  •  %s%s\n",
			green, len(affected), skipped, elapsed.Round(time.Microsecond), reset)
		fmt.Println()
	}
}
