# Search-tables

Commands for querying and managing AI search tables (Cortex search).

## Available Commands

| Command | Purpose |
|--|--|
| query | Query an AI search table |
| rebuild | Rebuild an AI search table (full re-index) |
| refresh | Refresh an AI search table (incremental re-index) |

## query - TODO

Execute a semantic search query against an AI search table.

The table can be either an aspect (app/element) search table or a table search table.
By default, both types are tried automatically. Use --type to specify explicitly.

Use --filter to apply additional filters to the search results. The filter is a JSON
object with type, field, and value properties.

```bash
ei search-tables query 94828968-a51a-4c95-923b-e08f7f4b22a8 "search text"
ei search-tables query 94828968-a51a-4c95-923b-e08f7f4b22a8 "search text" --limit 5
ei search-tables query 94828968-a51a-4c95-923b-e08f7f4b22a8 "search text" --type table
ei search-tables query 94828968-a51a-4c95-923b-e08f7f4b22a8 "search text" --json

Filter examples:
# Text equals
--filter '{"type":"EQUALS","field":"<field-id>","value":{"type":"TEXT","value":"Active"}}'

# Boolean equals
--filter '{"type":"EQUALS","field":"<field-id>","value":{"type":"BOOLEAN","value":true}}'

# AND filter with multiple conditions
--filter '{"type":"AND","children":[{"type":"EQUALS","field":"<id1>","value":{"type":"TEXT","value":"A"}},{"type":"EQUALS","field":"<id2>","value":{"type":"TEXT","value":"B"}}]}'
```

```bash
# TODO JSON filter (e.g., '{"type":"EQUALS","field":"<field-id>","value":{"type":"TEXT","value":"Active"}}')
ei search-tables query <table-id> <query> --filter "<value>"

# TODO Maximum number of results to return (default 10)
ei search-tables query <table-id> <query> --limit <value>

# TODO Search table type: aspect or table (auto-detects if not specified)
ei search-tables query <table-id> <query> --type "<value>"
```

## rebuild - TODO

Trigger a full rebuild of an AI search table's Cortex Search Service.

This performs CREATE OR REPLACE, dropping and recreating the entire service.
Use sparingly - only when incremental refresh is not sufficient.

WARNING: This temporarily makes the search table unavailable.

```bash
ei search-tables rebuild 950d9901-2cf7-4ad0-94cb-c9ab01dafcf1
ei search-tables rebuild 950d9901-2cf7-4ad0-94cb-c9ab01dafcf1 --json
```

## refresh - TODO

Trigger an incremental refresh of an AI search table's Cortex Search Service.

This updates vector embeddings for records that have changed since the last refresh.
Use this for routine index maintenance.

```bash
ei search-tables refresh 950d9901-2cf7-4ad0-94cb-c9ab01dafcf1
ei search-tables refresh 950d9901-2cf7-4ad0-94cb-c9ab01dafcf1 --json
```
