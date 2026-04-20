# Elementum CLI

A beautiful, interactive CLI tool for discovering and exporting Elementum resources to Terraform.

## Features

- 🔐 **Secure Authentication** - Store credentials safely with OS keychain integration
- 📋 **List Objects** - Display all objects (apps and elements) in your organization
- 🌳 **Show Details** - View app structure in a beautiful tree format
- 📦 **Export to Terraform** - Generate import configurations with `terraform plan -generate-config-out`
- 🚀 **Terraform/Tofu Helpers** - Run plan, apply, and destroy with auto-injected credentials
- 🎨 **Beautiful UI** - Powered by the Charm stack (Bubble Tea, Lip Gloss, Huh)
- 🔄 **GraphQL Client** - Type-safe GraphQL queries generated with genqlient

## Installation

### From Source

```bash
go build -o ~/go/bin/ei .
```

The CLI will be installed to `~/go/bin/ei`. Make sure `~/go/bin` is in your `$PATH`.

### From Homebrew (coming soon)

```bash
brew install elementumltd/tap/elementum
```

## Quick Start

### 1. Authenticate

```bash
ei auth login
```

This will interactively prompt for your:

- Organization ID
- Client ID
- Client Secret
- Environment (production, staging, development)
- Profile name

Credentials are securely stored in your OS keychain.

### 2. List Objects

```bash
ei objects list
```

Shows a table of all objects (apps and elements) with their types, names, and namespaces.

### 3. Show App Details

```bash
# By namespace
ei apps show accountassignment

# By URL (in quotes)
ei apps show "https://appdemo.elementum.io/app/accountassignment"
```

Displays a tree view of the app's structure including:

- Fields
- Layouts
- Flows
- Automations (with triggers and tasks)
- Agents (with tools)
- Widgets
- AI File Readers
- Approval Processes

### 4. Export to Terraform

```bash
# apps exports (interactive selection)
ei apps export accountassignment

# apps exports by URL (in quotes)
ei apps export "https://appdemo.elementum.io/app/accountassignment"

# Custom output file
ei apps export accountassignment --output my-app.tf

# Multiple output files
ei apps export accountassignment --directory my/custom/directory

# Export everything without prompting
ei apps export accountassignment --all

# Export elements
ei elements export products                # By namespace
ei elements export elem_12345678           # By ID

# Export groups
ei groups export "Engineering Team"        # By name
ei groups export grp_12345678              # By ID

# Export CloudLinks
ei cloudlinks export "Production API"      # By name
ei cloudlinks export cl_12345678           # By ID

```

The CLI will:

1. Discover all resources in the app/element/group
2. Generate import blocks
3. Run `terraform init` and `terraform plan -generate-config-out`
4. Output a ready-to-use Terraform configuration file

For apps, interactive mode lets you select which resource types to export:

- Fields (text, number, dropdown, etc.)
- Layouts (display blocks)
- Flows (process diagrams)
- Automations (includes triggers and tasks)
- Agents (includes tools)
- Widgets
- AI File Readers
- Approval Processes

This generates an `imports.tf` file with Terraform import blocks. Then run:

```bash
terraform plan -generate-config-out=generated.tf
```

## Commands

### Authentication

```bash
ei auth login                  # Interactive sign-in
ei auth login --profile work   # Save as named profile
ei auth status                 # Show current auth status
ei auth logout                 # Clear stored credentials
ei auth switch <profile>       # Switch between profiles
```

### Discovery

```bash
ei objects list                      # List all objects (apps and elements)
ei apps show accountassignment       # Show app details by namespace
ei apps show "https://..."           # Show app details by URL
```

### Export

```bash
# apps exports
ei apps export accountassignment                    # Interactive selection
ei apps export accountassignment --all              # Export everything
ei apps export accountassignment --output my-app.tf # Custom output file
ei apps export accountassignment --directory my/custom/directory # Multiple output files

# Export elements
ei elements export products                        # By namespace
ei elements export elem_12345678                   # By ID
ei elements export products -o element.tf          # Custom output file

# Export groups
ei groups export "Engineering Team"                # By name
ei groups export grp_12345678                      # By ID
ei groups export "Engineering Team" -o group.tf    # Custom output file

# Export CloudLinks
ei cloudlinks export "Production API"              # By name
ei cloudlinks export cl_12345678                   # By ID
ei cloudlinks export "Production API" -o cloudlink.tf  # Custom output file
```

### Stored Functions

```bash
ei functions list                          # List all stored Snowflake functions
ei functions delete "Invoice Parser"       # Delete by display name (with confirmation)
ei functions delete <uuid> --force         # Delete by ID without confirmation
```

### A2A Skills

```bash
ei a2a-skills list "Support Bot"           # List A2A skills on an agent
ei a2a-skills list <agent-uuid>            # List by agent ID
ei a2a-skills delete <skill-uuid>          # Delete an A2A skill
```

### Terraform/Tofu Helpers

```bash
ei plan                    # Run terraform/tofu plan with auto-injected auth
ei plan -out=tfplan        # Save plan to file
ei apply                   # Run terraform/tofu apply with auto-injected auth
ei apply -auto-approve     # Apply without confirmation
ei destroy                 # Run terraform/tofu destroy with auto-injected auth
ei destroy -auto-approve   # Destroy without confirmation
```

These commands automatically:

- Detect whether to use `terraform` or `tofu` (prefers tofu if both are available)
- Inject your stored Elementum credentials from currently active `ei auth` profile as environment variables (`ELEMENTUM_ORGANIZATION`, `ELEMENTUM_CLIENT_ID`, `ELEMENTUM_CLIENT_SECRET`, `ELEMENTUM_ENVIRONMENT`)
- Pass through all flags to the underlying terraform/tofu command
- Provide enhanced, colored output for better readability

## Configuration

### Config File Location

- **macOS/Linux:** `~/.config/ei/config.yaml`
- **Windows:** `%APPDATA%\ei\config.yaml`

### Multiple Profiles

You can manage multiple environments or organizations:

```bash
# Add a staging profile
ei auth login --profile staging

# Switch to staging
ei auth switch staging

# Use a specific profile for a command
ei objects list --profile staging
```

### Credential Priority

The CLI resolves credentials in this order:

1. Command-line flags (`--org`, `--client-id`, etc.)
2. Environment variables (`ELEMENTUM_ORGANIZATION`, `ELEMENTUM_CLIENT_ID`, etc.)
3. Config file (current profile)

## Architecture

The CLI is structured as:

```
elementum-cli/
├── main.go              # Entry point
├── cmd/                 # Command implementations
│   ├── auth.go          # Authentication commands
│   ├── apps.go          # App commands
│   ├── agents.go        # Agent commands
│   └── ...              # Other resource commands
├── auth/                # Authentication & config management
│   ├── config.go        # Config file handling
│   ├── credentials.go   # Keychain integration
│   ├── login.go         # Login flow
│   └── client.go        # API client creation
├── analysis/            # Automation & execution analysis
├── discovery/           # Resource discovery
│   ├── app.go           # App discovery logic
│   ├── unified.go       # Unified discovery
│   └── types.go         # Resource types
├── export/              # HCL/Terraform generation
│   ├── hcl_generator.go # HCL code generation
│   ├── beautify.go      # Output formatting
│   └── ...              # Resource-specific exporters
├── internal/            # Internal packages
│   ├── client/          # GraphQL client (genqlient)
│   ├── fieldtypes/      # Field type mapping
│   └── logging/         # Structured logging
├── logger/              # CLI log levels
├── server/              # Local server for auth flow
├── state/               # State management
├── terraform/           # Terraform/Tofu execution
├── ui/                  # UI components (Charm stack)
└── update/              # Self-update mechanism
```

### GraphQL Client

The CLI includes a type-safe GraphQL client in `internal/client/`, generated with [genqlient](https://github.com/Khan/genqlient) from the operation files in `internal/client/operations/`:

- `SearchAspectsQuery` - Lists apps and elements
- `GetAspectFieldsQuery` - Fetches app fields with configuration
- `GetAspectAgentsQuery` - Fetches app agents and their tools

Discovery-specific queries (for automations, layouts, flows, widgets) are defined in the CLI's `discovery` package since they're only used for export generation.

- `GetAspectWidgetsQuery`

## Import ID Formats

The CLI generates import IDs in the same format expected by the Terraform provider:

| Resource Type              | Import ID Format                    |
| -------------------------- | ----------------------------------- |
| `elementum_app`            | `{app_id}`                          |
| `elementum_element`        | `{element_id}`                      |
| `elementum_group`          | `{group_id}`                        |
| `elementum_cloudlink`      | `{cloudlink_id}`                    |
| `elementum_*_field`        | `{app_id}:{field_id}`               |
| `elementum_layout`         | `{app_id}:{layout_id}`              |
| `elementum_automation`     | `{app_id}:{automation_id}`          |
| `elementum_*_trigger`      | `{automation_id}:{trigger_id}`      |
| `elementum_*_task`         | `{workflow_id}:{task_id}`           |
| `elementum_agent`          | `{app_id}:{agent_id}`               |
| `elementum_agent_*_tool`   | `{app_id}:{agent_id}:{tool_id}`     |
| `elementum_ai_file_reader` | `{app_id}:{file_reader_id}`         |
| `elementum_approval_process` | `{app_id}:{approval_id}`          |
| `elementum_widget`         | `{app_id}:{widget_id}`              |

## Examples

### Export an entire app

```bash
# Authenticate
ei auth login

# Find your app
ei objects list

# View app structure (by namespace)
ei apps show accountassignment

# Export everything
ei apps export accountassignment --all --directory my/custom/directory

# Generate Terraform config
terraform plan -generate-config-out=generated.tf

# Review and customize
cat generated.tf
```

### Export an element

```bash
# Authenticate
ei auth login

# List all objects to find your element
ei objects list 

# Export element by namespace
ei elements export products

# Or by ID
ei elements export  elem_12345678

# Review the generated config
cat generated.tf
```

### Export a group

```bash
# Authenticate
ei auth login

# Export group by name (use quotes if name has spaces)
ei groups export "Engineering Team"

# Or by ID
ei groups export  grp_12345678

# Review the generated config
cat generated.tf
```

### Export a CloudLink

```bash
# Authenticate
ei auth login

# Export CloudLink by name (use quotes if name has spaces)
ei cloudlinks export "Production API"

# Or by ID
ei cloudlinks export cl_12345678

# Review the generated config
cat generated.tf
```

### Complete workflow with terraform helpers

```bash
# Authenticate once
ei auth login

# Export an app
ei apps export myapp --all

# Review the plan (credentials auto-injected)
ei plan -generate-config-out=generated.tf

# Review generated config
cat generated.tf

# Apply changes (credentials auto-injected)
ei apply

# Later, if you need to destroy
ei destroy
```

### Export only specific resources

```bash
# Export with interactive selection
ei apps export accountassignment

# Select:
# [x] Fields
# [x] Automations
# [ ] Layouts
# [ ] Agents

# Generates imports.tf with selected resources
```

### Multiple environments

```bash
# Production
ei auth login --profile prod
ei apps export app_prod_123 --all -o prod-imports.tf

# Staging
ei auth login --profile staging
ei apps export app_staging_456 --all -o staging-imports.tf
```

## Development

### Build

```bash
go build -o ~/go/bin/ei .
```

### Test

```bash
go test ./...
```

See [TESTING.md](TESTING.md) for detailed information about test coverage.

### Dependencies

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Styling
- [Huh](https://github.com/charmbracelet/huh) - Forms and prompts
- [go-keyring](https://github.com/zalando/go-keyring) - Secure credential storage

## License

Apache-2.0 — see [LICENSE](LICENSE) for details.
