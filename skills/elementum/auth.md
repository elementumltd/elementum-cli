# Authentication

## Available commands

| Command | Purpose |
|--|--|
| env | Print environment variables for current profile |
| export | Export auth profiles to a passphrase-encrypted file |
| import | Import auth profiles from a passphrase-encrypted file |
| list | List all auth profiles |
| login | Sign in to Elementum and save credentials in named profile |
| logout | Sign out from Elementum and delete named profile and stored credentials |
| rename | Rename a profile |
| status | Show authentication status |
| switch | Switch to a different profile |

## login - Authentication Login (interactive)
This command is interactive and MUST be ran by user in their own terminal. Instruct user to open the terminal and complete steps. Provide user with URL to https://docs.elementum.io/api-reference/api-introduction#step-1-create-api-credentials which explains how to obtain authentication credentials.

```bash
# 
ei auth login
```

User will be interactively prompted for:
- Instance/region (us, eu, stage, dev, custom)
- Organization ID (`acme` -> `acme.elementum.io`)
- Environment for multi-environment orgs (`staging` -> `staging-acme.elementum.io`)
- Client ID and Client Secret from your Elementum OAuth API Credentials
- Profile Name to save these credentials under /Users/dodievich/.config/ei/config.yaml

Authentication credentials are saved to:
- **macOS/Linux:** `$HOME/.config/ei/config.yaml`(aka `~/.config/ei/config.yaml`)
- **Windows:** `%USERPROFILE%\ei\config.yaml`

The secrets are stored in the OS keychain.

## status - Authentication Status for Current Profile

```bash
# Get status of current profile and all available that can be switched to
ei auth status
```

## list - List of Available Profiles

```bash
# Get all available authentication profiles
ei auth list
```

## switch - Switch To a Different Profiles

```bash
# Pass name of the profile that was retrieved using `auth list` or `auth status` commands:
ei auth switch ProfileToSwitchTo
```

## logout - Log Out and Remove Stored Credentials

```bash
# Log out of currently selected profile and delete stored credentials
ei auth logout

# Log out of specifig profile
ei auth logout --profile ProfileToLogOutFrom

```