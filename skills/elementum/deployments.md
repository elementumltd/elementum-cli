# Deployments

Commands for listing, creating, showing, and configuring deployments across environments.

## Available Commands

| Command | Purpose |
|--|--|
| configure | Configure missing datasets for a deployment |
| create | Create a deployment |
| list | List deployments |
| show | Show deployment details |

## configure - TODO

Configure missing dataset mappings for a deployment that is in CONFIGURATION state.

Reads a JSON config file that specifies cloud link, Snowflake mapping, and field
column mappings for each dataset that needs configuration.

Example config file:
  {
    "datasets": [
      {
        "id": "element-uuid",
        "cloud_link_id": "cloudlink-uuid",
        "snowflake_mapping": {
          "database_name": "MY_DB",
          "schema_name": "MY_SCHEMA",
          "table_name": "MY_TABLE"
        },
        "fields": [
          { "id": "field-uuid", "name": "Title", "type": "TEXT", "column_name": "TITLE", "semantic_tags": ["TITLE"] },
          { "id": "field-uuid", "name": "ID", "type": "TEXT", "column_name": "ID", "semantic_tags": ["HANDLE"] }
        ]
      }
    ]
  }

```bash
ei deployments configure abc-123 --config config.json
ei deployments configure abc-123 --config config.json --dry-run
ei deployments configure abc-123 --config config.json --json
```

```bash
# TODO Path to JSON config file (required)
ei deployments configure <deployment-id> --config "<value>"

# TODO Show what would be configured without applying
ei deployments configure <deployment-id> --dry-run
```

## create - TODO

Create a new deployment to promote an app from one environment to another.

By default, waits for the deployment to complete (or reach configuration state).
Use --no-wait to return immediately after creation.

```bash
ei deployments create my-app --source staging --target production
ei deployments create my-app --source staging --target production --name "v2.1 release"
ei deployments create my-app --source staging --target production --no-wait
ei deployments create my-app --source staging --target production --json
```

```bash
# TODO Deployment message
ei deployments create <app-namespace> --message "<value>"

# TODO Deployment name
ei deployments create <app-namespace> --name "<value>"

# TODO Don't wait for deployment to complete
ei deployments create <app-namespace> --no-wait

# TODO Source environment name or ID (required)
ei deployments create <app-namespace> --source "<value>"

# TODO Target environment name or ID (required)
ei deployments create <app-namespace> --target "<value>"

# TODO Timeout for waiting (default 10m) (default 10m0s)
ei deployments create <app-namespace> --timeout 5m
```

## list - TODO

List deployments in the organization.

```bash
ei deployments list
ei deployments list --app my-app
ei deployments list --limit 50
ei deployments list --json
```

```bash
# TODO Filter by app namespace
ei deployments list --app "<value>"

# TODO Maximum number of deployments to return (default 20)
ei deployments list --limit <value>
```

## show - TODO

Show detailed information about a deployment including status,
deployed entities, and missing configurations.

When used with --json, includes a config_template that can be used as a
starting point for ei deployments configure.

```bash
ei deployments show abc-123-uuid
ei deployments show abc-123-uuid --json
```
