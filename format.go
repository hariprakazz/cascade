package main

import "encoding/json"

func formatGithubMatrix(affected []string) (string, error) {
	out, err := json.Marshal(affected)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
