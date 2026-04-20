# Testing Guide for Elementum CLI

## Overview

This guide covers testing the Elementum CLI, including unit tests, integration tests, and manual testing with real apps.

## Unit Tests

### Running All Tests

```bash
# Run all tests from the repo root
go test ./...
```

### Running Specific Test Packages

```bash
# Discovery tests
go test ./discovery/... -v

# Export tests
go test ./export/... -v
```

### Test Coverage

```bash
# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

## Manual Testing with Real Apps

### Test App

The CLI can be tested against real Elementum apps. Example:
- **URL**: https://appdemo.elementum.io/admin/apps/dealclosingprocess/flow
- **Namespace**: `dealclosingprocess`

### Setup

```bash
# 1. Build the CLI
go build -o ~/go/bin/ei .

# 2. Authenticate
ei auth login
# Follow prompts to enter credentials
```

### Test Commands

#### 1. List Objects

```bash
ei objects list
```

Expected output:
- Shows all apps and elements in your organization
- Displays type, name, and namespace

#### 2. Show App Details

```bash
# By namespace
ei apps show dealclosingprocess

# By URL
ei apps show "https://appdemo.elementum.io/admin/apps/dealclosingprocess/flow"
```

Expected output:
```
Deal Closing Process (app_xyz123)
├─ Fields (N)
│  ├─ Title [text] required
│  ├─ Status [dropdown] (3 options)
│  └─ ...
├─ Layouts (N)
├─ Flows (N)
├─ Automations (N)
├─ Agents (N)
├─ Widgets (N)
├─ AI File Readers (N)
└─ Approval Processes (N)

Summary
  • N fields
  • N layouts
  • N automations
  • N agents
  • N flows
  • N widgets
  • N AI file readers
  • N approval processes
```

#### 3. Export App (Interactive)

```bash
ei apps export dealclosingprocess
```

Expected behavior:
- Shows interactive multi-select
- Displays counts for each resource type
- Includes AI File Readers and Approval Processes options
- Generates import blocks and Terraform config

#### 4. Export App (All Resources)

```bash
ei apps export dealclosingprocess --all -o dealclosingprocess.tf
```

Expected behavior:
- Discovers all resources automatically
- Generates import blocks for:
  - App
  - Fields (all types)
  - Layouts
  - Flows
  - Agents + Tools
  - Widgets
  - AI File Readers
  - Approval Processes
- Runs terraform init and plan
- Outputs generated.tf

#### 5. Export Element

```bash
# Create a test element first, then:
ei elements export <namespace-or-id>
```

#### 6. Export Group

```bash
ei groups export "Engineering Team"
```

#### 7. Export CloudLink

```bash
ei cloudlinks export "Production API"
```

### Verification Steps

After export, verify the generated Terraform:

```bash
# 1. Check the generated file
cat generated.tf

# 2. Validate Terraform syntax
terraform validate

# 3. Check plan (should show no changes after import)
terraform plan

# Expected: "No changes. Your infrastructure matches the configuration."
```

## Integration Testing Checklist

### Discovery

- [ ] ParseNamespaceFromURL handles various URL formats
- [ ] GetAppByNamespace finds apps correctly
- [ ] GetApp discovers all resource types:
  - [ ] Fields
  - [ ] Layouts
  - [ ] Flows
  - [ ] Automations
  - [ ] Agents
  - [ ] Widgets
  - [ ] AI File Readers
  - [ ] Approval Processes
- [ ] GetElement works by namespace and ID
- [ ] GetGroup works by name and ID
- [ ] GetCloudLink works by name and ID

### Export

- [ ] SanitizeName converts names correctly
- [ ] BuildImportID generates correct formats:
  - [ ] `{app_id}` for apps
  - [ ] `{element_id}` for elements
  - [ ] `{group_id}` for groups
  - [ ] `{cloudlink_id}` for cloudlinks
  - [ ] `{app_id}:{field_id}` for fields
  - [ ] `{app_id}:{file_reader_id}` for AI file readers
  - [ ] `{app_id}:{approval_id}` for approval processes
- [ ] GenerateImportBlocks handles all resource types
- [ ] Duplicate names get suffixes (_1, _2, etc.)
- [ ] RenderImportBlocks produces valid HCL

### CLI Commands

- [ ] `apps show` displays all resources in tree
- [ ] `apps export` includes all resource types in interactive mode
- [ ] `apps export --all` exports everything
- [ ] `elements export` works for elements
- [ ] `groups export` works for groups
- [ ] `cloudlinks export` works for cloudlinks
- [ ] Help text is accurate
- [ ] Error messages are helpful

### ImportState

- [ ] All resources implement ImportState interface
- [ ] Import IDs match expected formats
- [ ] Terraform import works end-to-end

## Test Data

### Sample App Structure

For comprehensive testing, create an app with:
- Multiple field types (text, number, dropdown, user, etc.)
- At least 2 layouts/stages
- 1+ flows
- 1+ automations with triggers and tasks
- 1+ agents with tools
- 1+ widgets
- 1+ AI file readers
- 1+ approval processes

### Sample Element

- Name: "Products"
- Handle: "PROD"
- Namespace: "products"
- Fields: Title, Description, SKU, Price

### Sample Group

- Name: "Engineering Team"
- Approver: true
- Members: 2+

### Sample CloudLink

- Name: "Production API"
- Type: "api"
- Valid: true

## Debugging

### Enable Verbose Mode

```bash
ei apps show dealclosingprocess -v
ei apps export dealclosingprocess --verbose
```

### Check GraphQL Queries

```bash
# Set debug env var
export DEBUG_GRAPHQL=1
ei apps show dealclosingprocess
```

### Check Terraform Working Directory

When export runs, it shows:
```
Working directory: /var/folders/.../elementum-export-xyz
```

You can inspect this directory to debug:
```bash
cd /var/folders/.../elementum-export-xyz
cat provider.tf
cat imports.tf
terraform plan
```

## Continuous Integration

### Pre-commit Tests

```bash
# Run before committing
go test ./...
go vet ./...
golangci-lint run
```

## Known Issues & Limitations

1. **Email Aliases** - Not exportable (no ImportState in provider)
2. **Automations** - Excluded from export (per requirements)
3. **System Fields** - Skipped automatically (TITLE, STATUS, etc.)
4. **Duplicate Names** - Get numeric suffixes

## Success Criteria

All tests pass:
```
✅ discovery tests
✅ export tests
✅ CLI builds without errors
✅ Commands work with real apps
✅ Generated Terraform is valid
✅ Terraform import succeeds
```

## Getting Help

If tests fail:
1. Check error messages carefully
2. Enable verbose mode
3. Verify authentication
4. Check API permissions
5. Review generated files in temp directory

For issues with specific resources:
1. Verify resource exists in app
2. Check import ID format
3. Verify ImportState implementation
4. Test with minimal example
