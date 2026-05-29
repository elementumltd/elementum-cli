# Elementum Skills

Three skills that teach AI agents how to operate Elementum platform.

## Skills

| Skill | Lines | Purpose |
|-------|-------|---------|
| **elementum** | ~500 | Operate Elementum platform, discover various objects, test agents via `ei chat`, generate reports |

The `elementum` skill has reference files for detailed HCL examples:

```
elementum/
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

Creates to `/dist/skills/elementumß.zip` for distribution.

## How It Works

Agents see each skill's `description` field and decide whether to load the full content based on what you're asking. The descriptions contain trigger phrases — natural language you'd type — so the agent can match your intent to the right skill.

If a skill isn't loading automatically, reference it by name or read the file directly.

## Full list of commands

| link | name | purpose |
|--|--|-- |
| [a2a-skills.md](elementum/a2a-skills.md) | a2a-skills | Manage A2A (Agent-to-Agent) skills
| [agent-tools.md](elementum/agent-tools.md) | agent-tools | Manage agent tools 
| [agents.md](elementum/agents.md) | agents | Manage agents 
| [ai-providers.md](elementum/ai-providers.md) | ai-providers | Manage AI providers 
| [ai-services.md](elementum/ai-services.md) | ai-services | Manage AI services 
| remove | apply | Run terraform/tofu apply with Elementum provider credentials 
| [approvals.md](elementum/approvals.md) | approvals | Manage approvals for records
| [apps.md](elementum/apps.md) | apps | Manage apps  
| [auth.md](elementum/auth.md) | auth | Manage authentication 
| [automations.md](elementum/automations.md) | automations | Manage and monitor automations 
| [categories.md](elementum/categories.md) | categories | Manage categories 
| [chat.md](elementum/chat.md) | chat | Start a conversation with an Elementum agent 
| [cloudlinks.md](elementum/cloudlinks.md) | cloudlinks | Manage cloudlinks 
| [completion.md](elementum/completion.md) | completion | Generate the autocompletion script for the specified shell 
| [conversation.md](elementum/conversation.md) | conversation | Analyze agent conversation performance 
| [datamines.md](elementum/datamines.md) | datamines | Manage datamines 
| [deployments.md](elementum/deployments.md) | deployments | Manage deployments 
| remove | destroy | Run terraform/tofu destroy with Elementum provider credentials 
| [elements.md](elementum/elements.md) | elements | Manage elements 
| [environments.md](elementum/environments.md) | environments | Manage environments 
| [feature-flags.md](elementum/feature-flags.md) | feature-flags | Manage feature flags 
| [fields.md](elementum/fields.md) | fields | Field operations 
| [file-readers.md](elementum/file-readers.md) | file-readers | Manage file readers 
| [functions.md](elementum/functions.md) | functions | Manage stored Snowflake functions 
| [graphql.md](elementum/graphql.md) | graphql | Execute a raw GraphQL query or mutation 
| [groups.md](elementum/groups.md) | groups | Manage groups 
| remove | import | Run terraform/tofu import with Elementum provider credentials 
| [interventions.md](elementum/interventions.md) | interventions | Manage automation interventions 
| [layouts.md](elementum/layouts.md) | layouts | Layout/stage operations 
| [objects.md](elementum/objects.md) | objects | Manage objects 
| [phone-providers.md](elementum/phone-providers.md) | phone-providers | Manage phone providers 
| [phone-services.md](elementum/phone-services.md) | phone-services | Manage phone services 
| remove | plan | Run terraform/tofu plan with Elementum provider credentials 
| [records.md](elementum/records.md) | records | Manage records 
| [refs.md](elementum/refs.md) | refs | Show available value references from Terraform state or remote API 
| [search-tables.md](elementum/search-tables.md) | search-tables | Manage AI search tables 
| [skill-tools.md](elementum/skill-tools.md) | skill-tools | Manage agentic skill tools 
| [skills.md](elementum/skills.md) | skills | Manage agentic skills 
| [tables.md](elementum/tables.md) | tables | Manage tables 
| [tasks.md](elementum/tasks.md) | tasks | Manage tasks 
| [update.md](elementum/update.md) | update | Update ei to the latest version 
| [users.md](elementum/users.md) | users | Manage users 