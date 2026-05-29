# Update

## update - TODO

Check for and install updates to the ei CLI.

By default, this command will check for a newer version and prompt you to
confirm before installing. Use --check to only check without updating, or
--yes to skip the confirmation prompt.

```bash
ei update           # Check and update to latest (with confirmation)
ei update --check   # Only check, don't update
ei update --yes     # Update without confirmation
ei update --force   # Update even if already on latest
```

```bash
# TODO Only check for updates, don't install
ei update --check

# TODO Force update even if already on latest version
ei update --force

# TODO Skip confirmation prompt
ei update --yes
```
