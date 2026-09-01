# Contributing

Contributions are welcome. Please open an issue before submitting a pull request for non-trivial changes so we can discuss the approach first.

## Development setup

Clone the repository and make sure Go 1.26 is installed. The project uses [just](https://github.com/casey/just) as a command runner. Available recipes:

| Recipe | Description |
|--------|-------------|
| `just build` | Build the provider |
| `just test` | Run unit tests |
| `just testacc` | Run acceptance tests against a stub PVE API |
| `just lint` | Run golangci-lint |
| `just vuln` | Run govulncheck |
| `just check` | Run lint, govulncheck and unit tests together |
| `just docs` | Generate registry documentation |
| `just docs-check` | Fail if the committed docs are out of date |
| `just validate-examples` | Validate the Terraform examples |
| `just dco` | Check the sign-off trailers on your branch |

Running `just` without arguments opens an [fzf](https://github.com/junegunn/fzf) recipe picker. fzf must be installed for this to work.

If you prefer not to install `just`, the underlying commands are plain `go build`, `go test`, and `go run` invocations you can run directly.

## Nix flake with direnv support

If you have [Nix](https://nixos.org) and [direnv](https://direnv.net/) installed, the repository includes a flake that provides a dev shell with Go, Terraform, and `just` pinned and ready to use.

```sh
direnv allow   
```

From that point on, entering the directory automatically activates the environment. No manual `nix develop` or tool installation required.

## Submitting changes

Keep pull requests focused on a single change. Describe what the change does and why in the PR description.

### Sign-off (DCO)

Every commit must carry a `Signed-off-by` trailer certifying the [Developer Certificate of Origin](https://developercertificate.org/). Git adds one for you:

```sh
git commit -s
```

The trailer has to match the commit author:

```
Signed-off-by: Your Name <your.email@example.com>
```

CI rejects any commit a pull request introduces without one. Check your branch before pushing with `just dco`. If you forgot, add the trailers to the commits you already made and force-push:

```sh
git rebase --signoff origin/trunk
git push --force-with-lease
```

By signing off you agree that your contribution is licensed under the [MIT License](./LICENSE) that covers this repository.

### Commit signatures

Commits should also be cryptographically signed, via `git commit -S` or `git config commit.gpgsign true`. This is separate from the sign-off above: the signature attests who wrote the commit, the sign-off certifies you have the right to contribute it. With `commit.gpgsign` set, the `git rebase --signoff` above re-signs the commits it rewrites, so the two stay compatible.

## Reporting issues

Please include the provider version, Terraform version, and a minimal reproducible configuration when opening a bug report.
