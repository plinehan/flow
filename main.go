package main

import (
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/docker/docker/pkg/namesgenerator"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "branch" || len(os.Args) > 3 {
		fmt.Fprintf(os.Stderr, "usage: glit branch [name]\n")
		os.Exit(2)
	}

	user, err := githubUsername()
	if err != nil {
		fmt.Fprintf(os.Stderr, "glit: %v\n", err)
		os.Exit(1)
	}

	var branch string
	if len(os.Args) == 3 {
		name := strings.TrimSpace(os.Args[2])
		if name == "" {
			fmt.Fprintf(os.Stderr, "glit: branch name must not be empty\n")
			os.Exit(2)
		}
		branch = fmt.Sprintf("%s/%s", user, name)
	} else {
		slug, err := randomBranchSlug()
		if err != nil {
			fmt.Fprintf(os.Stderr, "glit: %v\n", err)
			os.Exit(1)
		}
		branch = fmt.Sprintf("%s/%s", user, slug)
	}

	cmd := exec.Command("git", "checkout", "-b", branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "glit: git checkout -b: %v\n", err)
		os.Exit(1)
	}
}

const alnum = "abcdefghijklmnopqrstuvwxyz0123456789"

func randomBranchSlug() (string, error) {
	base := strings.ReplaceAll(namesgenerator.GetRandomName(0), "_", "-")
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	suffix := make([]byte, len(b))
	for i := range b {
		suffix[i] = alnum[int(b[i])%len(alnum)]
	}
	return base + "-" + string(suffix), nil
}

func githubUsername() (string, error) {
	out, err := exec.Command("git", "config", "--get", "github.user").Output()
	if err == nil {
		if u := strings.TrimSpace(string(out)); u != "" {
			return u, nil
		}
	}

	out, err = exec.Command("gh", "api", "user", "--jq", ".login").Output()
	if err != nil {
		return "", fmt.Errorf("GitHub username: set `git config github.user <login>` or run `gh auth login`")
	}
	if u := strings.TrimSpace(string(out)); u != "" {
		return u, nil
	}
	return "", fmt.Errorf("gh returned empty login")
}
