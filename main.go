package main

import (
	"fmt"
	"os/exec"
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

	fmt.Println("changed files:")
	for _, f := range files {
		fmt.Println(" ", f)
	}
}
