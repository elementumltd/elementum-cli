# Elementum Skills

Three skills that teach AI agents how to operate Elementum platform.

## Skills

| Skill | Lines | Purpose |
|-------|-------|---------|
| **elementum-cli** | ~500 | Operate Elementum platform, discover various objects, test agents via `ei chat`, generate reports |

The `elementum-cli` skill has reference files for detailed HCL examples:

```
elementum-cli/
├── SKILL.md             # Core patterns, architecture, gotchas
├── debugging.md         # detailed automation debugging steps
└── deployment.md        # Cross-environment promotion
├── parallel-testing.md  # How to test various things in parallel
├── records.md           # ei CLI reference for records discovery/manipulations
```

## Installation

```bash
make skills-install
```

Installs to `~/.claude/skills/`, `~/.agents/skills/`, and `~/.cursor/skills/`.

```bash
make skills-package
```

Creates to `/dist/skills/elementum-cli.zip` for distribution.

## How It Works

Agents see each skill's `description` field and decide whether to load the full content based on what you're asking. The descriptions contain trigger phrases — natural language you'd type — so the agent can match your intent to the right skill.

If a skill isn't loading automatically, reference it by name or read the file directly.
