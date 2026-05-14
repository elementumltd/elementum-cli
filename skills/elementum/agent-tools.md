# Agent-tools

Commands for listing, creating, updating, and deleting agent tools.

## Available Commands

| Command | Purpose |
|--|--|
| create | Create a tool on an agent |
| delete | Delete an agent tool |
| list | List tools on an agent |
| update | Update an agent tool |

## create - TODO

Create a new tool on an agent.

Tool types:
  run-automation   - Execute an automation (requires --automation-id)
  create-record    - Create a record in an app (requires --target-id)
  search-records   - Search records in an app (requires --target-id, --query-description)
  update-record    - Update a record (requires --target-id, --handle-description)
  relate-record    - Relate two records (requires --target-id, --related-target-id)
  search-table     - Search a table (requires --search-table-id, --query-description)
  run-agent        - Run another agent (requires --target-agent-id, --worker-task-prompt)
  mcp              - MCP tool (requires --mcp-tool-name, --server-url)

```bash
# Create an automation tool
ei agent-tools create "Support Agent" --type run-automation \
  --name "Classify Ticket" --description "Runs classification" \
  --automation-id <automation-id>

# Create a search records tool
ei agent-tools create "Support Agent" --type search-records \
  --name "Find Tickets" --description "Search support tickets" \
  --target-id <app-id> --query-description "Search for tickets matching criteria"

# Create a create-record tool
ei agent-tools create "Support Agent" --type create-record \
  --name "Create Ticket" --description "Creates a new ticket" \
  --target-id <app-id>

# Create an update-record tool
ei agent-tools create "Support Agent" --type update-record \
  --name "Update Ticket" --description "Updates ticket fields" \
  --target-id <app-id> --handle-description "Update the ticket"

# Create a relate-record tool
ei agent-tools create "Support Agent" --type relate-record \
  --name "Link KB Article" --description "Links a KB article to ticket" \
  --target-id <app-id> --related-target-id <element-id>

# Create an MCP tool
ei agent-tools create "Support Agent" --type mcp \
  --name "External API" --description "Calls external service" \
  --mcp-tool-name "api_call" --server-url "https://example.com/mcp"

# Create a run-agent tool
ei agent-tools create "Support Agent" --type run-agent \
  --name "Delegate" --description "Delegates to specialist" \
  --target-agent-id <agent-id> --worker-task-prompt "Handle this request"
```

```bash
# TODO Automation ID (for run-automation)
ei agent-tools create <agent-name-or-id> --automation-id "<value>"

# TODO Tool description
ei agent-tools create <agent-name-or-id> --description "<value>"

# TODO Handle description (for update-record)
ei agent-tools create <agent-name-or-id> --handle-description "<value>"

# TODO MCP tool name (for mcp)
ei agent-tools create <agent-name-or-id> --mcp-tool-name "<value>"

# TODO Tool name (required)
ei agent-tools create <agent-name-or-id> --name "<value>"

# TODO Query description (for search-records, search-table)
ei agent-tools create <agent-name-or-id> --query-description "<value>"

# TODO Related app/element ID (for relate-record)
ei agent-tools create <agent-name-or-id> --related-target-id "<value>"

# TODO Max results returned (for search types)
ei agent-tools create <agent-name-or-id> --result-limit <value>

# TODO Search table ID (for search-table)
ei agent-tools create <agent-name-or-id> --search-table-id "<value>"

# TODO Server URL (for mcp)
ei agent-tools create <agent-name-or-id> --server-url "<value>"

# TODO Message shown when tool starts
ei agent-tools create <agent-name-or-id> --start-message "<value>"

# TODO Target agent ID (for run-agent)
ei agent-tools create <agent-name-or-id> --target-agent-id "<value>"

# TODO Target app ID (for create-record, search-records, update-record, relate-record)
ei agent-tools create <agent-name-or-id> --target-id "<value>"

# TODO Tool type: run-automation, create-record, search-records, update-record, relate-record, search-table, run-agent, mcp (required)
ei agent-tools create <agent-name-or-id> --type "<value>"

# TODO Worker task prompt (for run-agent)
ei agent-tools create <agent-name-or-id> --worker-task-prompt "<value>"
```

## delete - TODO

Delete an agent tool

## list - TODO

List tools on an agent

## update - TODO

Update properties of an existing agent tool.

Provide the original tool type along with the fields you want to change.

```bash
ei agent-tools update <tool-id> --type run-automation --name "New Name"
ei agent-tools update <tool-id> --type search-records --query-description "Updated query"
ei agent-tools update <tool-id> --type run-automation --description "Updated desc"
```

```bash
# TODO New tool description
ei agent-tools update <tool-id> --description "<value>"

# TODO New handle description (update-record)
ei agent-tools update <tool-id> --handle-description "<value>"

# TODO New tool name
ei agent-tools update <tool-id> --name "<value>"

# TODO New query description (search types)
ei agent-tools update <tool-id> --query-description "<value>"

# TODO New start message
ei agent-tools update <tool-id> --start-message "<value>"

# TODO Tool type (required to determine update shape)
ei agent-tools update <tool-id> --type "<value>"

# TODO New worker task prompt (run-agent)
ei agent-tools update <tool-id> --worker-task-prompt "<value>"
```
