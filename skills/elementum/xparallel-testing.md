# Parallel Testing with Agent Teams

For faster UAT execution, spawn parallel subagents. Each runs a subset of tests independently.

## When to Use

- **10+ test cases** — sequential becomes slow
- **Independent tests** — don't share conversation state
- **Single-turn tests** — search and clarification (Phase 1 & 2)

## Setup

Split tests across 2-4 subagents:

```
Agent 1 (Search Tests): GEN-001, GEN-003, APP-003, SF-001
Agent 2 (Clarification):  EDGE-001, EDGE-003, EDGE-004
Agent 3 (Submission):     FULL-001, FULL-002 (sequential within agent)
```

## Task Tool Invocation

Use `subagent_type: "generalPurpose"` with this prompt template:

```markdown
You are a UAT tester for Elementum agents. Execute these tests and return structured results.

**Agent ID**: {agent_id}
**Tests to Run**:

### Test {test_id}: {test_name}
- **Input**: "{intent_phrase}"
- **Expected**: {expected_behavior}
- **Evaluate**:
  - Search executed? (yes/no)
  - Groups/records returned? (list names)
  - Clarification asked? (yes/no, quote if yes)
  - Business rule applied? (which one, or none)
  - Asked for email? (should be NO)

**Execution**: Use `ei chat {agent_id} -m "<message>"` for each test.

**Output**: Return YAML results:
```yaml
tests:
  - test_id: {id}
    outcome: PASS|FAIL|PARTIAL
    conversation_id: {id}
    response_time: {from ei chat output}
    notes: {observations}
```
```

## Coordination

After all subagents complete:
1. Collect results
2. Merge PASS/FAIL counts
3. Generate unified report

## Constraints

**DO parallelize**: Single-turn search, clarification, independent business rule tests.

**DO NOT parallelize**: Multi-turn submission tests (sequential within conversation), tests sharing state, rate-limited APIs.

## Time Savings

11 tests × 30s each = 5.5 min sequential → ~3 min with 3 agents (45% faster).
