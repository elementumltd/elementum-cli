<!-- Copyright 2026 Elementum Ltd. All Rights Reserved.
Licensed under the Apache License, Version 2.0 -->

# Security Policy

## Reporting a Vulnerability

The Elementum team takes the security of our software seriously. If you believe you have found a security vulnerability in `elementum-cli`, please report it responsibly.

**Please do not file a public GitHub issue for security vulnerabilities.**

Instead, use GitHub's private vulnerability reporting feature:

[Report a vulnerability](https://github.com/elementumltd/elementum-cli/security/advisories/new)

## What to Include

To help us triage and respond quickly, please include:

- A description of the vulnerability
- Steps to reproduce the issue
- The potential impact (e.g., credential exposure, remote code execution)
- Any suggested fixes, if you have them

## What to Expect

- **Acknowledgment** within 48 hours of your report
- A **detailed response** within 5 business days, including our assessment and next steps
- We will work with you to understand and resolve the issue before any public disclosure

## Credit

We believe in recognizing the work of security researchers. Unless you prefer to remain anonymous, we will credit you in the release notes and any associated advisory.

## Scope

The following are considered security issues:

- Authentication or credential handling flaws (e.g., keychain storage, token leakage)
- Command injection or arbitrary code execution
- Sensitive data exposure in logs, error messages, or temporary files
- Dependency vulnerabilities that are exploitable in the context of this CLI

The following are generally **not** security issues (please file a regular [GitHub issue](https://github.com/elementumltd/elementum-cli/issues) instead):

- Bugs that do not have a security impact
- Feature requests
- Questions about usage or configuration

## Supported Versions

Security updates are applied to the latest release. We do not maintain security patches for older versions.
