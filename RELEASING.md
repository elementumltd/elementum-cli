<!-- Copyright 2026 Elementum Ltd. All Rights Reserved.
Licensed under the Apache License, Version 2.0 -->

# Releasing

This project uses [Semantic Versioning](https://semver.org/), [Conventional Commits](https://www.conventionalcommits.org/), and automated tooling to manage releases.

## How It Works

Releases are fully automated through a pipeline of three components:

1. **PR title validation** — Every PR must have a title following the [Conventional Commits](https://www.conventionalcommits.org/) format (e.g., `feat: add widget export`). This is enforced by CI.

2. **release-please** — When PRs are squash-merged to `main`, [release-please](https://github.com/googleapis/release-please) reads the commit messages and maintains a **Release PR** that tracks pending changes, updates `CHANGELOG.md`, and determines the next version number.

3. **GoReleaser** — When the Release PR is merged, release-please creates a GitHub Release with a semver tag (e.g., `v1.2.0`). [GoReleaser](https://goreleaser.com/) then runs inline (in the same workflow) to build cross-platform binaries and upload them to the release.

## Semantic Versioning Rules

Version numbers follow `MAJOR.MINOR.PATCH`:

| Change Type | Version Bump | PR Title Examples |
|---|---|---|
| Breaking change | **Major** (1.0.0 → 2.0.0) | `feat!: redesign auth flow` or any commit with a `BREAKING CHANGE` footer |
| New feature | **Minor** (1.0.0 → 1.1.0) | `feat: add widget export command` |
| Bug fix | **Patch** (1.0.0 → 1.0.1) | `fix: resolve token refresh on expired credentials` |
| No release | — | `chore:`, `docs:`, `ci:`, `test:`, `refactor:` |

Only `feat` and `fix` trigger a release. Other types (`chore`, `docs`, `ci`, etc.) are included in the changelog but do not bump the version on their own.

## How to Create a Release

**Normal flow (recommended):**

1. Merge PRs to `main` with Conventional Commits titles
2. release-please automatically opens or updates a Release PR titled `chore(main): release X.Y.Z`
3. Review the Release PR — verify the changelog and version bump are correct
4. Merge the Release PR
5. release-please creates a GitHub Release and `vX.Y.Z` tag
6. GoReleaser builds and uploads binaries automatically

That's it. You do not need to manually create tags, write changelogs, or build binaries.

**Emergency / manual release:**

If you need to bypass the automated flow, manually trigger the Release Please workflow from the Actions tab, or create a tag and GitHub Release manually, then run GoReleaser locally:

```bash
goreleaser release --clean
```

You will need to set the `GITHUB_TOKEN` environment variable and manually update `CHANGELOG.md`.

## Conventional Commits Quick Reference

PR titles must follow this format:

```
type(optional-scope): description
```

**Allowed types:** `feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `perf`, `ci`, `build`, `revert`

**Breaking changes** can be indicated in two ways:
- Add `!` after the type: `feat!: remove deprecated API`
- Add a `BREAKING CHANGE:` footer in the PR body

## Build Artifacts

GoReleaser produces binaries for:

| Platform | Architecture | Archive |
|---|---|---|
| Linux | amd64 | `ei_linux_amd64.tar.gz` |
| macOS | amd64 | `ei_darwin_amd64.tar.gz` |
| macOS | arm64 (Apple Silicon) | `ei_darwin_arm64.tar.gz` |
| Windows | amd64 | `ei_windows_amd64.zip` |

Each release also includes a `checksums.txt` file with SHA256 hashes.

## Version Injection

The CLI version is set at build time via linker flags. GoReleaser runs:

```
go build -ldflags "-s -w -X main.version=X.Y.Z"
```

This replaces the default `var version = "dev"` in `main.go`. When building locally from source, the version will show as `dev`.

## Self-Update

Users can update their installed binary by running:

```bash
ei update
```

This checks for the latest GitHub Release and downloads the appropriate binary. The self-update mechanism uses the asset naming convention (`ei_<os>_<arch>`) to find the correct binary.

## Squash Merge Requirement

This project requires **squash merging** for all PRs to `main`. This ensures each PR becomes a single commit with the PR title as the commit message, which is what release-please reads to determine version bumps.

This is configured in GitHub Settings → General → Pull Requests.
