package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type pkgConfig struct {
	Name string   `json:"name"`
	Deps []string `json:"deps"`
}

func getChangedFiles(base, head string) ([]string, error) {
	out, err := exec.Command("git", "diff", "--name-only", base, head).Output()
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

	for {
		added := false
		for pkg, deps := range graph {
			if affected[pkg] {
				continue
			}
			for _, dep := range deps {
				if affected[dep] {
					affected[pkg] = true
					added = true
					break
				}
			}
		}
		if !added {
			break
		}
	}

	result := []string{}
	for pkg := range affected {
		result = append(result, pkg)
	}
	return result
}

func main() {
	base := flag.String("base", "HEAD~1", "base ref to diff against (e.g. main, HEAD~1, a commit SHA)")
	head := flag.String("head", "HEAD", "head ref (default: current HEAD)")
	format := flag.String("format", "plain", "output format: plain | json")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "cascade — affected package detector for monorepos\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n  cascade [flags]\n\nFlags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  cascade --base=main\n")
		fmt.Fprintf(os.Stderr, "  cascade --base=main --format=json\n")
	}

	flag.Parse()

	files, err := getChangedFiles(*base, *head)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: couldn't get changed files: %v\n", err)
		os.Exit(1)
	}

	graph, err := loadGraph("packages")
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
