# Fields

Commands for working with Elementum fields.

## Available Commands

| Command | Purpose |
|--|--|
| create | Create a field on an app or element |
| delete | Delete a field from an app or element |
| list | List fields on an app or element |
| update | Update a field on an app or element |
| values | List values for a dropdown/picklist field |

## create - TODO

Create a new field on an Elementum App or Element.

Supported field types: text, number, boolean, date, datetime, dropdown,
multiselect, user, group, longtext, attachment, json, decimal.

For dropdown and multiselect fields, provide options with --options.

```bash
# Create a text field
ei fields create support-tickets --name "Customer Email" --type text

# Create a required text field
ei fields create support-tickets --name "Subject" --type text --required

# Create a dropdown with options
ei fields create support-tickets --name "Priority" --type dropdown --options "Low,Medium,High,Critical"

# Create a multiselect
ei fields create support-tickets --name "Tags" --type multiselect --options "Bug,Feature,Question"

# Create a number field
ei fields create support-tickets --name "Score" --type number

# Create a boolean field
ei fields create support-tickets --name "Escalated" --type boolean

# Create a user field
ei fields create support-tickets --name "Assigned To" --type user

# Create a date field with description
ei fields create support-tickets --name "Due Date" --type date --description "Expected resolution date"

# Dry run
ei fields create support-tickets --name "Test" --type text --dry-run
```

```bash
# TODO Field description
ei fields create <namespace> --description "<value>"

# TODO Show what would be created without creating
ei fields create <namespace> --dry-run

# TODO Field name (required)
ei fields create <namespace> --name "<value>"

# TODO strings      Dropdown/multiselect options (comma-separated)
ei fields create <namespace> --options

# TODO Make field required
ei fields create <namespace> --required

# TODO Show field on record creation form
ei fields create <namespace> --show-on-create

# TODO Field type: text, number, boolean, date, datetime, dropdown, multiselect, user, group, longtext, attachment, json, decimal (required)
ei fields create <namespace> --type "<value>"
```

## delete - TODO

Delete a field from an Elementum App or Element by name.

WARNING: This permanently removes the field and all its data from all records.

```bash
ei field delete support-tickets "Customer Email"
ei field delete support-tickets Priority --force
ei field delete support-tickets Score --dry-run
```

```bash
# TODO Show what would be deleted without deleting
ei fields delete <namespace> <field-name> --dry-run

# TODO Skip confirmation prompt
ei fields delete <namespace> <field-name> --force
```

## list - TODO

List all fields on an Elementum App or Element.

```bash
ei fields list support-tickets
ei fields list support-tickets --json
ei fields list support-tickets --all   # Include system fields
```

```bash
# TODO Include system fields
ei fields list <namespace> --all
```

## update - TODO

Update properties of an existing field on an Elementum App or Element.

You can rename the field, update its description, or toggle required/show-on-create.

```bash
ei field update support-tickets "Customer Email" --name "Contact Email"
ei field update support-tickets Priority --description "Ticket priority level"
ei field update support-tickets Priority --required
ei field update support-tickets Score --no-required
```

```bash
# TODO New field description
ei fields update <namespace> <field-name> --description "<value>"

# TODO New field name
ei fields update <namespace> <field-name> --name "<value>"

# TODO Make field not required
ei fields update <namespace> <field-name> --no-required

# TODO Hide field from record creation form
ei fields update <namespace> <field-name> --no-show-on-create

# TODO Make field required
ei fields update <namespace> <field-name> --required

# TODO Show field on record creation form
ei fields update <namespace> <field-name> --show-on-create
```

## values - TODO

Fetch and display available values for a dropdown or picklist field.

For static picklists, displays all available options with their IDs and labels.
For dynamic picklists, shows the related aspect and suggests searching for values.

```bash
ei fields values support-tickets Status
ei fields values support-tickets Priority
ei fields values luma Stage --json
```
