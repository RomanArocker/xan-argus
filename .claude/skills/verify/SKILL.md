---
name: verify
description: Run Go static analysis and tests to verify the project builds and passes all checks. Use after making changes or before committing.
---

## Verify Project

Run these checks in order, stopping on first failure:

1. **Format check:** `gofmt -l .` — if any files listed, they need formatting
2. **Vet:** `go vet ./...`
3. **Lint:** `golangci-lint run ./...`
4. **Test:** `go test ./...`

Report results concisely: which checks passed, which failed, and the relevant error output.

If tests require a database, check if `DATABASE_URL` is set. If not, note which tests were skipped.
