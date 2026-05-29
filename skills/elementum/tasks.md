# Tasks

Commands for listing and exporting Elementum tasks (objects of type Task).

## Available Commands

| Command | Purpose |
|--|--|
| delete | Delete a task |
| export | Export a task to Terraform |
| list | List all tasks |

## delete - TODO

Delete an Elementum Task by namespace or ID.

WARNING: This permanently deletes the task and all its records.

```bash
ei tasks delete action-items
ei tasks delete action-items --force
ei tasks delete 9063aed1-bf8c-430d-882f-8c502355a3c7
ei tasks delete action-items --dry-run
```

```bash
# TODO Show what would be deleted without deleting
ei tasks delete <namespace-or-id> --dry-run

# TODO Skip confirmation prompt
ei tasks delete <namespace-or-id> --force
```

## export - TODO

Export an Elementum task as Terraform configuration.

You can specify the task by:
- Namespace: my-task
- ID: uuid

The command will generate an import block and run terraform to create the configuration.

```bash
# TODO Output file for generated Terraform configuration (default "generated.tf")
ei tasks export [namespace-or-id] --output "<value>"

# TODO Enable verbose debug output
ei tasks export [namespace-or-id] --verbose
```

## list - TODO

Display a table of all tasks (objects of type Task) in your organization.
