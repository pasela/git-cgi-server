package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
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

func stripPrefix(path, prefix string) string {
	if prefix == "" || prefix == "/" {
		return path
	} else {
		return strings.TrimPrefix(path, prefix)
	}
}

func isDir(file string) bool {
	s, err := os.Stat(file)
	if err != nil {
		return false
	}
	return s.IsDir()
}

func toURLPort(addr string) string {
	s := strings.SplitN(addr, ":", 2)
	port := s[1]

	if port == "80" || port == "443" {
		return ""
	} else {
		return ":" + port
	}
}
