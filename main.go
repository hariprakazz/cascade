package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type pkgConfig struct {
	Name string   `json:"name"`
	Deps []string `json:"deps"`
}

func getChangedFiles(base string) ([]string, error) {
	out, err := exec.Command("git", "diff", "--name-only", base).Output()
	if err != nil {
		return nil, err
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return []string{}, nil
	}
	return strings.Split(raw, "\n"), nil
}

func getPackage(filePath string) string {
	parts := strings.Split(filepath.ToSlash(filePath), "/")
	if len(parts) < 2 {
		return "root"
	}
	return parts[1]
}

func loadGraph(dir string) (map[string][]string, error) {
	graph := map[string][]string{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		cfgPath := filepath.Join(dir, e.Name(), "cascade.json")
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			continue
		}
		var cfg pkgConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			continue
		}
		graph[cfg.Name] = cfg.Deps
	}
	return graph, nil
}

func findAffected(changed []string, graph map[string][]string) []string {
	affected := map[string]bool{}
	for _, pkg := range changed {
		affected[pkg] = true
	}
	for pkg, deps := range graph {
		for _, dep := range deps {
			if affected[dep] {
				affected[pkg] = true
			}
		}
	}
	result := []string{}
	for pkg := range affected {
		result = append(result, pkg)
	}
	return result
}

func main() {
	files, err := getChangedFiles("HEAD")
	if err != nil {
		fmt.Println("couldn't get changes:", err)
		return
	}

	graph, err := loadGraph("packages")
	if err != nil {
		fmt.Println("couldn't load packages:", err)
		return
	}

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

	fmt.Println("affected packages:")
	for _, pkg := range affected {
		fmt.Println(" ", pkg)
	}
}
