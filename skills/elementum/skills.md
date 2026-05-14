# Skills

Commands for listing, creating, showing, updating, and deleting agentic skills.

## Available Commands

| Command | Purpose |
|--|--|
| create | Create an agentic skill |
| delete | Delete an agentic skill |
| list | List agentic skills in an app |
| show | Show skill details |
| update | Update an agentic skill |

## create - TODO

Create a new agentic skill in an app.

Skills define reusable capabilities for agents. They have instructions that guide
agent behavior and can have tools attached for executing actions.

```bash
# Create a skill on an app
ei skills create support-tickets --name "ticket-triage" \
  --description "Triages incoming support tickets" \
  --instructions "When a ticket comes in, classify it by urgency..."

# Create with a specific status
ei skills create support-tickets --name "draft-skill" \
  --description "Work in progress" \
  --instructions "..." \
  --status DRAFT
```

```bash
# TODO Skill description (required)
ei skills create <namespace> --description "<value>"

# TODO Show what would be created without creating
ei skills create <namespace> --dry-run

# TODO Skill instructions (required)
ei skills create <namespace> --instructions "<value>"

# TODO Skill name (required)
ei skills create <namespace> --name "<value>"

# TODO Skill status: ACTIVE, DRAFT, INACTIVE (default "ACTIVE")
ei skills create <namespace> --status "<value>"
```

## delete - TODO

Delete an agentic skill by name within an app. Also deletes all tools on the skill.

```bash
ei skills delete support-tickets "Ticket Triage"
ei skills delete clm escalation-handler
```

## list - TODO

List agentic skills belonging to an app.

```bash
ei skills list support-tickets
ei skills list clm --json
```

## show - TODO

Display detailed information about an agentic skill within an app.

```bash
ei skills show support-tickets "Ticket Triage"
ei skills show clm escalation-handler --json
```

## update - TODO

Update properties of an existing agentic skill within an app.

```bash
ei skills update support-tickets "Ticket Triage" --name "New Triage Skill"
ei skills update support-tickets "Ticket Triage" --description "Updated description"
ei skills update clm escalation-handler --instructions "New instructions..."
ei skills update clm escalation-handler --status INACTIVE
```

```bash
# TODO New skill description
ei skills update <namespace> <skill-name> --description "<value>"

# TODO New skill instructions
ei skills update <namespace> <skill-name> --instructions "<value>"

# TODO New skill name
ei skills update <namespace> <skill-name> --name "<value>"

# TODO New status: ACTIVE, DRAFT, INACTIVE
ei skills update <namespace> <skill-name> --status "<value>"
```
