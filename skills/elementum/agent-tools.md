# Agent-tools
Commands for listing, creating, updating, and deleting agent tools.

## Available Commands

| Command | Purpose |
|--|--|
| create | Create a tool on an agent |
| delete | Delete an agent tool |
| list | List tools on an agent |
| update | Update an agent tool |

## create - Create New Agent Tool
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
ei agent-tools create "Support Agent" --type run-automation  --name "Classify Ticket" --description "Runs classification" --automation-id <automation-id>

# Create a search records tool
ei agent-tools create "Support Agent" --type search-records --name "Find Tickets" --description "Search support tickets" --target-id <app-id> --query-description "Search for tickets matching criteria"

# Create a create-record tool
ei agent-tools create "Support Agent" --type create-record --name "Create Ticket" --description "Creates a new ticket" --target-id <app-id>

# Create an update-record tool
ei agent-tools create "Support Agent" --type update-record --name "Update Ticket" --description "Updates ticket fields" --target-id <app-id> --handle-description "Update the ticket"

# Create a relate-record tool
ei agent-tools create "Support Agent" --type relate-record --name "Link KB Article" --description "Links a KB article to ticket" --target-id <app-id> --related-target-id <element-id>

# Create an MCP tool
ei agent-tools create "Support Agent" --type mcp --name "External API" --description "Calls external service" --mcp-tool-name "api_call" --server-url "https://example.com/mcp"

# Create a run-agent tool
ei agent-tools create "Support Agent" --type run-agent --name "Delegate" --description "Delegates to specialist" --target-agent-id <agent-id> --worker-task-prompt "Handle this request"
```

## delete - Delete Agent tool
Delete an agent tool

```bash
# Delete agent tool from agent
ei agent-tools delete <tool-id>
```

## list - TODO
List tools on an agent

```bash
# List all tools on an agent
ei agent-tools list <agent-name-or-id>
```

## update - TODO
Update properties of an existing agent tool.

Provide the original tool type along with the fields you want to change.

```bash
# Update agent tool name
ei agent-tools update <tool-id> --type run-automation --name "New Name"

# Update search record agent tool query description
ei agent-tools update <tool-id> --type search-records --query-description "Updated query"

# Update run-automation agent tool description
ei agent-tools update <tool-id> --type run-automation --description "Updated desc"

# Update tool with new start message
ei agent-tools update <tool-id> --start-message "<value>"

# Update tool with new handle description (for type update-record)
ei agent-tools update <tool-id> --handle-description "<value>"

# Update new  worker task prompt (for type run-agent)
ei agent-tools update <tool-id> --worker-task-prompt "<value>"
```