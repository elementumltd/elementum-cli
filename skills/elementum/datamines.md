# Datamines

Commands for listing and exporting Elementum datamines.

## Available Commands

| Command | Purpose |
|--|--|
| export | Export a datamine to Terraform |
| list | List datamines |

## export - TODO

Export an Elementum Datamine as Terraform configuration.

You can specify the datamine by:
- Table ID/name and Datamine name: ei datamines export "Sales Summary" "Stale Records Monitor"
- Table ID and Datamine ID: ei datamines export tbl_12345 dm_67890

The command will generate an import block and run terraform to create the configuration.

```bash
# TODO Resolve UUIDs to references and strip null attributes (default true)
ei datamines export [table-id-or-name] [datamine-name-or-id] --beautify

# TODO Output file for generated Terraform configuration (default "generated.tf")
ei datamines export [table-id-or-name] [datamine-name-or-id] --output "<value>"

# TODO Enable verbose debug output
ei datamines export [table-id-or-name] [datamine-name-or-id] --verbose
```

## list - TODO

Display datamines in your organization.

If a table name or ID is provided, list datamines for that specific table.
Otherwise, list all datamines across all tables.
