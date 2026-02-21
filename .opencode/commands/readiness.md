---
description: Run full spec readiness checks
agent: general
subtask: true
---
Run the repository readiness checks and summarize results by failing check first.

Check output:
!`bash scripts/check-readiness.sh`

Return:
1. Overall status (pass/fail).
2. Any failing checks with minimal remediation steps.
3. If all green, confirm the repo is ready for implementation planning.
