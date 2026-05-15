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

## create - Create new File Reader
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
# Create AI file reader with fields
ei file-readers create <namespace> --type ai --name "Invoice Parser" \
  --instructions "Extract invoice data from the document" \
  --field "vendor_name:text:Name of the vendor:true" \
  --field "amount:decimal:Total invoice amount:true" \
  --field "invoice_date:date:Date on the invoice:false"

# Create Text/OCR file reader
ei file-readers create <namespace> --type text --name "Document OCR"

# Create JSON file reader with structure file
ei file-readers create <namespace> --type json --name "API Response" --structure-file ./schema.json

# Create file reader in dry run mode without actually doing anything
ei file-readers create <namespace> --type ai --name "Test" --instructions "Test instructions" --dry-run
```

## delete - Delete File Reader
Delete a file reader by name or ID.

WARNING: This permanently deletes the file reader.

```bash
# Delete file reader with user confirmation
ei file-readers delete <namespace> "Invoice Parser"

# Delete file reader without prompting
ei file-readers delete <namespace> "Invoice Parser" --force

# Delete file reader by ID
ei file-readers delete <namespace> 9063aed1-bf8c-430d-882f-8c502355a3c7 --force
```

## list - List File Readers
List file readers (document models) in an Elementum app.

File readers extract structured data from files. Supported types:
  - ai:   AI-powered extraction with custom fields
  - text: OCR text extraction
  - json: JSON parsing with schema
  - xml:  XML parsing with schema

```bash
# List all File Readers
ei file-readers list <namespace>

# List all File Readers of certain type (ai, text, json, xml)
ei file-readers list <namespace> --type ai
ei file-readers list <namespace> --json
```

## show - Show Field Reader 
Display detailed information about a file reader.

```bash
# Show File Reader
ei file-readers show <namespace> "Invoice Parser"

# Show File Reader by ID
ei file-readers show <namespace> 794e19063aed1-bf8c-430d-882f-8c502355a3c7
```

## update - Update File Reader
Update properties of an existing file reader.

```bash
# Update File Reader name
ei file-readers update <namespace> "Invoice Parser" --name "New Invoice Parser"

# Update File Reader instructions
ei file-readers update <namespace> "Invoice Parser" --instructions "New extraction instructions"

# Update AI Reader fields (replaces all fields)
ei file-readers update <namespace> "Invoice Parser" \
  --field "vendor:text:Vendor name:true" \
  --field "total:decimal:Total amount:true"

# Update JSON file reader with new structure definition (for JSON/XML type)
ei file-readers update <app-namespace> <name-or-id> --structure-file "<value>"
```