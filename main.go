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

const grey = "\x1b[90m"
const reset = "\x1b[0m"

// run executes a command with its stdio connected to the terminal, logging it in grey first.
func run(name string, args ...string) {
	fmt.Printf("%s> %s %s%s\n", grey, name, strings.Join(args, " "), reset)
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "glit: %s %s: %v\n", name, strings.Join(args, " "), err)
		os.Exit(1)
	}
}


func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: glit <command> [args]\n")
		fmt.Fprintf(os.Stderr, "commands: branch, create, view, merge, clean, rebase\n")
		os.Exit(2)
	}

	switch os.Args[1] {
	case "branch":
		cmdBranch(os.Args[2:])
	case "create":
		cmdCreate(os.Args[2:])
	case "view":
		cmdView(os.Args[2:])
	case "merge":
		cmdMerge(os.Args[2:])
	case "clean":
		cmdClean(os.Args[2:])
	case "rebase":
		cmdRebase(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "glit: unknown command %q\n", os.Args[1])
		fmt.Fprintf(os.Stderr, "commands: branch, create, view, merge, clean, rebase\n")
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

	run("git", "checkout", "-b", branch)
}

func cmdCreate(args []string) {
	view := false
	var filtered []string
	for _, a := range args {
		if a == "-v" {
			view = true
		} else {
			filtered = append(filtered, a)
		}
	}
	args = filtered

	if len(args) > 1 {
		fmt.Fprintf(os.Stderr, "usage: glit create [-v] [branch]\n")
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
		var err error
		branch, err = currentBranch()
		if err != nil {
			fmt.Fprintf(os.Stderr, "glit: %v\n", err)
			os.Exit(1)
		}
	}

	if err := assertNotDefaultBranch(branch); err != nil {
		fmt.Fprintf(os.Stderr, "glit: %v\n", err)
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

	run("git", "push", "-u", "origin", branch)

	title, body, err := commitTitleBody(branch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "glit: %v\n", err)
		os.Exit(1)
	}

	run("gh", "pr", "create", "--head", branch, "--title", title, "--body", body)

	if view {
		run("gh", "pr", "view", "--web", branch)
	}
}

func cmdView(args []string) {
	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "usage: glit view\n")
		os.Exit(2)
	}

	branch, err := currentBranch()
	if err != nil {
		fmt.Fprintf(os.Stderr, "glit: %v\n", err)
		os.Exit(1)
	}

	if err := assertNotDefaultBranch(branch); err != nil {
		fmt.Fprintf(os.Stderr, "glit: %v\n", err)
		os.Exit(1)
	}

	run("gh", "pr", "view", "--web", branch)
}

func cmdMerge(args []string) {
	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "usage: glit merge\n")
		os.Exit(2)
	}

	branch, err := currentBranch()
	if err != nil {
		fmt.Fprintf(os.Stderr, "glit: %v\n", err)
		os.Exit(1)
	}

	if err := assertNotDefaultBranch(branch); err != nil {
		fmt.Fprintf(os.Stderr, "glit: %v\n", err)
		os.Exit(1)
	}

	def, err := defaultBranch()
	if err != nil {
		fmt.Fprintf(os.Stderr, "glit: %v\n", err)
		os.Exit(1)
	}

	run("gh", "pr", "merge", branch, "--squash", "--auto")
	run("git", "checkout", def)
	run("git", "pull", "--rebase")
}

func cmdRebase(args []string) {
	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "usage: glit rebase\n")
		os.Exit(2)
	}

	def, err := defaultBranch()
	if err != nil {
		fmt.Fprintf(os.Stderr, "glit: %v\n", err)
		os.Exit(1)
	}

	run("git", "fetch", "origin", def)
	run("git", "rebase", "origin/"+def)
}

func cmdClean(args []string) {
	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "usage: glit clean\n")
		os.Exit(2)
	}

	def, err := defaultBranch()
	if err != nil {
		fmt.Fprintf(os.Stderr, "glit: %v\n", err)
		os.Exit(1)
	}

	out, err := exec.Command("git", "branch", "--format=%(refname:short)").Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "glit: git branch: %v\n", err)
		os.Exit(1)
	}
	var local []string
	for _, b := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if b = strings.TrimSpace(b); b != "" && b != def {
			local = append(local, b)
		}
	}
	if len(local) == 0 {
		return
	}

	out, err = exec.Command("gh", "pr", "list", "--state", "merged", "--json", "headRefName", "--limit", "1000").Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "glit: gh pr list: %v\n", err)
		os.Exit(1)
	}
	var prs []struct {
		HeadRefName string `json:"headRefName"`
	}
	if err := json.Unmarshal(out, &prs); err != nil {
		fmt.Fprintf(os.Stderr, "glit: parsing gh output: %v\n", err)
		os.Exit(1)
	}
	merged := make(map[string]bool, len(prs))
	for _, pr := range prs {
		merged[pr.HeadRefName] = true
	}

	for _, b := range local {
		if merged[b] {
			run("git", "branch", "-D", b)
		}
	}
}

func currentBranch() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("could not determine current branch: %v", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// assertNotDefaultBranch returns an error if branch is the repo's default branch.
func assertNotDefaultBranch(branch string) error {
	def, err := defaultBranch()
	if err != nil {
		return err
	}
	if branch == def {
		return fmt.Errorf("%q is the default branch", branch)
	}
	return nil
}

// defaultBranch returns the repository's default branch name as reported by GitHub.
func defaultBranch() (string, error) {
	out, err := exec.Command("gh", "repo", "view", "--json", "defaultBranchRef", "--jq", ".defaultBranchRef.name").Output()
	if err != nil {
		return "", fmt.Errorf("gh repo view: %v", err)
	}
	if name := strings.TrimSpace(string(out)); name != "" {
		return name, nil
	}
	return "", fmt.Errorf("gh returned empty defaultBranchRef")
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
