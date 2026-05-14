# Cloudlinks

Commands for listing and managing cloudlinks (data connectors) in your organization.

## Available Commands

| Command | Purpose |
|--|--|
| explore | Explore databases, schemas, tables, and columns in a Snowflake cloudlink |
| export | Export a CloudLink to Terraform |
| functions | List configured and available Snowflake functions on a cloudlink |
| list | List all cloudlinks |
| search-services | List Cortex Search Services on a Snowflake cloudlink or AI provider |

## explore - TODO

Hierarchically explore a Snowflake cloudlink's structure.

With no extra arguments, lists databases.
With a database name, lists schemas in that database.
With database and schema, lists tables in that schema.
With database, schema, and table, lists columns in that table.

Use flags to jump directly to a level:
  --database ANALYTICS                        Jump to schemas in ANALYTICS
  --database ANALYTICS --schema PUBLIC        Jump to tables
  --database DB --schema SCH --table TBL      Jump to columns

```bash
ei cloudlinks explore my-snowflake              # List databases
ei cloudlinks explore my-snowflake ANALYTICS    # List schemas in ANALYTICS
ei cloudlinks explore my-snowflake ANALYTICS PUBLIC  # List tables
ei cloudlinks explore my-snowflake ANALYTICS PUBLIC MY_TABLE  # List columns
ei cloudlinks explore my-snowflake --database DB --schema SCH --table TBL  # Same as above
```

```bash
# TODO Filter to specific database
ei cloudlinks explore <cloudlink> [database] [schema] [table] --database "<value>"

# TODO Filter to specific schema (requires --database)
ei cloudlinks explore <cloudlink> [database] [schema] [table] --schema "<value>"

# TODO Filter to specific table (requires --database and --schema)
ei cloudlinks explore <cloudlink> [database] [schema] [table] --table "<value>"
```

## export - TODO

Export a CloudLink as Terraform configuration.

You can specify the CloudLink by:
- Name: Production API
- ID: cl_12345678

The command will generate an import block and run terraform to create the configuration.

```bash
# TODO Output file for generated Terraform configuration (default "generated.tf")
ei cloudlinks export [name-or-id] --output "<value>"
```

## functions - TODO

Show both configured (stored) and available Snowflake functions for a cloudlink.

Configured functions are already set up in Elementum. Available functions are discovered
from the Snowflake database and can be configured as elementum_function resources.

The --database and --schema flags are required to scope the Snowflake query and avoid timeouts.

```bash
# TODO Filter available functions by database name
ei cloudlinks functions <cloudlink-id-or-name> --database <DB> --schema <SCHEMA> --database "<value>"

# TODO Filter available functions by schema name
ei cloudlinks functions <cloudlink-id-or-name> --database <DB> --schema <SCHEMA> --schema "<value>"

# TODO Filter by function type: procedure, udf
ei cloudlinks functions <cloudlink-id-or-name> --database <DB> --schema <SCHEMA> --type "<value>"
```

## list - TODO

Display a table of all cloudlinks (data connectors) in your organization.

## search-services - TODO

List available Snowflake Cortex Search Services that can be linked via elementum_linked_ai_search_table.

This command discovers existing Cortex Search Services in your Snowflake account that can be
connected to Elementum. Use the service name, database, and schema in your Terraform configuration.

You can specify either:
- A CloudLink name or ID
- A Snowflake AI provider name

The --database and --schema flags are required to scope the query.

```bash
ei cloudlinks search-services my-snowflake --database ANALYTICS --schema PUBLIC
ei cloudlinks search-services "Snowflake" --database ELEMENTUM --schema PUBLIC
```

```bash
# TODO Filter by database name (required)
ei cloudlinks search-services <cloudlink-or-ai-provider> --database <DB> --schema <SCHEMA> --database "<value>"

# TODO Filter by schema name (required)
ei cloudlinks search-services <cloudlink-or-ai-provider> --database <DB> --schema <SCHEMA> --schema "<value>"
```
