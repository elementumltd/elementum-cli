# Elementum CLI Testing

This document describes the test coverage for the Elementum CLI tool.

## Running Tests

```bash
# Run all CLI tests
go test ./...

# Run tests with verbose output
go test ./... -v

# Run specific package tests
go test ./export/...
go test ./discovery/...

# Run shared field types tests
go test ./internal/fieldtypes/...
```

## Test Coverage

### 1. Field Type Mapping (`internal/fieldtypes/fieldtypes_test.go`)

Tests the GraphQL field type mapping.

**Coverage:**
- All 16 field types (text, number, dropdown, etc.)
- Handle field (system ID field)
- Relation fields
- Unknown/unrecognized field types
- Empty typename handling

**Total:** 18 test cases

### 2. Import ID Generation (`export/ids_test.go`)

Tests the generation of composite import IDs for Terraform resources.

**Coverage:**
- App imports (`app_id`)
- Field imports (`app_id:field_id`)
- Layout imports (`app_id:layout_id`)
- Automation imports (`app_id:automation_id`)
- Trigger imports (`automation_id:trigger_id`)
- Task imports (`workflow_id:task_id`)
- Agent imports (`app_id:agent_id`)
- Agent tool imports (`app_id:agent_id:tool_id`)
- Widget imports (`app_id:widget_id`)
- Unknown resource type fallback

**Total:** 10 test cases for BuildImportID + 12 additional helper function tests

### 3. Import Block Generation (`export/imports_test.go`)

Tests the generation of Terraform import blocks and HCL configuration.

**Coverage:**
- **Name Sanitization:** 9 test cases
  - CamelCase conversion
  - Special character removal
  - Space/dash/underscore handling
  - Numbers and empty names
  
- **Field Filtering:** Verifies that system fields are correctly skipped
  - Handle fields (system ID)
  - Fields with semantic tags (title, status, stage)
  - System-managed fields (`system = true`)
  - Unknown field types
  
- **Duplicate Name Handling:** Tests that duplicate resource names get unique suffixes
  
- **All Resource Types:** Tests generation of all resource types (app, fields, layouts, flows, agents, widgets)
  
- **HCL Rendering:** Tests that import blocks and provider config are correctly formatted

**Total:** 18 test cases

### 4. Discovery Layer (`discovery/app_test.go`)

Tests the app discovery and resource parsing logic.

**Coverage:**
- **URL Parsing:** 9 test cases
  - Full URLs with various paths
  - URLs without protocol
  - Plain namespaces
  - Malformed URLs
  
- **Field Type Mapping:** 18 test cases (matches provider field types)
  
- **Trigger Type Mapping:** 12 test cases
  - All trigger types (webhook, record_created, on_demand, etc.)
  
- **Task Type Mapping:** 15 test cases
  - All task types (message, update_field, api, etc.)
  - Including API and AI tasks with correct casing

**Total:** 54 test cases

## Test Statistics

```
Total Test Files: 4
Total Test Cases: ~110
Test Packages: 3 (fieldtypes, export, discovery)
```

## What's Tested

✅ **Field Type Mapping** - GraphQL field type conversion logic  
✅ **Import ID Generation** - All composite ID formats  
✅ **Field Filtering** - System field exclusion logic  
✅ **Name Sanitization** - Terraform resource name conversion  
✅ **Duplicate Handling** - Unique name generation  
✅ **HCL Generation** - Import blocks and provider config  
✅ **URL Parsing** - Namespace extraction from URLs  
✅ **Type Mapping** - All GraphQL typename mappings  

## What's Not Tested

⚠️ **Authentication** - Keyring integration is complex, requires mocking  
⚠️ **API Integration** - GraphQL queries require live API or mocking  
⚠️ **Terraform Execution** - `terraform init/plan` requires terraform binary  
⚠️ **UI Components** - TUI components (Bubble Tea, Huh) require terminal  

## Test Philosophy

The test suite focuses on **pure logic** that can be tested without external dependencies:
- String manipulation (sanitization, parsing)
- Data transformation (type mapping, ID generation)
- Filtering logic (system field exclusion)
- HCL generation (string formatting)

Components that require external dependencies (API, keyring, terraform, TUI) are better suited for integration tests or manual testing.

## Continuous Integration

These tests are designed to run in CI environments without:
- Network access
- Terraform installation
- Keyring/keychain access
- Terminal/TTY

All tests use pure Go stdlib and run in milliseconds.
