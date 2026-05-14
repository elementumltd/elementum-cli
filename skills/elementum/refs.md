# Refs

## refs - TODO

Display value references (refs) from Elementum resources.

By default, reads from terraform.tfstate in the current directory.
Use --from-remote to query refs directly from the Elementum API.

This command helps you discover what refs are available for use in
automation tasks, filters, and other value reference fields.

```bash
# From state (default)
ei refs
ei refs --state-file ./terraform.tfstate
ei refs --resource elementum_record_created_trigger.my_trigger

# From remote API (requires authentication)
ei refs --from-remote <automation-id>
ei refs --from-remote <automation-id> --after <task-id>
ei refs --from-remote <automation-id> --json
```

```bash
# TODO Show refs available after a specific task (for --from-remote)
ei refs [automation-id] --after "<value>"

# TODO Query refs from remote server (requires authentication)
ei refs [automation-id] --from-remote

# TODO Filter to specific resource by name or address pattern
ei refs [automation-id] --resource "<value>"

# TODO Path to terraform.tfstate file (default: ./terraform.tfstate)
ei refs [automation-id] --state-file "<value>"
```
