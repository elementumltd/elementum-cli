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

## create - TODO

Create a new Elementum AI agent in an app.

```bash
# Create an Elementum agent with a model name
ei agents create support-tickets --name "Support Agent" \
  --description "Handles support tickets" \
  --instructions "You are a helpful support agent." \
  --model "gpt-4o"

# Create with explicit AI provider connector ID
ei agents create support-tickets --name "Support Agent" \
  --description "Handles support tickets" \
  --instructions "You are a helpful support agent." \
  --ai-provider-connector-id "abc-123"

# Create with a first message
ei agents create support-tickets --name "Support Agent" \
  --description "Handles support tickets" \
  --instructions "You are a helpful support agent." \
  --model "gpt-4o" \
  --first-message "Hello! How can I help you today?"

# Dry run
ei agents create support-tickets --name "Test Agent" \
  --description "Test" --instructions "Test" --model "gpt-4o" --dry-run
```

```bash
# TODO AI provider connector ID (alternative to --model)
ei agents create <namespace> --ai-provider-connector-id "<value>"

# TODO Agent description (required)
ei agents create <namespace> --description "<value>"

# TODO Show what would be created without creating
ei agents create <namespace> --dry-run

# TODO Agent's greeting message
ei agents create <namespace> --first-message "<value>"

# TODO Agent instructions/system prompt (required)
ei agents create <namespace> --instructions "<value>"

# TODO AI model name (e.g. gpt-4o, claude-3-5-sonnet)
ei agents create <namespace> --model "<value>"

# TODO Agent name (required)
ei agents create <namespace> --name "<value>"

# TODO Agent type: elementum, snowflake, bedrock, browser_use (default "elementum")
ei agents create <namespace> --type "<value>"
```

## delete - TODO

Delete an Elementum agent by name within an app.

WARNING: This permanently deletes the agent and all its tools.

```bash
ei agents delete support-tickets "Support Agent"
ei agents delete support-tickets "Support Agent" --force
ei agents delete clm ticket-classifier --force
```

```bash
# TODO Skip confirmation prompt
ei agents delete <namespace> <agent-name> --force
```

## list - TODO

List agents belonging to an app.

```bash
ei agents list support-tickets
ei agents list clm --json
```

## show - TODO

Display detailed information about an agent within an app.

```bash
ei agents show support-tickets "Support Agent"
ei agents show clm ticket-classifier --json
```

```bash
# TODO Export agent as Terraform HCL
ei agents show <namespace> <agent-name> --hcl
```

## update - TODO

Update properties of an existing Elementum agent within an app.

```bash
ei agents update support-tickets "Support Agent" --name "New Support Agent"
ei agents update support-tickets "Support Agent" --description "Updated description"
ei agents update clm ticket-classifier --instructions "New instructions..."
ei agents update clm ticket-classifier --model "gpt-4o-mini"
```

```bash
# TODO New AI provider connector ID
ei agents update <namespace> <agent-name> --ai-provider-connector-id "<value>"

# TODO New agent description
ei agents update <namespace> <agent-name> --description "<value>"

# TODO New first message
ei agents update <namespace> <agent-name> --first-message "<value>"

# TODO New agent instructions
ei agents update <namespace> <agent-name> --instructions "<value>"

# TODO New AI model name
ei agents update <namespace> <agent-name> --model "<value>"

# TODO New agent name
ei agents update <namespace> <agent-name> --name "<value>"
```
