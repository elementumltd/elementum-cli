# Interventions

View automation interventions that require human attention.

Interventions are created when automations fail and need manual review.

## Available Commands

| Command | Purpose |
|--|--|
| list | List interventions for an aspect |
| show | Show intervention details |

## list - TODO

List automation interventions for an app, element, or task.

```bash
# List all interventions for an app
ei interventions list my-app

# Filter by status
ei interventions list my-app --status OPEN

# Filter by automation name (partial match)
ei interventions list my-app --automation "Process Orders"

# Limit results
ei interventions list my-app --limit 50

# JSON output
ei interventions list my-app --json
```

```bash
# TODO Filter by automation name (partial match)
ei interventions list <aspect-namespace-or-id> --automation "<value>"

# TODO Max interventions to fetch (filtering done client-side) (default 100)
ei interventions list <aspect-namespace-or-id> --limit <value>

# TODO Filter by status: OPEN, IN_PROGRESS, RESOLVED, IGNORED
ei interventions list <aspect-namespace-or-id> --status "<value>"
```

## show - TODO

Show detailed information about a specific intervention.

```bash
# Show intervention details
ei interventions show my-app abc-123-intervention-id

# JSON output
ei interventions show my-app abc-123-intervention-id --json
```
