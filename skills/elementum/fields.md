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

## create - Create a Field
Create a new field on an Elementum App or Element.

Supported field types: text, number, boolean, date, datetime, dropdown,
multiselect, user, group, longtext, attachment, json, decimal.

For dropdown and multiselect fields, provide options with --options.

```bash
# Create a text field with some description. Type can be (text, number, boolean, date, datetime, dropdown, multiselect, user, group, longtext, attachment, json, decimal (required))
ei fields create <namespace> --name "Customer Email" --type text --description "<value>"

# Create a required text field
ei fields create <namespace> --name "Subject" --type text --required

# Create a dropdown with options
ei fields create <namespace> --name "Priority" --type dropdown --options "Low,Medium,High,Critical"

# Create a multiselect
ei fields create <namespace> --name "Tags" --type multiselect --options "Bug,Feature,Question"

# Create a number field
ei fields create <namespace> --name "Score" --type number

# Create a boolean field
ei fields create <namespace> --name "Escalated" --type boolean

# Create a user field
ei fields create <namespace> --name "Assigned To" --type user

# Create a date field with description
ei fields create <namespace> --name "Due Date" --type date --description "Expected resolution date"

# Create a field in dry run mode without actually creating
ei fields create <namespace> --name "Test" --type text --dry-run

# TODO Show field on record creation form
ei fields create <namespace> --show-on-create

```

## delete - Delete a Field
Delete a field from an Elementum App or Element by name.

WARNING: This permanently removes the field and all its data from all records.

```bash
# Delete field with confirmation
ei field delete <namespace> "Customer Email"

# Delete field without confirmation
ei field delete <namespace> Priority --force

# Delete field in dry run mode without actually doing it
ei field delete <namespace> Score --dry-run
```

## list - List Fields
List all fields on an Elementum App or Element.

```bash
# List fields in app
ei fields list <namespace>

# List all fields including system ones
ei fields list <namespace> --all
```

## update - Update a Field
Update properties of an existing field on an Elementum App or Element.

You can rename the field, update its description, or toggle required/show-on-create.

```bash
# Update field name
ei field update <namespace> "Customer Email" --name "Contact Email"

# Update field description
ei field update <namespace> Priority --description "Ticket priority level"

# Set field to be required
ei field update <namespace> Priority --required

# Set field to not be required
ei field update <namespace> Score --no-required

# Set field to show on create
ei fields update <namespace> <field-name> --show-on-create

# Set field to not show on create
ei field update <namespace> Score --no-show-on-create
```

## values - Show Field Values in Dropdowns
Fetch and display available values for a dropdown or picklist field.

For static picklists, displays all available options with their IDs and labels.
For dynamic picklists, shows the related aspect and suggests searching for values.

```bash
# Show field values
ei fields values <namespace> Status
```