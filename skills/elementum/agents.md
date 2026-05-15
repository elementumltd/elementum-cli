# Agents
Commands for listing, creating, showing, updating, and deleting Elementum agents.

## Available Commands

| Command | Purpose |
|--|--|
| create | Create a new agent |
| delete | Delete an agent |
| list | List agents in an app |
| show | Show agent details |
| update | Update an agent |

## create - Create Agent

Create a new Elementum AI agent in an app.

```bash
# Create an Elementum agent with a model name
ei agents create <namespace> --name "Support Agent" --description "Handles support tickets" --instructions "You are a helpful support agent." --model "gpt-4o"

# Create with explicit AI provider connector ID
ei agents create <namespace> --name "Support Agent" --description "Handles support tickets" --instructions "You are a helpful support agent." --ai-provider-connector-id "abc-123"

# Create with a first message
ei agents create <namespace> --name "Support Agent" --description "Handles support tickets" --instructions "You are a helpful support agent." --model "gpt-4o" --first-message "Hello! How can I help you today?"

# Create an agent in dry run mode without actually doing anything
ei agents create <namespace> --name "Test Agent" --description "Test" --instructions "Test" --model "gpt-4o" --dry-run

# Specify AI provider connector ID (alternative to --model)
ei agents create <namespace> --ai-provider-connector-id "<value>"

# Specify Agent type: elementum, snowflake, bedrock, browser_use (default "elementum")
ei agents create <namespace> --type "<value>"
```

## delete - Delete Agent
Delete an Elementum agent by name within an app.

WARNING: This permanently deletes the agent and all its tools.

```bash
# Delete Agent by name with confirmation
ei agents delete <namespace> "Support Agent"

# Delete Agent without confirmation
ei agents delete <namespace> "Support Agent" --force
```

## list - List Agents
List agents belonging to an app.

```bash
# List all Agents
ei agents list <namespace>
```

## show - Show Agent Details
Display detailed information about an agent within an app.

```bash
# Show details about Agent
ei agents show <namespace> "Support Agent"
```

## update - Update Agent

Update properties of an existing Elementum agent within an app.

```bash
# Update Agent name
ei agents update <namespace> "Support Agent" --name "New Support Agent"

# Update Agent description
ei agents update <namespace> "Support Agent" --description "Updated description"

# Update Agent instructions
ei agents update clm ticket-classifier --instructions "New instructions..."

# Update Agent model
ei agents update clm ticket-classifier --model "gpt-4o-mini"

# Update agent to different AI provider
ei agents update <namespace> <agent-name> --ai-provider-connector-id "<value>"

# Update agent to new first message
ei agents update <namespace> <agent-name> --first-message "<value>"
```