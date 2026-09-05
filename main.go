package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

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
