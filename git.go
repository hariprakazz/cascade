package main

import (
	"os/exec"
	"strings"
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
