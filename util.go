package main

import (
	"bytes"
	"os/exec"
	"path/filepath"
)

func findGitPath() (string, error) {
	return exec.LookPath("git")
}

func findBackendCGI() (string, error) {
	// git, err := findGitPath()
	// if err != nil {
	// 	return "", err
	// }

	dir, err := exec.Command("git", "--exec-path").Output()
	if err != nil {
		return "", err
	}
	dir = bytes.TrimRight(dir, "\r\n")

	return filepath.Join(string(dir), "git-http-backend"), nil
}
