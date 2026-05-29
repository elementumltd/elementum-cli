# Tables
Link to documentation https://docs.elementum.io/getting-started/fundamentals/core-concepts#manage-data-with-tables

## Available commands

| Command | Purpose |
|--|--|
| create | Create a multi-join table |
| export | Export a table to Terraform |
| list | List all tables |

## list - List Tables

```bash
# Get list of all applications
ei tables list
```

## create - Create a table
This command is interactive and MUST be ran by user in their own terminal. Instruct user to open the terminal and complete steps. 

```bash
# Create a table through wizard
ei tables create 

# Create table rom YAML config file (format defined in help output of this comand)
ei tables create --config table.yaml

# Generate curl command instead of Terraform
ei tables create --config table.yaml --curl
```