# File-readers

Commands for listing, creating, showing, updating, and deleting Elementum file readers (document models).

## Available Commands

| Command | Purpose |
|--|--|
| create | Create a new file reader |
| delete | Delete a file reader |
| list | List file readers in an app |
| show | Show file reader details |
| update | Update a file reader |

## create - TODO

Create a new file reader (document model) in an Elementum app.

Supported types:
  - ai:   AI-powered extraction with custom fields
  - text: OCR text extraction
  - json: JSON parsing with schema
  - xml:  XML parsing with schema

Field format (for AI type): name:type:description:required
  - name: Field name
  - type: text, number, decimal, boolean, date, datetime
  - description: Field description
  - required: true or false

```bash
# AI file reader with fields
ei file-readers create support-tickets --type ai --name "Invoice Parser" \
  --instructions "Extract invoice data from the document" \
  --field "vendor_name:text:Name of the vendor:true" \
  --field "amount:decimal:Total invoice amount:true" \
  --field "invoice_date:date:Date on the invoice:false"

# Text/OCR file reader
ei file-readers create support-tickets --type text --name "Document OCR"

# JSON file reader with structure file
ei file-readers create support-tickets --type json --name "API Response" \
  --structure-file ./schema.json

# Dry run
ei file-readers create support-tickets --type ai --name "Test" \
  --instructions "Test instructions" --dry-run
```

```bash
# TODO Preview what would be created without creating
ei file-readers create <app-namespace> --dry-run

# TODO strings           Field definition: name:type:description:required (for type=ai, repeatable)
ei file-readers create <app-namespace> --field

# TODO AI instructions for extraction (required for type=ai)
ei file-readers create <app-namespace> --instructions "<value>"

# TODO File reader name (required)
ei file-readers create <app-namespace> --name "<value>"

# TODO JSON file with structure definition (for type=json/xml)
ei file-readers create <app-namespace> --structure-file "<value>"

# TODO File reader type: ai, text, json, xml (required)
ei file-readers create <app-namespace> --type "<value>"
```

## delete - TODO

Delete a file reader by name or ID.

WARNING: This permanently deletes the file reader.

```bash
ei file-readers delete support-tickets "Invoice Parser"
ei file-readers delete support-tickets "Invoice Parser" --force
ei file-readers delete support-tickets 794e1e48-73af-4760-... --force
```

```bash
# TODO Skip confirmation prompt
ei file-readers delete <app-namespace> <name-or-id> --force
```

## list - TODO

List file readers (document models) in an Elementum app.

File readers extract structured data from files. Supported types:
  - ai:   AI-powered extraction with custom fields
  - text: OCR text extraction
  - json: JSON parsing with schema
  - xml:  XML parsing with schema

```bash
ei file-readers list support-tickets
ei file-readers list support-tickets --type ai
ei file-readers list support-tickets --json
```

```bash
# TODO Filter by type: ai, text, json, xml
ei file-readers list <app-namespace> --type "<value>"
```

## show - TODO

Display detailed information about a file reader.

```bash
ei file-readers show support-tickets "Invoice Parser"
ei file-readers show support-tickets 794e1e48-73af-4760-...
```

## update - TODO

Update properties of an existing file reader.

```bash
# Update name
ei file-readers update support-tickets "Invoice Parser" --name "New Invoice Parser"

# Update AI instructions
ei file-readers update support-tickets "Invoice Parser" --instructions "New extraction instructions"

# Update fields (replaces all fields)
ei file-readers update support-tickets "Invoice Parser" \
  --field "vendor:text:Vendor name:true" \
  --field "total:decimal:Total amount:true"
```

```bash
# TODO strings           Replace fields with new definitions: name:type:description:required (for AI type)
ei file-readers update <app-namespace> <name-or-id> --field

# TODO New AI instructions (for AI type only)
ei file-readers update <app-namespace> <name-or-id> --instructions "<value>"

# TODO New file reader name
ei file-readers update <app-namespace> <name-or-id> --name "<value>"

# TODO JSON file with new structure definition (for JSON/XML type)
ei file-readers update <app-namespace> <name-or-id> --structure-file "<value>"
```
