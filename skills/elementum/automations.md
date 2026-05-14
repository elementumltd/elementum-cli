# Automations

Commands for listing, monitoring and managing Elementum automations.

## Available Commands

| Command | Purpose |
|--|--|
| config | Render automation configuration flow |
| delete | Delete an automation |
| list | List automations for an app or element |
| run | Run an on-demand or webhook automation |
| show | Show automation details |
| status | Show automation execution status |

## config - TODO

Display the full configuration of an automation as a visual flow.

Shows the trigger and all tasks in execution order, with proper
rendering of switch branches and for_each loops.

Output modes:
  (default)     CLI visual flow with connectors and nesting
  --json        Full structured JSON of the automation config
  --timeline    Alias for default CLI visual flow
  --html        Interactive HTML visualization opened in browser

```bash
ei automation config abc-123-uuid
ei automation config abc-123-uuid --json
ei automation config abc-123-uuid --html
ei automation config abc-123-uuid --timeline
```

```bash
# TODO Open interactive HTML visualization in browser
ei automations config <automation-id> --html

# TODO Show CLI visual flow (same as default)
ei automations config <automation-id> --timeline
```

## delete - TODO

Delete an automation by ID or by app namespace and automation name.

Usage patterns:
  ei automation delete <automation-id>
  ei automation delete <app-namespace> <automation-name>

```bash
# Delete by UUID
ei automation delete 9063aed1-bf8c-430d-882f-8c502355a3c7

# Delete by app namespace + automation name
ei automation delete myapp "Process Request"

# Skip confirmation prompt
ei automation delete myapp "Process Request" --force

# Preview what would be deleted without actually deleting
ei automation delete myapp "Process Request" --dry-run
```

```bash
# TODO Show what would be deleted without actually deleting
ei automations delete (<automation-id> | <app-namespace> <automation-name>) --dry-run

# TODO Skip confirmation prompt
ei automations delete (<automation-id> | <app-namespace> <automation-name>) --force
```

## list - TODO

Display automations configured for the specified app or element.

```bash
# TODO Show full task configuration for each automation
ei automations list <app-namespace> --details
```

## run - TODO

Triggers an automation with webhook or on-demand trigger.

This command supports two execution modes:

1. Webhook mode (default): Triggers via the automation's webhook URL
2. Widget mode: Triggers via a run automation widget on a specific record

```bash
# Run via webhook and wait for completion (default)
ei automation run abc-123-uuid

# Run without waiting
ei automation run abc-123-uuid --no-wait

# Run via widget on a specific record
ei automation run abc-123-uuid --widget def-456 --record "app-id:REC-001"

# JSON output
ei automation run abc-123-uuid --json
```

```bash
# TODO Don't wait for execution to complete
ei automations run <automation-id> --no-wait

# TODO Record ID in format 'aspectID:handle' (requires --widget)
ei automations run <automation-id> --record "<value>"

# TODO Widget ID for on-demand execution (requires --record)
ei automations run <automation-id> --widget "<value>"
```

## show - TODO

Display details about an automation workflow.

By default, reads from Terraform state in the current directory.
Use --from-remote to query the Elementum API directly.

Shows the automation structure including:
- Trigger type and configuration
- Tasks with their refs
- Workflow status

```bash
# From state (default)
ei automations show my_automation
ei automations show abc-123-uuid
ei automations show --state-file ./terraform.tfstate my_automation

# From remote API
ei automations show --from-remote abc-123-uuid
ei automations show --from-remote abc-123-uuid --json
```

```bash
# TODO Query automation from Elementum API instead of state
ei automations show [name-or-id] --from-remote

# TODO Export automation as Terraform HCL
ei automations show [name-or-id] --hcl

# TODO Path to Terraform state file (default: ./terraform.tfstate)
ei automations show [name-or-id] --state-file "<value>"
```

## status - TODO

Show recent execution history for an automation.

Usage patterns:
  ei automation status <automation-id> [execution-id]
  ei automation status <app-namespace> <automation-name> [execution-id]

With no execution ID, lists recent executions with a health summary.
With an execution ID, shows detailed action breakdown for that execution.

Shortcut flags (auto-select execution):
  --latest          Auto-select the most recent execution
  --latest-failure  Auto-select the most recent failed execution

Output modes for execution detail:
  (default)  Fast table view of actions (no I/O data)
  --io NAME  Fetch I/O for specific action by name or 1-based index
  --all-io   Fetch I/O for all actions (slower, sequential)
  --expand   Same as --all-io (legacy flag)
  --timeline CLI visual timeline with duration bars
  --html     Open interactive HTML waterfall in browser (lazy loads I/O)
  --json     Full structured JSON output

```bash
# By automation UUID
ei automation status 9063aed1-bf8c-430d-882f-8c502355a3c7

# By app namespace + automation name
ei automation status lumanow ValidateAndSubmit
ei automation status lumanow "Process Request" --status FAILURE

# Filter by status
ei automation status lumanow ValidateAndSubmit --status FAILURE

# Show last 24 hours
ei automation status lumanow ValidateAndSubmit --since 24h

# Auto-select latest execution and show details
ei automation status lumanow ValidateAndSubmit --latest

# Auto-select latest failure and show timeline
ei automation status lumanow ValidateAndSubmit --latest-failure --timeline

# One-liner debugging: latest failure + specific action I/O
ei automation status lumanow ValidateAndSubmit --latest-failure --io "Send Email"

# Show detail for a specific execution (fast, no I/O)
ei automation status lumanow ValidateAndSubmit exec-456-uuid

# Fetch I/O for specific action by name
ei automation status lumanow ValidateAndSubmit exec-456-uuid --io "Send Email"

# Fetch I/O for specific action by index (1-based)
ei automation status lumanow ValidateAndSubmit exec-456-uuid --io 3

# Fetch I/O for all actions (slower)
ei automation status lumanow ValidateAndSubmit exec-456-uuid --all-io

# Execution detail with timeline
ei automation status lumanow ValidateAndSubmit exec-456-uuid --timeline

# Open HTML waterfall in browser (lazy loads I/O on click)
ei automation status lumanow ValidateAndSubmit exec-456-uuid --html

# Watch for new executions (live stream)
ei automation status lumanow ValidateAndSubmit --watch

# JSON output
ei automation status lumanow ValidateAndSubmit --json
```

```bash
# TODO Fetch I/O for all actions (slower, sequential)
ei automations status (<automation-id> | <app-namespace> <automation-name>) [execution-id] --all-io

# TODO Show full inputs/outputs for each action
ei automations status (<automation-id> | <app-namespace> <automation-name>) [execution-id] --expand

# TODO Open interactive HTML waterfall in browser (lazy loads I/O)
ei automations status (<automation-id> | <app-namespace> <automation-name>) [execution-id] --html

# TODO Fetch I/O for specific action (name or 1-based index)
ei automations status (<automation-id> | <app-namespace> <automation-name>) [execution-id] --io "<value>"

# TODO Auto-select the most recent execution
ei automations status (<automation-id> | <app-namespace> <automation-name>) [execution-id] --latest

# TODO Auto-select the most recent failed execution
ei automations status (<automation-id> | <app-namespace> <automation-name>) [execution-id] --latest-failure

# TODO Number of executions to show (default 20)
ei automations status (<automation-id> | <app-namespace> <automation-name>) [execution-id] --limit <value>

# TODO Time range: 7d, 24h, 30m, or date like 2026-02-01 (default "7d")
ei automations status (<automation-id> | <app-namespace> <automation-name>) [execution-id] --since "<value>"

# TODO Filter by status: SUCCESS, FAILURE, RUNNING, QUEUED, CANCELLED
ei automations status (<automation-id> | <app-namespace> <automation-name>) [execution-id] --status "<value>"

# TODO Show CLI visual timeline with duration bars
ei automations status (<automation-id> | <app-namespace> <automation-name>) [execution-id] --timeline

# TODO Watch for new executions (append-only live stream)
ei automations status (<automation-id> | <app-namespace> <automation-name>) [execution-id] --watch
```
