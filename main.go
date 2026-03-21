package main

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var adjectives = []string{
	"swift", "quiet", "brave", "calm", "clever", "curious", "eager", "gentle",
	"happy", "lucky", "mighty", "nimble", "proud", "sharp", "silent", "sturdy",
	"wild", "wise", "bold", "bright", "cosmic", "dapper", "frosty", "golden",
}

var nouns = []string{
	"badger", "beacon", "canyon", "cedar", "comet", "coral", "crane", "delta",
	"falcon", "fjord", "harbor", "heron", "island", "juniper", "lagoon", "meadow",
	"orca", "penguin", "quartz", "raven", "river", "sapphire", "summit", "willow",
}

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

	branch := fmt.Sprintf("%s/%s-%s", user, adjectives[randInt(len(adjectives))], nouns[randInt(len(nouns))])

	cmd := exec.Command("git", "checkout", "-b", branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "glit: git checkout -b: %v\n", err)
		os.Exit(1)
	}
}

func randInt(n int) int {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return int(binary.BigEndian.Uint64(b[:]) % uint64(n))
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
