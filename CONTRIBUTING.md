# Contributing

Contributions are welcome. Please open an issue before submitting a pull request for non-trivial changes so we can discuss the approach first.

## Development setup

Clone the repository and make sure Go 1.26 is installed. The project uses [just](https://github.com/casey/just) as a command runner. Available recipes:

| Recipe | Description |
|--------|-------------|
| `just build` | Build the provider |
| `just test` | Run unit tests |
| `just lint` | Run golangci-lint |
| `just check` | Run lint and tests together |
| `just docs` | Generate registry documentation |

Running `just` without arguments opens an [fzf](https://github.com/junegunn/fzf) recipe picker. fzf must be installed for this to work.

If you prefer not to install `just`, the underlying commands are plain `go build`, `go test`, and `go run` invocations you can run directly.

## Submitting changes

Keep pull requests focused on a single change. Describe what the change does and why in the PR description. All commits must be signed.

## Reporting issues

Please include the provider version, Terraform version, and a minimal reproducible configuration when opening a bug report.
