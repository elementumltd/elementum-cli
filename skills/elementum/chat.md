# Chat

## chat - TODO

Start an interactive conversation with an Elementum agent.

The agent can be specified by name (case-insensitive) or UUID.

```bash
ei agents --app clm                                # List agents in the CLM app
ei chat "IT Support Agent"                         # Chat with agent by name
ei chat 794e1e48-73af-4760-8a1f-...               # Chat with agent by ID
ei chat "IT Support Agent" -m "Hello"              # Send single message and exit
ei chat "IT Support Agent" --list                  # List recent conversations
ei chat "IT Support Agent" --continue <conv-id>    # Continue existing conversation

Multi-turn example:
ei chat "IT Support Agent" -m "Hello"                              # Returns conversation ID
ei chat "IT Support Agent" -c <conv-id> -m "Tell me more"         # Continue conversation
```

```bash
# TODO Continue an existing conversation
ei chat <agent-name-or-id> --continue "<value>"

# TODO List recent conversations for this agent
ei chat <agent-name-or-id> --list

# TODO Send a single message (non-interactive)
ei chat <agent-name-or-id> --message "<value>"
```
