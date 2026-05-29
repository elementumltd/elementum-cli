# Layouts

Commands for managing layouts (stages) on Elementum apps.

In Elementum, a "layout" is a stage on an app. Each stage defines what sections
(field groups, activity log, attachments, etc.) appear on the record detail page.

## Available Commands

| Command | Purpose |
|--|--|
| create | Create a new layout (stage) on an app |
| delete | Delete a layout (stage) from an app |
| list | List layouts (stages) for an app |
| show | Show layout details for a stage |
| update | Update a layout (stage) on an app |

## create - TODO

Create a new layout stage on an Elementum App.

Stages define the different phases a record goes through (e.g., Open, In Progress, Done).
Each stage can have its own layout with different display block sections.

```bash
ei layouts create support-tickets --name "In Progress"
ei layouts create support-tickets --name "Review" --color "#FF5733" --icon "eye"
ei layouts create support-tickets --name "Done" --dry-run
```

```bash
# TODO Stage color (hex, e.g., #FF5733)
ei layouts create <namespace> --color "<value>"

# TODO Show what would be created without creating
ei layouts create <namespace> --dry-run

# TODO Stage icon name
ei layouts create <namespace> --icon "<value>"

# TODO Stage name (required)
ei layouts create <namespace> --name "<value>"
```

## delete - TODO

Delete a stage from an Elementum App.

WARNING: This permanently removes the stage and its layout configuration.

```bash
ei layout delete support-tickets "In Progress"
ei layout delete support-tickets Review --force
ei layout delete support-tickets Done --dry-run
```

```bash
# TODO Show what would be deleted without deleting
ei layouts delete <namespace> <stage-name> --dry-run

# TODO Skip confirmation prompt
ei layouts delete <namespace> <stage-name> --force
```

## list - TODO

List all layouts/stages configured on an Elementum App.

Each stage has a name and contains display block sections (field groups, activity log, etc.).

```bash
ei layouts list support-tickets
ei layouts list support-tickets --json
```

## show - TODO

Show detailed layout information for a specific stage, including all display block sections.

```bash
ei layout show support-tickets "Open"
ei layout show support-tickets Initiate --json
```

## update - TODO

Update properties of an existing stage on an Elementum App.

```bash
ei layout update support-tickets "Open" --name "New"
ei layout update support-tickets "In Progress" --color "#FF5733"
ei layout update support-tickets "Done" --icon "check"
```

```bash
# TODO New stage color (hex)
ei layouts update <namespace> <stage-name> --color "<value>"

# TODO New stage icon
ei layouts update <namespace> <stage-name> --icon "<value>"

# TODO New stage name
ei layouts update <namespace> <stage-name> --name "<value>"

# TODO New display order (default -1)
ei layouts update <namespace> <stage-name> --order <value>
```
