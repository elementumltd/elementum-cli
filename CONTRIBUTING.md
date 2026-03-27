<!-- Copyright 2026 Elementum Ltd. All Rights Reserved.
Licensed under the Apache License, Version 2.0 -->

# Contributing to Elementum CLI

Thank you for your interest in contributing to the Elementum CLI! We welcome contributions from the community and are grateful for any help you can provide.

Please read and follow our [Code of Conduct](CODE_OF_CONDUCT.md) in all interactions with the project.

## How to Report Bugs

If you find a bug, please [open a GitHub Issue](https://github.com/elementumltd/elementum-cli/issues/new) and include:

- Steps to reproduce the problem
- Expected behavior vs. actual behavior
- CLI version (`ei --version`)
- Operating system and version
- Any relevant error messages or logs

For security vulnerabilities, please follow the process in [SECURITY.md](SECURITY.md) instead of filing a public issue.

## How to Request Features

Feature requests are welcome! [Open a GitHub Issue](https://github.com/elementumltd/elementum-cli/issues/new) with:

- A clear description of the feature
- The use case — what problem does it solve?
- Any examples of how it would work

## How to Submit a Pull Request

1. **Fork** the repo and create a branch from `main`
2. **Name your branch** using the convention `type/short-description`:
   - `feat/add-export-widgets`
   - `fix/auth-token-refresh`
   - `docs/update-readme`
   - `chore/update-dependencies`
3. **Write your code** — keep PRs focused on a single feature or fix
4. **Use Conventional Commits** for your commit messages ([spec](https://www.conventionalcommits.org/)):
   - `feat: add widget export command`
   - `fix: resolve token refresh on expired credentials`
   - `docs: update installation instructions`
   - `chore: update dependencies`
5. **Verify your changes** before submitting:
   ```bash
   go build ./...
   go test ./...
   ```
6. **Open a PR** with a clear description of what the change does and why
7. PRs require review from a code owner before merging

## Contributor License Agreement (CLA)

All contributors must sign a CLA before their first PR can be merged. When you submit a PR, the CLA bot will automatically comment with instructions. This is a one-time requirement.

> **Note:** CLA details coming soon.

## Development Setup

### Prerequisites

- Go 1.25.5 or later (see `go.mod` for the exact version)

### Clone and Build

```bash
git clone https://github.com/elementumltd/elementum-cli.git
cd elementum-cli
go build -o ei .
```

### Run Tests

```bash
go test ./...
```

### Regenerate GraphQL Client

If you modify any `.graphql` files in `internal/client/operations/`:

```bash
go generate ./internal/client/
```

## Community

- **Questions and discussion** — use [GitHub Discussions](https://github.com/elementumltd/elementum-cli/discussions) on this repo
- **Bugs and feature requests** — use [GitHub Issues](https://github.com/elementumltd/elementum-cli/issues)

## Important Note About Commits

Please use your personal GitHub account for contributions, not a work email address.
