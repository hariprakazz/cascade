package main

import (
	"encoding/json"
	"fmt"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

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
