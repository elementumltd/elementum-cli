# Debugging Automation Executions

The 4-step drill-down workflow for inspecting automation failures.

## Usage

```bash
# By app namespace + automation name (recommended)
ei automation status <app-namespace> <automation-name> [execution-id]

# By automation UUID
ei automation status <automation-id> [execution-id]
```

## Step 1: List Recent Executions

```bash
ei automation status myapp "Process Request"
ei automation status myapp "Process Request" --status FAILURE
ei automation status myapp "Process Request" --since 24h
ei automation status myapp "Process Request" --watch
```

## Step 2: Get an Execution ID

```bash
# JSON output for scripting
ei automation status myapp "Process Request" --json | jq '.executions[0].node.id'

# Shortcut flags:
ei automation status myapp "Process Request" --latest --timeline
ei automation status myapp "Process Request" --latest-failure --io "Send API Request"
```

## Step 3: View Execution Details

```bash
ei automation status myapp "Process Request" <execution-id>             # Table
ei automation status myapp "Process Request" <execution-id> --timeline  # Visual
ei automation status myapp "Process Request" <execution-id> --json      # Machine
```

## Step 4: Get Action I/O

```bash
# By name (fast, targeted)
ei automation status myapp "Process Request" <exec-id> --io "Send API Request"

# By index
ei automation status myapp "Process Request" <exec-id> --io 3

# One-liner for latest failure
ei automation status myapp "Process Request" --latest-failure --io "Send API Request"
```

## I/O Flag Comparison

| Flag | Use Case |
|------|----------|
| `--io "Name"` | One action's I/O by name (fast) |
| `--io 3` | By index when name is ambiguous |
| `--all-io` | All actions' I/O (slower, full picture) |
| `--html` | Interactive browser with expandable I/O |

## Output Modes

| Flag | Best For |
|------|----------|
| (default) | Quick table of action status and duration |
| `--timeline` | CLI visualization with bars, colors, inline errors |
| `--html` | Interactive browser, expandable I/O, error highlighting |
| `--json` | Programmatic analysis, piping to `jq` |

## Common Failure Patterns

| Symptom | How to Check | Common Cause |
|---------|--------------|--------------|
| First task fails | `--timeline` red on task 1 | Bad input, missing field |
| API task fails | `--io "API Task"` | Wrong URL, auth failure, bad payload |
| Search returns empty | `--io "Search Task"` | Filter too restrictive, wrong field ID |
| Timeout | `--timeline` duration bars | External API slow |
| Partial completion | `--timeline` shows stop point | Wrong conditional path |

## Example Session

```bash
# 1. Find failures
ei automation status myapp "Process Request" --status FAILURE

# 2. View latest failure
ei automation status myapp "Process Request" --latest-failure

# 3. Get I/O for failed action
ei automation status myapp "Process Request" --latest-failure --io "Send API Request"
# Shows: Inputs: {"url": "...", "body": {...}}
#        Outputs: {"status": 401, "error": "Unauthorized"}

# 4. Or open interactive HTML
ei automation status myapp "Process Request" --latest-failure --html
```

## Debugging Skill Tool Failures

When an agent's skill tool fails (runs an automation behind the scenes):

```bash
# Find the automation from skill tool config, then:
ei automation status servicerequests "ValidateRequest" --latest-failure --timeline
ei automation status servicerequests "ValidateRequest" --latest-failure --io "Lookup Target"
```

### Adding Debug Info to Test Reports

```yaml
tool_failure:
  tool_name: "ValidateSharedFolderRequest"
  automation_id: "5ec08f66-..."
  execution_id: "exec-456"
  failed_task: "Lookup Target Group"
  error: "No records found matching filter"
  root_cause: "Group name had trailing space"
```
