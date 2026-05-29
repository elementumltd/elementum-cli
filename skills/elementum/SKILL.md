---
name: elementum
description: >
  Discover and manipulate all aspects of the Elementum platform using `ei` CLI
  Covers all Elementum platform work:
  data modeling, automation workflows, AI agents, views, filters, API integrations,
  record management, and cross-environment promotion.
  Triggers: "find app", "create app", "automation", "trigger", "workflow", "agent",
  "chatbot", "field", "view", "layout", "filter", "deploy", "records", "import",
  "API integration", "webhook", "Elementum", "ei CLI", "elementum",
  "data model", "refs", "publish", "workflow_revision", "skill", "tool".
allowed-tools: Bash(ei *)
---

# Elementum Platform Discovery and Manipulation

You enumerate and manipulate all components of the Elementum platform using the `ei` command-level interface (CLI).

# Basic Command Structure
# Noun-Verb Structure
`ei` follows a consistent noun-verb structure: you specify the object type (noun) and the action (verb). For example:

```bash
# List all apps
ei apps list
```

# Common Flags
These common flags apply to all `ei` commands:

| Command | Purpose |
|------|---------|
| `--client-id` | Client ID (overrides config) |
| `--client-secret` | Client Secret (overrides config) | 
| `--custom-api-url` | Custom API/OAuth URL for custom instances (e.g., http://localhost:8080) |
| `--custom-graphql-url` | Custom GraphQL URL for custom instances (e.g., http://localhost:3000) |
| `--debug` | Enable debug logging (shortcut for `--log-level debug`) |
| `--environment` | Organization environment slug, e.g. 'staging' (overrides config) |
| `--instance` | Instance/region: us, eu, stage, dev, custom (overrides config) |
| `--json` | Output in JSON format |
| `--log-level` | Log level: trace, debug, info, warn, error, off (default: info) |
| `--org` | Organization ID (overrides config) |
| `-h`, `--help` | Help for the current command |
| `-q`, `--quiet` | Suppress all non-error output (shortcut for --log-level error) |
| `-v` `--version` | Outputs version information |

# Prefer Structured Output
As agent, use `--json` flag to get structured output that's easier to parse and manipulate. For example:

```bash
# Get list of applications as JSON
ei apps list --json
```

# Reference for All Commands

| When you need to | Read |
|--|--|
| Manage Agent to Agent (A2A) Skills (read, relete) | [a2a-skills.md](a2a-skills.md) |
| Manage Agent Tools (create, read, update, delete) | [agent-tools.md](agent-tools.md) |
| Manage Agents (create, read, update, delete, show detail) | [agents.md](agents.md) |
| Manage AI Providers available (read) | [ai-providers.md](ai-providers.md) |
| Manage AI Services/model configurations (read) | [ai-services.md](ai-services.md) |
| Manage Approval Requests (read, approve, deny) | [approvals.md](approvals.md) |
| Manage Applications (create, read, update, delete, import, export) | [apps.md](apps.md) |
| Authenticate/Connect to Elementum organization/platform | [auth.md](auth.md) |
| Manage and operate Automations (list, show detail, config, run, check status) | [automations.md](automations.md) |
| Manage Categories (list) | [categories.md](categories.md) |
| Chat with Elemenutm hosted Agents (interactive) | [chat.md](chat.md) |
| Manage CloudLinks (list, explore, search) | [cloudlinks.md](cloudlinks.md) |
| Enrich your shell with auto-completion scripts (bash, fish, zsh, powershell) | [completion.md](completion.md) |
| Retrieve conversation history and display metrics | [conversation.md](conversation.md) |
| Manage Data Mines (list, export) | [datamines.md](datamines.md) |
| Manage Deployments (create, list, show detail, configure) | [deployments.md](deployments.md) |
| Manage Elements (create, read, update, delete, manage search tables) | [elements.md](elements.md) |
| Manage Environments (list) | [environments.md](environments.md) |
| Manage Feature Flags (list) | [feature-flags.md](feature-flags.md) |
| Manage Fields (create, read, update, delete, list values for dropdowns) | [fields.md](fields.md) |
| Manage File Readers (create, read, update, delete, show detail) | [file-readers.md](file-readers.md) |
| Manage stored Snowflake functions (procedures and UDFs) (read, show detail) | [functions.md](functions.md) |
| Run GraphQL queries | [graphql.md](graphql.md) |
| Manage Groups (read, export) | [groups.md](groups.md) |
| Manage failures in automations (list, show detail) | [interventions.md](interventions.md) |
| Manage Layouts (create, read, update, delete, show detail) | [layouts.md](layouts.md) |
| Manage objects (read, search, show detail) | [objects.md](objects.md) |
| Manage Phone Providers (read) | [phone-providers.md](phone-providers.md) |
| Manage Phone Services (phone numbers linked to agents) (read, delete) | [phone-services.md](phone-services.md) |
| Manage records via CLI (create, read, update, delete, import, export) | [records.md](records.md) |
| Display value references (refs) from Elementum resources for an Automation | [refs.md](refs.md) |
| Manage Search Tables (query, rebuild, refresh) | [search-tables.md](search-tables.md) |
| Manage Agentic Skill Tools (create, read, update, delete) | [skill-tools.md](skill-tools.md) |
| Manage Agentic Skills (create, read, update, delete, show detail) | [skills.md](skills.md) |
| Manage Tables (create, read, export) | [tables.md](tables.md) |
| Manage Tasks (read, delete, export) | [tasks.md](tasks.md) |
| Update Elementum CLI | [update.md](update.md) |
| Manage Users in Organization (list, search) | [users.md](users.md) |
| Document application configuration | [documentation.md](documentation.md) | 
| TODO | [xdebugging.md](xdebugging.md) |
| TODO | [xdeployment.md](xdeployment.md) |
| TODO | [xparallel-testing.md](xparallel-testing.md) |
| TODO Deploy apps across environments | [deployment.md](deployment.md) |
