# Elements
Link to documentation https://docs.elementum.io/getting-started/fundamentals/core-concepts#data-%26-elements

## Available commands

| Command | Purpose |
|--|--|
| delete | Delete an element |
| list | List all elements |
| search-tables | Manage search tables for an element |

## list - List Elements

```bash
# Get list of all elements
ei elements list
```

## create - Create new element
The `apps` command encompasses creation of element:

```bash
# Create new element 
ei apps create --name "NameOfElement" --namespace "NamespaceOfElement" --category "CategoryOfElement" --type "element"

# Create new element in dry run mode, not actually creating anything  
ei apps create --name "NameOfElement" --namespace "NamespaceOfElement" --category "CategoryOfElement" --type "element" --dry-run
```

## delete - Delete existing element
The `apps` command encompasses deletion of element:

```bash
# Delete element by name with confirmation prompt
ei elements delete NameOfElement

# Delete element by name without confirmation
ei elements delete NameOfElement --force

# Delete element by ID with confirmation prompt
ei elements delete 9063aed1-bf8c-430d-882f-8c502355a3c7

# Delete application by name in dry run mode, not actually deleting anything
ei elements delete NameOfElement --dry-run
```