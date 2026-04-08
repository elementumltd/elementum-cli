# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## 1.0.0 (2026-04-08)


### Bug Fixes

* bump Go to 1.25.9 to resolve stdlib vulnerabilities ([#17](https://github.com/elementumltd/elementum-cli/issues/17)) ([87fb709](https://github.com/elementumltd/elementum-cli/commit/87fb7090b37f0f0e852a04a373c3bed29a617850))
* remove generated code check from CI ([#15](https://github.com/elementumltd/elementum-cli/issues/15)) ([7b1950b](https://github.com/elementumltd/elementum-cli/commit/7b1950bf6b3e11370410147e77c0ae63494198e7))

## [Unreleased]

### Added

- Initial open source release of the Elementum CLI (`ei`)
- Authentication with OS keychain integration (`ei auth login`)
- App discovery and tree view (`ei show app`)
- Export to Terraform/OpenTofu import configurations (`ei export app`)
- Element, group, and CloudLink export support
- Terraform/Tofu helper commands (`ei plan`, `ei apply`, `ei destroy`)
- Stored Snowflake function management (`ei functions list`, `ei functions delete`)
- A2A skill management (`ei a2a-skills list`, `ei a2a-skills delete`)
- Automation status and analysis commands
- Record CRUD operations
- Agent, skill, and tool management
- Interactive UI powered by Charm (Bubble Tea, Lip Gloss, Huh)
- Type-safe GraphQL client generated with genqlient
- Self-update mechanism
- Multi-profile configuration support
