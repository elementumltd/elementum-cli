# Deployment Reference

## Lifecycle

```
create → PENDING → SCOPING → EXTRACTING → DIFFING → COMPLETED
                                                   → CONFIGURATION (needs dataset mapping)
                                                   → FAILED
```

## Full Deployment

```bash
# 1. Discover environments
ei environments list

# 2. Create deployment (polls until terminal state)
ei deployments create <app-ns> --source Staging --target Production

# 3. If CONFIGURATION state, inspect and configure
ei deployments show <deployment-id> --json | jq '.config_template' > config.json
# Edit config.json: update cloud_link_id and snowflake_mapping for target env
ei deployments configure <deployment-id> --config config.json --dry-run
ei deployments configure <deployment-id> --config config.json

# 4. Verify
ei deployments show <deployment-id>
```

## Configuration File

```json
{
  "datasets": [{
    "id": "element-uuid",
    "cloud_link_id": "target-cloudlink-uuid",
    "snowflake_mapping": {
      "database_name": "TARGET_DB",
      "schema_name": "TARGET_SCHEMA",
      "table_name": "TARGET_TABLE"
    },
    "fields": [{ "id": "field-uuid", "name": "Title", "type": "TEXT", "column_name": "TITLE" }]
  }]
}
```

Change `cloud_link_id` and `snowflake_mapping` for target environment. Keep field mappings.

Find target CloudLink: `ei cloudlinks list`

## Commands

```bash
ei environments list
ei deployments list [--app <ns>]
ei deployments create <app-ns> --source <env> --target <env> [--no-wait] [--json]
ei deployments show <id> [--json]
ei deployments configure <id> --config file.json [--dry-run]
```
