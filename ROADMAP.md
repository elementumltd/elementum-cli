<!-- Copyright 2026 Elementum Ltd. All Rights Reserved.
Licensed under the Apache License, Version 2.0 -->

# Roadmap for Elementum CLI

These are the things that Elementum dev team are planning for the Elementum CLI.

- Installable Binaries and Release
    - Create a release with binaries for Mac, Windows and Linux in Releases
    - Create Homebrew release
- Building on Other Platforms
    - Validate build/contribution from:
        - Windows
        - Linux
- Enabling Claude
    - Claude Code skill with [Skills](/skills/README.md)
    - Claude Cowork installable skill with correct platform binaries in the VM
- Refactor of Code:
    - Change of `ei records` to public API (CRUD)
    - Change of `ei users` to public API (CRUD)
    - Change of `ei groups` to public API (RUD)
- Configuration as Code
    - Change to the upcoming JSON format with TF
    - Remove dependency on the terraform provider
- New feature for audit export (similar to https://github.com/elementumltd/org_activity_report)
    - Audit of user activity
    - Audit of changes to record
    - Audit of changes to app
- Better documentation for automation 
    - Showing of the If/Else Switch clauses and loops
    - Export of diagram as Mermaid diagram