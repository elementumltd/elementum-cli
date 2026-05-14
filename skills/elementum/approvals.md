# Approvals

Commands for listing and responding to approval requests.

## Available Commands

| Command | Purpose |
|--|--|
| approve | Approve an approval request |
| deny | Deny an approval request |
| list | List approvals for a record |

## approve - TODO

Approve a pending approval request by its ID.

```bash
# TODO Optional comment/reason for the approval
ei approvals approve <approval-id> --comment "<value>"
```

## deny - TODO

Deny a pending approval request by its ID.

```bash
# TODO Optional comment/reason for the denial
ei approvals deny <approval-id> --comment "<value>"
```

## list - TODO

List all approval chains and their status for a specific record.

```bash
# TODO Record ID to list approvals for (required)
ei approvals list --record-id "<value>"
```
