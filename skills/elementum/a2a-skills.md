# A2a-skills

Commands for listing and deleting A2A skills on agents.

A2A skills define what capabilities an agent exposes to other agents,
including supported input/output MIME types. This enables agents to
discover and invoke each other's capabilities.

## Available Commands

| Command | Purpose |
|--|--|
| delete | Delete an A2A skill |
| list | List A2A skills on an agent |

## delete - TODO

Delete an A2A skill by its ID.

```bash
ei a2a-skills delete abc123-skill-uuid
```

## list - TODO

List all A2A skills attached to an agent.

```bash
ei a2a-skills list "Support Bot"
ei a2a-skills list abc123-agent-uuid
ei a2a-skills list "Sales Assistant" --json
```
