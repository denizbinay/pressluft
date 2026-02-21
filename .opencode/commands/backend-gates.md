---
description: Run Go build, vet, and tests
agent: general
subtask: true
---
Run backend validation gates from repository root.

Gate output:
!`if [ ! -f "cmd/pressluft/main.go" ]; then echo "SKIP: cmd/pressluft/main.go not present"; else go build -o ./bin/pressluft ./cmd/pressluft && go vet ./... && go test ./internal/... -v; fi`

Return:
1. Overall status (pass/fail/skip).
2. Failing command and first actionable error.
3. Minimal next fix sequence.
