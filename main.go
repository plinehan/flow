package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/docker/docker/pkg/namesgenerator"
)

func main() {
	if len(os.Args) != 3 || os.Args[1] != "flow" || os.Args[2] != "branch" {
		fmt.Fprintf(os.Stderr, "usage: glit flow branch\n")
		os.Exit(2)
	}

	user, err := githubUsername()
	if err != nil {
		fmt.Fprintf(os.Stderr, "glit: %v\n", err)
		os.Exit(1)
	}

	branch := fmt.Sprintf("%s/%s", user, namesgenerator.GetRandomName(0))

	cmd := exec.Command("git", "checkout", "-b", branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "glit: git checkout -b: %v\n", err)
		os.Exit(1)
	}
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
