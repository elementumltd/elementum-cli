# Users

## Available commands

| Command | Purpose |
|--|--|
| list | List all users |
| search | Search for users by name or email |

## list - List All Users
List all users in your organization. Supports pagination with --limit, --after, and --all flags.

```bash
# List a page of users (default 25)
ei list users

# Retrieve the next page of users with cursor value from previous call
ei list users --after CursorValueFromPreviousPage

# List a page of users with page size specified
ei list users --limit 42

# List all users
ei list users --all
```

## search - Search for Users
Search for users in your organization by name or email address (case-insensitive partial match). Uses server-side filtering for efficiency.

```bash
# Search for users whose name or email contains "C"
ei users search C

# Search for users whose name or email contains "elementum"
ei users search elementum

# Search for user with specific email
ei users search user@domain.com
```