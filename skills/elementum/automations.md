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

## config - Show Automation Configuration Hierarchy
Display the full configuration of an automation as a visual flow.

Shows the trigger and all tasks in execution order, with proper
rendering of switch branches and for_each loops.

Output modes:
  (default)     CLI visual flow with connectors and nesting
  --json        Full structured JSON of the automation config
  --timeline    Alias for default CLI visual flow
  --html        Interactive HTML visualization opened in browser

```bash
# Show visual flow with connectors and nesting in default timeline view
ei automation config abc-123-uuid

# Show visual flow with connectors and nesting in default timeline view
ei automation config abc-123-uuid --timeline

# Show flow in json format
ei automation config abc-123-uuid --json

# Show visual flow with connectors and nesting in HTML file in tmp folder
ei automation config abc-123-uuid --html
```

## delete - Delete An Automation
Delete an automation by ID or by app namespace and automation name.

Usage patterns:
  ei automation delete <automation-id>
  ei automation delete <app-namespace> <automation-name>

```bash
# Delete by Automation ID
ei automation delete 9063aed1-bf8c-430d-882f-8c502355a3c7

# Delete by app namespace + automation name
ei automation delete <namespace> "Process Request"

# Delete Skip confirmation prompt
ei automation delete <namespace> "Process Request" --force

# Preview what would be deleted in dry-run mode without actually deleting
ei automation delete <namespace> "Process Request" --dry-run
```

## list - List Available Automations
Display automations configured for the specified app or element.

```bash
# Show list of automations in a table
ei automations list <app-namespace> 

# Show full task configuration for each automation with format similar to config verb
ei automations list <app-namespace> --details
```

## run - Run an Automation
Triggers an automation with webhook or on-demand trigger.

This command supports two execution modes:

1. Webhook mode (default): Triggers via the automation's webhook URL
2. Widget mode: Triggers via a run automation widget on a specific record

```bash
# Run via webhook and wait for completion (default)
ei automation run abc-123-uuid

# Run and don't block for completion
ei automation run abc-123-uuid --no-wait

# Run via widget on a specific record specified by ID
ei automation run abc-123-uuid --widget def-456 --record "app-id:REC-001"
```

## show - Show Automation Detail
Display details about an automation workflow.

Use --from-remote to query the Elementum API directly.

Shows the automation structure including:
- Trigger type and configuration
- Tasks with their refs
- Workflow status


```bash
# Query automation from Elementum API instead of state
ei automations show [name-or-id] --from-remote
```

## status - Get Automation Execution Historyß
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
# Get automation executions by automation UUID
ei automation status 9063aed1-bf8c-430d-882f-8c502355a3c7

# Get automation executions by app namespace + automation name
ei automation status <app-namespace> ValidateAndSubmit

# Get automation executions by app namespace + automation name and with status (SUCCESS, FAILURE, RUNNING, QUEUED, CANCELLED)
ei automation status <app-namespace> ValidateAndSubmit --status FAILURE

# Show last 24 hours
ei automation status <app-namespace> ValidateAndSubmit --since 24h

# Auto-select latest execution and show details
ei automation status <app-namespace> ValidateAndSubmit --latest

# Auto-select latest failure and show timeline
ei automation status <app-namespace> ValidateAndSubmit --latest-failure --timeline

# One-liner debugging: latest failure + specific action I/O
ei automation status <app-namespace> ValidateAndSubmit --latest-failure --io "Send Email"

# Show detail for a specific execution (fast, no I/O)
ei automation status <app-namespace> ValidateAndSubmit exec-456-uuid

# Fetch I/O for specific action by name
ei automation status <app-namespace> ValidateAndSubmit exec-456-uuid --io "Send Email"

# Fetch I/O for specific action by index (1-based)
ei automation status <app-namespace> ValidateAndSubmit exec-456-uuid --io 3

# Fetch I/O for all actions (slower)
ei automation status <app-namespace> ValidateAndSubmit exec-456-uuid --all-io

# Execution detail with timeline
ei automation status <app-namespace> ValidateAndSubmit exec-456-uuid --timeline

# Open HTML waterfall in browser (lazy loads I/O on click)
ei automation status <app-namespace> ValidateAndSubmit exec-456-uuid --html

# Watch for new executions (live stream)
ei automation status <app-namespace> ValidateAndSubmit --watch

# Show full inputs/outputs for each action
ei automations status (<automation-id> | <app-namespace> <automation-name>) [execution-id] --expand

# Limit number of executions to show (default 20)
ei automations status (<automation-id> | <app-namespace> <automation-name>) [execution-id] --limit <value>

# Watch for new executions (append-only live stream)
ei automations status (<automation-id> | <app-namespace> <automation-name>) [execution-id] --watch
```