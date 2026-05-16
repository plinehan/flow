# flow

A small Git/GitHub workflow helper.

## Prerequisites

- [Go](https://go.dev/dl/) 1.22+
- [GitHub CLI](https://cli.github.com/) (`gh`), authenticated via `gh auth login`

## Install

Clone the repo and install the binary into your `GOBIN` (typically `~/go/bin`):

```sh
git clone https://github.com/plinehan/flow.git
cd flow
go install .
```

Make sure `~/go/bin` is on your `PATH`:

```sh
export PATH="$PATH:$(go env GOBIN):$(go env GOPATH)/bin"
```

## Development

Install [pre-commit](https://pre-commit.com/) and enable the hooks:

```sh
uv add --dev pre-commit
uv run pre-commit install
```

## Commands

- `flow branch [name]` — create and check out a new branch (`<user>/<name>` or a random slug)
- `flow create [-v]` — push the current branch and open a PR (`-v` opens it in the browser)
- `flow view` — open the current branch's PR in the browser
- `flow merge` — auto-squash-merge the current branch's PR, then return to the default (e.g. `main`)
  branch
- `flow rebase` — rebase the current branch onto the latest default branch
- `flow push` — force-push the current branch (refused on the default branch)
- `flow clean` — delete local branches whose PRs have been merged
