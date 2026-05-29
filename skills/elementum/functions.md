# Functions

Commands for listing and managing stored Snowflake functions (procedures and UDFs).

## Available Commands

| Command | Purpose |
|--|--|
| delete | Delete a stored function |
| list | List all stored functions |

## delete - TODO

Delete a stored Snowflake function by ID or display name.

WARNING: This permanently deletes the stored function.

```bash
ei functions delete "Invoice Parser"
ei functions delete "Invoice Parser" --force
ei functions delete 794e1e48-73af-4760-... --force
```

```bash
# TODO Skip confirmation prompt
ei functions delete <id-or-name> --force
```

## list - TODO

Display a table of all stored Snowflake functions across all CloudLinks.
