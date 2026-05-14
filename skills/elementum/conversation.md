# Conversation

## conversation - TODO

Fetch an agent conversation and produce a timing/performance analysis.

Shows tool call durations, LLM thinking time, turn-by-turn latency,
and aggregate statistics. Useful for understanding agent performance
bottlenecks after a chat session or UAT test.

The agent can be specified by name (case-insensitive) or UUID.

Output modes:
  (default)     Text summary with tables
  --json        Full structured JSON analysis
  --timeline    CLI visual timeline with horizontal bars
  --html        Interactive HTML waterfall opened in browser

```bash
ei conversation "IT Support Agent" <conversation-id>
ei conversation "IT Support Agent" <conversation-id> --json
ei conversation <agent-id> <conversation-id> --timeline
ei conversation <agent-id> <conversation-id> --html
```

```bash
# TODO Open interactive HTML waterfall in browser
ei conversation <agent-name-or-id> <conversation-id> --html

# TODO Show CLI visual timeline with horizontal bars
ei conversation <agent-name-or-id> <conversation-id> --timeline
```
