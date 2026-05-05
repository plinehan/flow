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

	push := exec.Command("git", "push", "-u", "origin", branch)
	push.Stdout = os.Stdout
	push.Stderr = os.Stderr
	if err := push.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "glit: git push: %v\n", err)
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

	cmd := exec.Command("gh", "pr", "view", "--web", branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "glit: gh pr view: %v\n", err)
		os.Exit(1)
	}
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

	merge := exec.Command("gh", "pr", "merge", branch, "--squash")
	merge.Stdout = os.Stdout
	merge.Stderr = os.Stderr
	merge.Stdin = os.Stdin
	if err := merge.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "glit: gh pr merge: %v\n", err)
		os.Exit(1)
	}

	runGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "glit: git %s: %v\n", strings.Join(args, " "), err)
			os.Exit(1)
		}
	}

	runGit("checkout", def)
	runGit("push", "origin", "--delete", branch)
	runGit("branch", "-D", branch)
	runGit("pull", "--rebase")
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

	runGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "glit: git %s: %v\n", strings.Join(args, " "), err)
			os.Exit(1)
		}
	}

	runGit("fetch", "origin", def)
	runGit("rebase", "origin/"+def)
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
		if !merged[b] {
			continue
		}
		cmd := exec.Command("git", "branch", "-D", b)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "glit: git branch -D %s: %v\n", b, err)
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
