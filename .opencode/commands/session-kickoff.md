---
description: Build a scoped implementation kickoff packet
agent: plan
subtask: true
---
Prepare a non-trivial session kickoff packet for feature spec `$ARGUMENTS`.

Required context:
- @AGENTS.md
- @docs/spec-index.md
- @docs/agent-session-playbook.md
- @docs/testing.md
- @$ARGUMENTS

Return a kickoff packet with:
1. Governing feature spec path.
2. Allowed change paths from the feature spec.
3. Explicitly forbidden paths for this session.
4. Numbered acceptance criteria to prove.
5. Verification commands to run.

Do not implement code in this command.
