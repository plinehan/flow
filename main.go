package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/docker/docker/pkg/namesgenerator"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: glit <command> [args]\n")
		fmt.Fprintf(os.Stderr, "commands: branch, create\n")
		os.Exit(2)
	}

	switch os.Args[1] {
	case "branch":
		cmdBranch(os.Args[2:])
	case "create":
		cmdCreate(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "glit: unknown command %q\n", os.Args[1])
		fmt.Fprintf(os.Stderr, "commands: branch, create\n")
		os.Exit(2)
	}
}

func cmdBranch(args []string) {
	if len(args) > 1 {
		fmt.Fprintf(os.Stderr, "usage: glit branch [name]\n")
		os.Exit(2)
	}

	user, err := githubUsername()
	if err != nil {
		fmt.Fprintf(os.Stderr, "glit: %v\n", err)
		os.Exit(1)
	}

	var branch string
	if len(args) == 1 {
		name := strings.TrimSpace(args[0])
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

func cmdCreate(args []string) {
	if len(args) > 1 {
		fmt.Fprintf(os.Stderr, "usage: glit create [branch]\n")
		os.Exit(2)
	}

	var branch string
	if len(args) == 1 {
		branch = strings.TrimSpace(args[0])
		if branch == "" {
			fmt.Fprintf(os.Stderr, "glit: branch name must not be empty\n")
			os.Exit(2)
		}
	} else {
		out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
		if err != nil {
			fmt.Fprintf(os.Stderr, "glit: could not determine current branch: %v\n", err)
			os.Exit(1)
		}
		branch = strings.TrimSpace(string(out))
	}

	if branch == "main" || branch == "master" {
		fmt.Fprintf(os.Stderr, "glit: cannot create a PR for %q\n", branch)
		os.Exit(1)
	}

	existing, err := prForBranch(branch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "glit: %v\n", err)
		os.Exit(1)
	}
	if existing != 0 {
		fmt.Fprintf(os.Stderr, "glit: branch %q already has PR #%d\n", branch, existing)
		os.Exit(1)
	}

	title, body, err := commitTitleBody(branch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "glit: %v\n", err)
		os.Exit(1)
	}

	cmd := exec.Command("gh", "pr", "create", "--head", branch, "--title", title, "--body", body)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "glit: gh pr create: %v\n", err)
		os.Exit(1)
	}
}

// commitTitleBody returns the first line and remainder of the tip commit message on branch.
func commitTitleBody(branch string) (title, body string, err error) {
	out, err := exec.Command("git", "log", "-1", "--format=%B", branch).Output()
	if err != nil {
		return "", "", fmt.Errorf("git log: %v", err)
	}
	msg := strings.TrimRight(string(out), "\n")
	idx := strings.IndexByte(msg, '\n')
	if idx == -1 {
		return msg, "", nil
	}
	return msg[:idx], strings.TrimLeft(msg[idx:], "\n"), nil
}

// prForBranch returns the PR number for the given branch, or 0 if none exists.
func prForBranch(branch string) (int, error) {
	out, err := exec.Command("gh", "pr", "list", "--head", branch, "--json", "number").Output()
	if err != nil {
		return 0, fmt.Errorf("gh pr list: %v", err)
	}
	var prs []struct {
		Number int `json:"number"`
	}
	if err := json.Unmarshal(out, &prs); err != nil {
		return 0, fmt.Errorf("parsing gh output: %v", err)
	}
	if len(prs) == 0 {
		return 0, nil
	}
	return prs[0].Number, nil
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
