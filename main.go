package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

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

func main() {
	files, err := getChangedFiles("HEAD")
	if err != nil {
		fmt.Println("couldn't get changes:", err)
		return
	}

	if len(files) == 0 {
		fmt.Println("nothing changed")
		return
	}

	seen := map[string]bool{}
	fmt.Println("affected packages:")
	for _, f := range files {
		pkg := getPackage(f)
		if !seen[pkg] {
			seen[pkg] = true
			fmt.Println(" ", pkg)
		}
	}
}
