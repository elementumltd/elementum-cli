# Records
Link to documentation https://docs.elementum.io/workflows/create-a-record#create-a-record

## Available commands

| Command | Purpose |
|--|--|
| create | Create a new record in an app, element, or task |
| delete | Delete records from an app, element, or task |
| export | Export records to a file (CSV, JSON, JSONL) |
| generate | Generate synthetic records for testing |
| get | Get a single record by handle |
| import | Import records from a file (CSV, JSON, JSONL) |
| list | List records from an app, element, or task |
| update | Update an existing record |

## Field Value Resolution

| Input | Resolution
|--|--
| Field names | Case-insensitive match
| Picklist labels | Resolved to IDs (e.g., "High" → UUID)
| User emails | Looked up via API
| Dates | YYYY-MM-DD
| Booleans | true/false, yes/no, 1/0

## list - List Records

```bash
# List a page of records (25 by default) in table format
ei records list <namespace>

# List a page of records (25 by default) in table format using URL to the app or table
ei records "https://yourorg.elementum.io/app/yournamespace"

# List a page of records with page size specified
ei records list <namespace> --limit 42

# List all records
ei records list <namespace> --all

# Retrieve the next page of records with cursor value from previous call
ei records list <namespace> --after CursorValueFromPreviousPage

# List a page of records with just these columns
ei records list <namespace> --columns "Title,Status,Priority"

# List records with Status column filtered to the specific value
ei records list <namespace> --status "SomeStatusValue"

# List records with some field column filtered to the specific value
ei records list <namespace> --where "FieldName=SomeValue"

# List records with some field column filtered to the specific value via raw GraphQL filter specifier. Wrap the filter in single quotes ` to avoid escaping double-quotes "
ei records list <namespace> --filter --filter '{"type":"EQUALS","field":"Status","value":{"type":"TEXT","value":"Open"}}'
```

## get - Get A Single Record

```bash  
# Get a single record
ei records get <namespace> <record-id>

# Get a single record with explicit --id parameter. Requires GUID ID of the application
ei records get --id "<IDOfTheApplication>:<record-id>"
```

## create - Create a New Record

```bash  
# Create new record with specific fields. Multiple fields can be updated with multiple --field parameters
ei records create <namespace> --field "Title=Bug Report" --field "Priority=High" --field "Assigned User=jane@company.com"

# Create new record with fields from JSON file
ei records create <namespace> --from-file data.json

# Create new record in dry run mode, not actually creating anything
ei records create <namespace> --dry-run

# Attach files (not sure how it works right now)
ei records create <namespace> --attach "FieldName=/path/to/file"

# Create record interactively by asking user for value of each field (only works in user's console)
ei records create <namespace> --interactive
```

## update - Update an Existing Record
Update fields on an existing record in an Elementum App, Element, or Task.

You can specify the record using namespace and handle, or by full record ID.

```bash  
# Update record fields. Multiple fields can be updated with multiple --field parameters
ei records update <namespace> <record-id> --field "Status=Closed" --field "Priority=Low"

# Update record with explicit --id parameter. Requires GUID ID of the application
ei records update --id "<IDOfTheApplication>:<record-id>" --field "Status=Closed"

# Update from JSON string, Wrap the JSON block in single quotes ` to avoid escaping double-quotes "
ei records update <namespace> <record-id> --data '{"Status": "Closed"}'
```

## delete - Delete an Existing Record
Delete one or more records from an Elementum App, Element, or Task.

You can specify the record using namespace and handle, or by full record ID.

```bash  
# Delete specific record
ei records delete <namespace> <record-id>

# Delete specific record in dry-run mode without actually deleting anything
ei records delete <namespace> <record-id>

# Delete record with explicit --id parameter. Requires GUID ID of the application
ei records delete --id "<IDOfTheApplication>:<record-id>"

# Delete all records in app or element skipping user confirmation
ei records delete <namespace> --all --force
```

## import - Import Records from External File
Bulk import records into an Elementum App from a file.

Supported formats: CSV, TSV, JSON, JSONL/NDJSON

CSV files should have a header row with field names as columns.
JSON files can be an array of objects, or an object with a "records" key.
JSONL files should have one JSON object per line.

```bash
# Import from CSV
ei records import <namespace> data.csv

# Import from JSON
ei records import <namespace> data.json

# Import from JSONL/NDJSON
ei records import <namespace> data.jsonl

# Import from file with explicit file type defined (csv, json, jsonl)
ei records import <namespace> data.txt --format csv

# Import from CSV in dry-run mode (validate without creating)
ei records import <namespace> data.csv --dry-run
```

## export - Export Records to File
Export all records from an Elementum App to a file.

The output format is auto-detected from the file extension.
All records are fetched (auto-paginated) and written to the file.

```bash  
# Export to CSV
ei records export <namespace> data.csv

# Export to JSON
ei records export <namespace> data.json

# Export to JSONL
ei records export <namespace> data.jsonl

# Export with explicit file type defined (csv, json, jsonl)
ei records export <namespace> data.txt --format csv
```

## generate - Create Synthetic Records
Generate and optionally create synthetic records in an Elementum App.

Records are generated based on the app's field schema, producing realistic
test data for each field type. By default records are created in the app;
use --dry-run to preview or --output to write to a file instead.

```bash  
# Generate a specified number of the records and create them
ei records generate <namespace> --count 100

# Generate a specified number of the records in dry run mode without actually creating them
ei records generate <namespace> --count 5 --dry-run

# Generate a specified number of the records and write them to a file without creating them in in app. The file format is inferred from extension (csv, json, jsonl). This can be useful in Import later
ei records generate <namespace> --count 100 --output test-data.json

# Reproducible generation with seed
ei records generate <namespace> --count 50 --seed 42  # Reproducible
```

## Common Workflows

```bash
# Seed some application environment
ei records generate my-app --count 200 --seed 42

# Data migration
ei records export source-app data.csv
ei records import target-app data.csv

# Cleanup
ei records delete my-app --all --force
```

## Troubleshooting

- **"missing required field 'ID'"** — object has user-managed ID field, provide unique value
- **Import slow** — CLI auto-falls back to individual creates if bulk API fails (5 parallel workers)
- **Search returns nothing after import** — AI Searßh Tables need time to re-index (check `target_lag`)
