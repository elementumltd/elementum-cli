# Documentation of the application

Ask user for the application name if it was not provided

Create a document (markdown by default) named after the application name

- App name and its details (Read [apps.md](apps.md))
- Short description of what the app does, inferred from the rest of the report
- List of app fields and their datatypes in table (Read [fields.md](fields.md))
  - field name, field type, required
  - add a category column inferred from their usage in the system
- List of stages
- List of status values and which stages they might map to
- List of layouts with their fields in them
- List of file readers (Read [file-readers.md](file-readers.md))
  - Show details of each file reader
  - Try to guess the purpose from the rest of the application
- List of automations (Read [automations.md](automations.md))
  - Name, Type of Trigger, Number of Steps
  - Show published/unpublished status
  - Come up with short summary of what it does
- Show diagram of end to end pipeline of interactions between record statuses and automations
- For each automation
  - Run `automation show` in the non-json mode using `--from-remote` parameter and no `--json` parameter and output the results exactly as returned in fixed-width code block
  - Run `automation config` and output the results exactly as returned in fixed-width code block
  - make a diagram out of each of the automation (retrieve it via JSON), analyze and render using mermaid
- List all agents (Read [agents.md](agents.md))
- List agents tools (Read [agent-tools.md](agent-tools.md))
- List all agent skills (Read [skills.md](skills.md))
- List all agent skill tools (Read [skill-tools.md](skill-tools.md))