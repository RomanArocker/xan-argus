# Versioning Design

**Date:** 2026-03-28
**Status:** Approved

## Overview

Add semantic versioning to XAN-Argus so operators can verify which version runs in Docker and developers can debug more easily. The version is embedded into the binary at build time and displayed in the web UI footer and the `/health` endpoint.

## Version Format

`v{semver}-{short-git-hash}` — e.g. `v1.2.0-a3f9d2c`

- Semver comes from a `VERSION` file in the repo root (e.g. `1.2.0`, no `v` prefix)
- Git hash comes from `git rev-parse --short HEAD` at build time
- Combined at runtime into a single string: `"v" + version + "-" + gitCommit`

## Components

### 1. `VERSION` File

Plain text file in the repo root containing only the current semantic version:

```
1.0.0
```

Manually bumped via Makefile targets (see below). Committed to git.

### 2. Go Package Variables (`cmd/server/main.go`)

Two package-level variables with safe defaults for local builds:

```go
var (
    version   = "dev"
    gitCommit = "unknown"
)
```

Set at build time via `-ldflags`. If built without the Makefile (e.g. `go build ./cmd/server/`), the binary reports `vdev-unknown` — this is the expected fallback string, no defensive logic needed.

### 3. Makefile

All build targets read `VERSION` and the current git hash and pass them as ldflags.

The Makefile requires `bash` (available via Git Bash on Windows):

```makefile
SHELL      := /bin/bash
VERSION    := $(shell cat VERSION 2>/dev/null || echo dev)
GIT_COMMIT := $(shell git rev-parse --short HEAD)
LDFLAGS    := -ldflags "-X main.version=$(VERSION) -X main.gitCommit=$(GIT_COMMIT)"

build:
	go build $(LDFLAGS) -o xan-argus ./cmd/server/

docker-build:
	VERSION=$(VERSION) GIT_COMMIT=$(GIT_COMMIT) docker compose up --build
```

If `VERSION` is absent, `cat` falls back to `dev` so the build never fails silently.

**Version bump targets** — each reads `VERSION`, increments the appropriate component with `awk`, writes it back, creates a git commit, and sets a git tag:

```makefile
bump-patch:
	@old=$$(cat VERSION); \
	new=$$(echo $$old | awk -F. '{print $$1"."$$2"."$$3+1}'); \
	echo $$new > VERSION; \
	git add VERSION; \
	git commit -m "chore: bump version to $$new"; \
	git tag v$$new; \
	echo "Bumped to $$new — run: git push && git push --tags"

bump-minor:
	@old=$$(cat VERSION); \
	new=$$(echo $$old | awk -F. '{print $$1"."$$2+1".0"}'); \
	echo $$new > VERSION; \
	git add VERSION; \
	git commit -m "chore: bump version to $$new"; \
	git tag v$$new; \
	echo "Bumped to $$new — run: git push && git push --tags"

bump-major:
	@old=$$(cat VERSION); \
	new=$$(echo $$old | awk -F. '{print $$1+1".0.0"}'); \
	echo $$new > VERSION; \
	git add VERSION; \
	git commit -m "chore: bump version to $$new"; \
	git tag v$$new; \
	echo "Bumped to $$new — run: git push && git push --tags"
```

After running `make bump-patch`:
- `VERSION` file is updated
- A commit `chore: bump version to 1.2.1` is created
- A git tag `v1.2.1` is set
- **Neither the commit nor the tag is pushed automatically** — run `git push && git push --tags` afterward

### 4. Dockerfile

The build stage receives version info via `ARG`:

```dockerfile
ARG VERSION=dev
ARG GIT_COMMIT=unknown
RUN go build -ldflags "-X main.version=${VERSION} -X main.gitCommit=${GIT_COMMIT}" -o /xan-argus ./cmd/server/
```

`docker-compose.yml` passes the args via env-var substitution with fallbacks:

```yaml
build:
  args:
    VERSION: ${VERSION:-dev}
    GIT_COMMIT: ${GIT_COMMIT:-unknown}
```

Always use `make docker-build` (defined in the Makefile above) to trigger Docker builds with version info. Direct `docker compose up --build` without env vars will produce a `dev-unknown` binary.

### 5. `TemplateEngine` — Global Version Injection

`TemplateEngine` in `internal/handler/template.go` gets a `Version string` field. `RenderPage` merges it into every page template data map automatically — no individual page handler needs to pass it. `RenderPartial` is **not** changed (partials are standalone fragments that never render `layout.html`).

```go
type TemplateEngine struct {
    // ...existing fields...
    Version string
}
```

`NewTemplateEngine` signature is **not** changed. After construction in `main.go`, the field is set directly:

```go
appVersion := "v" + version + "-" + gitCommit
tmpl.Version = appVersion
```

`RenderPage` uses a blind write — callers never supply a `"Version"` key, so no collision check is needed:

```go
func (t *TemplateEngine) RenderPage(w http.ResponseWriter, name string, data map[string]any) {
    if data == nil {
        data = map[string]any{}
    }
    data["Version"] = t.Version
    // ...existing render logic...
}
```

### 6. `layout.html` Footer

A `<footer>` element is added after `<main>`:

```html
<footer>
    <div class="container">
        <span class="text-muted">XAN-Argus {{.Version}}</span>
    </div>
</footer>
```

Uses existing CSS classes (`.container`, `.text-muted`). No new CSS needed.

### 7. `/health` Endpoint

The health response is extended to include the version:

```json
{"status": "ok", "version": "v1.2.0-a3f9d2c"}
```

## Data Flow

```
VERSION file + git rev-parse
        ↓
    Makefile / Dockerfile ARG
        ↓
    go build -ldflags
        ↓
    main.version + main.gitCommit (compiled in)
        ↓
    appVersion = "v" + version + "-" + gitCommit
        ↙               ↘
/health JSON        TemplateEngine.Version
                        ↓
                    layout.html footer
```

## Files Changed

| File | Change |
|------|--------|
| `VERSION` | New file — `1.0.0` |
| `Makefile` | New file — build, bump-patch, bump-minor, bump-major targets |
| `Dockerfile` | Add `ARG VERSION` / `ARG GIT_COMMIT`, update `go build` |
| `docker-compose.yml` | Pass build args |
| `cmd/server/main.go` | Add `version` / `gitCommit` vars, update `/health`, pass version to template engine |
| `internal/handler/template.go` | Add `Version` field, inject into all render calls |
| `web/templates/layout.html` | Add `<footer>` with `{{.Version}}` |

## Out of Scope

- Automatic version bumping in CI/CD
- Displaying version in API responses other than `/health`
- Version-based feature flags
