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

## create - Create Agent Skill
Create a new agentic skill in an app.

Skills define reusable capabilities for agents. They have instructions that guide
agent behavior and can have tools attached for executing actions.

```bash
# Create a skill on an app with description and instructions
ei skills create <namespace> --name "ticket-triage" --description "Triages incoming support tickets" --instructions "When a ticket comes in, classify it by urgency..."

# Create with a specific status
ei skills create <namespace> --name "draft-skill" --description "Work in progress" --instructions "..." --status DRAFT

# Create skill in dry run mode without actually creating
ei skills create <namespace> --dry-run
```

## delete - Delete Agent Skill
Delete an agentic skill by name within an app. Also deletes all tools on the skill.

```bash
# Delete agent skill
ei skills delete <namespace> "Ticket Triage"
```

## list - List Agent Skills
List agentic skills belonging to an app.

```bash
# List all skills
ei skills list <namespace>
```

## show - Show Agent Skill details 
Display detailed information about an agentic skill within an app.

```bash
# Show details about agent skill
ei skills show <namespace> "Ticket Triage"
```

## update - Update Agent Skill
Update properties of an existing agentic skill within an app.

```bash
# Update skill name
ei skills update <namespace> "Ticket Triage" --name "New Triage Skill"

# Update skill description
ei skills update <namespace> "Ticket Triage" --description "Updated description"

# Update skill instructions
ei skills update clm escalation-handler --instructions "New instructions..."

# Set skill status (options ACTIVE, DRAFT, INACTIVE)
ei skills update clm escalation-handler --status INACTIVE
```