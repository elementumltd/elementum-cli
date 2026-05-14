# Applications/Apps
Link to documentation https://docs.elementum.io/getting-started/build-an-app

## Available commands

| Command | Purpose |
|--|--|
| create | Create a new app or element |
| delete | Delete an app or element |
| list | List all apps |
| show | Show app details |
| update | Update an app or element |

## list - List Applications

```bash
# Get list of all applications
ei apps list
```

## show - Show Application Details in JSON form
The JSON form is useful for programmatic analysis of subsequent results:
ß
```bash
# Show detais of the application
ei apps show <namespace-or-url> --json

# Example by namespace
ei apps show docdigi --json

# Example by the app URL
ei apps show https://appdemo.elementum.io/apps/docdigi --json
```

## show - Show Application Details in Human Readable Form
Note that skipping `--json` command provides a nice human readable output:

```bash
# Show detais of the application
ei apps show [namespace-or-url]
```

## create - Create new application

```bash
# Create new application with custom name, namespace (url portion, no spaces allowed) and category. If category doesn't exist it will get created
ei apps create --name "NameOfApplication" --namespace "NamespaceOfApplicaiton" --category "CategoryOfApplication" --description "DescriptionOfApplication" -type "app"

# Create application in dry run mode, not actually creating anything
ei apps create --name "NameOfApplication" --namespace "NamespaceOfApplicaiton" --category "CategoryOfApplication" --description "DescriptionOfApplication" -type "app" --dry-run
```

## delete - Delete existing application

```bash
# Delete application by name with confirmation prompt
ei apps delete NameOfApplication

# Delete application by name without confirmation
ei apps delete NameOfApplication --force

# Delete application by ID with confirmation prompt
ei apps delete 9063aed1-bf8c-430d-882f-8c502355a3c7

# Delete application by name in dry run mode, not actually deleting anything
ei apps delete NameOfApplication --dry-run
```