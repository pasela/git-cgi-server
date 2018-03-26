package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
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

func writePIDFile(file string) error {
	pid := strconv.Itoa(os.Getpid())
	return ioutil.WriteFile(file, []byte(pid), 0644)
}

func removePIDFile(file string) error {
	err := os.Remove(file)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func subtreePath(uri string) string {
	if uri == "" || uri == "/" {
		return "/"
	}

	// ensure the trailing slash
	return path.Clean(uri) + "/"
}
