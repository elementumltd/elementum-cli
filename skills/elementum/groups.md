# Groups

Commands for listing and exporting Elementum groups.

## Available Commands

| Command | Purpose |
|--|--|
| export | Export a group to Terraform |
| list | List all groups |

## export - TODO

Export an Elementum group as Terraform configuration.

You can specify the group by:
- Name: "Engineering Team"
- ID: uuid

The command will generate an import block and run terraform to create the configuration.

```bash
# TODO Output file for generated Terraform configuration (default "generated.tf")
ei groups export [name-or-id] --output "<value>"

# TODO Enable verbose debug output
ei groups export [name-or-id] --verbose
```

## list - TODO

Display a table of all groups in your organization.
