# Skill-tools

Commands for listing, creating, updating, and deleting agentic skill tools.

## Available Commands

| Command | Purpose |
|--|--|
| create | Create a tool on an agentic skill |
| delete | Delete an agentic skill tool |
| delete-all | Delete all tools on an agentic skill |
| list | List tools on an agentic skill |
| update | Update an agentic skill tool |

## create - TODO

Create a new tool on an agentic skill within an app.

Tool types:
  automation       - Execute an automation (requires --automation-id)
  create-record    - Create a record in an app (requires --target-id)
  search-records   - Search records in an app (requires --target-id, --query-description)
  update-record    - Update a record in an app (requires --target-id, --handle-description)
  search-table     - Search a table (requires --search-table-id, --query-description)
  run-agent        - Run another agent (requires --target-agent-id, --worker-task-prompt)

```bash
# Create an automation tool
ei skill-tools create support-tickets "ticket-triage" --type automation \
  --name "Classify Ticket" --description "Runs classification" \
  --automation-id <automation-id>

# Create a search records tool
ei skill-tools create support-tickets "ticket-triage" --type search-records \
  --name "Find Tickets" --description "Search support tickets" \
  --target-id <app-id> --query-description "Search for tickets matching criteria"

# Create a create-record tool
ei skill-tools create support-tickets "ticket-triage" --type create-record \
  --name "Create Ticket" --description "Creates a new ticket" \
  --target-id <app-id>

# Create an update-record tool
ei skill-tools create support-tickets "ticket-triage" --type update-record \
  --name "Update Ticket" --description "Updates a ticket" \
  --target-id <app-id> --handle-description "Update the ticket record"

# Create a run-agent tool
ei skill-tools create support-tickets "ticket-triage" --type run-agent \
  --name "Delegate to Specialist" --description "Runs specialist agent" \
  --target-agent-id <agent-id> --worker-task-prompt "Handle this request"
```

```bash
# TODO Automation ID (for automation type)
ei skill-tools create <namespace> <skill-name> --automation-id "<value>"

# TODO Tool description
ei skill-tools create <namespace> <skill-name> --description "<value>"

# TODO Handle description (for update-record)
ei skill-tools create <namespace> <skill-name> --handle-description "<value>"

# TODO Tool name (required)
ei skill-tools create <namespace> <skill-name> --name "<value>"

# TODO Query description (for search-records, search-table)
ei skill-tools create <namespace> <skill-name> --query-description "<value>"

# TODO Max results returned (for search types)
ei skill-tools create <namespace> <skill-name> --result-limit <value>

# TODO Search table ID (for search-table type)
ei skill-tools create <namespace> <skill-name> --search-table-id "<value>"

# TODO Message shown when tool starts executing
ei skill-tools create <namespace> <skill-name> --start-message "<value>"

# TODO Target agent ID (for run-agent type)
ei skill-tools create <namespace> <skill-name> --target-agent-id "<value>"

# TODO Target app ID (for create-record, search-records, update-record, search-table)
ei skill-tools create <namespace> <skill-name> --target-id "<value>"

# TODO Tool type: automation, create-record, search-records, update-record, search-table, run-agent (required)
ei skill-tools create <namespace> <skill-name> --type "<value>"

# TODO Worker task prompt (for run-agent)
ei skill-tools create <namespace> <skill-name> --worker-task-prompt "<value>"
```

## delete - TODO

Delete an agentic skill tool

## delete-all - TODO

Delete all tools attached to an agentic skill within an app.

```bash
ei skill-tools delete-all support-tickets "Ticket Triage"
ei skill-tools delete-all clm escalation-handler
```

## list - TODO

List all tools attached to an agentic skill within an app.

```bash
ei skill-tools list support-tickets "Ticket Triage"
ei skill-tools list clm escalation-handler --json
```

## update - TODO

Update properties of an existing agentic skill tool.

The update uses the same type-specific input as create. Provide the original tool type
along with the fields you want to change.

```bash
ei skill-tools update <tool-id> --type automation --name "New Name" --description "New desc"
ei skill-tools update <tool-id> --type search-records --query-description "Updated query"
ei skill-tools update <tool-id> --type automation --start-message "Processing..."
```

```bash
# TODO New tool description
ei skill-tools update <tool-id> --description "<value>"

# TODO New handle description (for update-record)
ei skill-tools update <tool-id> --handle-description "<value>"

# TODO New tool name
ei skill-tools update <tool-id> --name "<value>"

# TODO New query description (for search types)
ei skill-tools update <tool-id> --query-description "<value>"

# TODO New start message
ei skill-tools update <tool-id> --start-message "<value>"

# TODO Tool type (required to determine update shape)
ei skill-tools update <tool-id> --type "<value>"

# TODO New worker task prompt (for run-agent)
ei skill-tools update <tool-id> --worker-task-prompt "<value>"
```
