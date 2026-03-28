# Versioning Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Embed a semantic version + git hash into the XAN-Argus binary at build time and display it in the web UI footer and `/health` endpoint.

**Architecture:** A `VERSION` file holds the semver (e.g. `1.0.0`). A `Makefile` reads it and passes it via `go build -ldflags` so the values are compiled into the binary. The `TemplateEngine` carries the version string and injects it into every page render; `layout.html` shows it in a footer.

**Tech Stack:** Go 1.26, GNU Make + bash (Git Bash on Windows), Docker Compose, Go HTML templates.

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `VERSION` | Create | Semver source of truth (`1.0.0`) |
| `Makefile` | Create | `build`, `docker-build`, `bump-patch/minor/major` |
| `Dockerfile` | Modify | Accept `ARG VERSION` / `ARG GIT_COMMIT`, pass to `go build` |
| `docker-compose.yml` | Modify | Pass `VERSION` / `GIT_COMMIT` build args |
| `cmd/server/main.go` | Modify | `var version`, `var gitCommit`, composite string, `/health` response, `tmpl.Version` |
| `internal/handler/template.go` | Modify | `Version string` field on `TemplateEngine`, inject in `RenderPage` |
| `web/templates/layout.html` | Modify | `<footer>` with `{{.Version}}` |

---

## Task 1: VERSION file and Makefile

**Files:**
- Create: `VERSION`
- Create: `Makefile`

- [ ] **Step 1: Create `VERSION` file**

```
1.0.0
```

File must contain exactly `1.0.0` with no trailing newline issues (a trailing newline is fine, leading whitespace is not).

- [ ] **Step 2: Create `Makefile`**

```makefile
SHELL      := /bin/bash
VERSION    := $(shell cat VERSION 2>/dev/null || echo dev)
GIT_COMMIT := $(shell git rev-parse --short HEAD)
LDFLAGS    := -ldflags "-X main.version=$(VERSION) -X main.gitCommit=$(GIT_COMMIT)"

.PHONY: build docker-build bump-patch bump-minor bump-major

build:
	go build $(LDFLAGS) -o xan-argus ./cmd/server/

docker-build:
	VERSION=$(VERSION) GIT_COMMIT=$(GIT_COMMIT) docker compose up --build

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

- [ ] **Step 3: Verify `make build` works**

Run in Git Bash:
```bash
make build
```
Expected: binary `xan-argus` created, no errors. The ldflags line in the output should contain the version and a short git hash.

- [ ] **Step 4: Commit**

```bash
git add VERSION Makefile
git commit -m "feat: add VERSION file and Makefile with build and bump targets"
```

---

## Task 2: Docker integration

**Files:**
- Modify: `Dockerfile`
- Modify: `docker-compose.yml`

- [ ] **Step 1: Read both files before editing**

Read `Dockerfile` and `docker-compose.yml` in full before making any changes.

- [ ] **Step 2: Update `Dockerfile` builder stage**

Find this line in `Dockerfile`:
```dockerfile
RUN CGO_ENABLED=0 go build -o /xan-argus ./cmd/server/
```

Replace with:
```dockerfile
ARG VERSION=dev
ARG GIT_COMMIT=unknown
RUN CGO_ENABLED=0 go build -ldflags "-X main.version=${VERSION} -X main.gitCommit=${GIT_COMMIT}" -o /xan-argus ./cmd/server/
```

The two `ARG` lines must appear **before** the `RUN go build` line and **after** the `COPY . .` line.

- [ ] **Step 3: Update `docker-compose.yml` app service**

Add a `build` section with args to the `app` service. The current `app` service has `build: .` — replace that with:

```yaml
    build:
      context: .
      args:
        VERSION: ${VERSION:-dev}
        GIT_COMMIT: ${GIT_COMMIT:-unknown}
```

(Keep all other fields — `ports`, `environment`, `depends_on` — unchanged.)

- [ ] **Step 4: Verify Docker build passes without env vars (fallback)**

```bash
docker compose build app
```
Expected: builds successfully, no errors. The binary will contain `vdev-unknown` (expected fallback).

- [ ] **Step 5: Commit**

```bash
git add Dockerfile docker-compose.yml
git commit -m "feat: pass VERSION and GIT_COMMIT build args to Docker"
```

---

## Task 3: Version variables in main.go

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Read `cmd/server/main.go` before editing**

- [ ] **Step 2: Add package-level variables**

Add these two lines immediately after `package main`:

```go
var (
	version   = "dev"
	gitCommit = "unknown"
)
```

These are the ldflags injection targets. Do **not** add a `const` — they must be `var` for ldflags to override them.

- [ ] **Step 3: Update `/health` endpoint**

Find the current health handler:
```go
mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"status":"ok"}`))
})
```

Replace with:
```go
appVersion := "v" + version + "-" + gitCommit
mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    fmt.Fprintf(w, `{"status":"ok","version":%q}`, appVersion)
})
```

Add `"fmt"` to the import block if not already present.

- [ ] **Step 4: Set version on template engine**

Find this line (it appears after `tmpl` is created):
```go
handler.NewPageHandler(tmpl, ...)
```

Insert **before** that line:
```go
tmpl.Version = appVersion
```

- [ ] **Step 5: Verify the server compiles**

```bash
go vet ./...
```
Expected: no errors.

- [ ] **Step 6: Verify `/health` response manually**

```bash
make build && ./xan-argus &
# wait 1s then:
curl http://localhost:8080/health
```
Expected output contains: `"version":"v1.0.0-<hash>"` (hash is the current short git hash).

Kill the background process after verifying.

- [ ] **Step 7: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: embed version and git hash into binary, expose in /health"
```

---

## Task 4: TemplateEngine version injection

**Files:**
- Modify: `internal/handler/template.go`

- [ ] **Step 1: Read `internal/handler/template.go` before editing**

- [ ] **Step 2: Add `Version` field to `TemplateEngine` struct**

Find:
```go
type TemplateEngine struct {
	pages    map[string]*template.Template // page templates (layout + content)
	partials *template.Template            // standalone partials (rows, etc.)
}
```

Replace with:
```go
type TemplateEngine struct {
	pages    map[string]*template.Template // page templates (layout + content)
	partials *template.Template            // standalone partials (rows, etc.)
	Version  string                        // app version string, injected into every page render
}
```

- [ ] **Step 3: Update `RenderPage` signature and inject version**

Find the current `RenderPage` method:
```go
func (e *TemplateEngine) RenderPage(w http.ResponseWriter, page string, data any) {
	t, ok := e.pages[page]
	if !ok {
		http.Error(w, "page not found: "+page, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "layout", data); err != nil {
```

Change the `data any` parameter to `data map[string]any`, add the nil guard and version injection before the `ExecuteTemplate` call:

```go
func (e *TemplateEngine) RenderPage(w http.ResponseWriter, page string, data map[string]any) {
	t, ok := e.pages[page]
	if !ok {
		http.Error(w, "page not found: "+page, http.StatusInternalServerError)
		return
	}
	if data == nil {
		data = map[string]any{}
	}
	data["Version"] = e.Version
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "layout", data); err != nil {
```

Keep the rest of the method body unchanged.

- [ ] **Step 4: Verify the project compiles**

```bash
go vet ./...
```
Expected: no errors. All page handler callers already pass `map[string]any`, so no callers need updating.

- [ ] **Step 5: Commit**

```bash
git add internal/handler/template.go
git commit -m "feat: inject version string into all page templates via TemplateEngine"
```

---

## Task 5: Footer in layout.html

**Files:**
- Modify: `web/templates/layout.html`

- [ ] **Step 1: Read `web/templates/layout.html` before editing**

- [ ] **Step 2: Add footer after `</main>`**

Find:
```html
    </main>
</body>
```

Replace with:
```html
    </main>
    <footer>
        <div class="container">
            <span class="text-muted">XAN-Argus {{.Version}}</span>
        </div>
    </footer>
</body>
```

No new CSS needed — `.container` and `.text-muted` are already defined in `web/static/css/style.css`.

- [ ] **Step 3: Add footer styling to style.css**

The existing `.container` and `.text-muted` classes style the footer content, but the `<footer>` element itself needs minimal spacing. Check if there are existing footer styles in `web/static/css/style.css`. If not, add at the end of the file:

```css
footer {
    margin-top: 2rem;
    padding: 1rem 0;
    border-top: 1px solid var(--border);
    text-align: center;
    font-size: 0.875rem;
}
```

- [ ] **Step 4: Start the server and verify the footer appears**

```bash
make build
./xan-argus &
```

Open `http://localhost:8080/customers` in a browser. Verify:
- Footer is visible at the bottom of the page
- It displays `XAN-Argus v1.0.0-<hash>` (or `vdev-unknown` if built without Makefile)
- Layout is not broken

Kill the background process after verifying.

- [ ] **Step 5: Commit**

```bash
git add web/templates/layout.html web/static/css/style.css
git commit -m "feat: add version footer to layout"
```

---

## Task 6: Smoke test and final verification

- [ ] **Step 1: Run static analysis**

```bash
go vet ./...
golangci-lint run ./...
```
Expected: no errors or warnings.

- [ ] **Step 2: Verify full Docker build with version**

```bash
make docker-build
```

Expected: Docker image builds and container starts. Check logs for `Starting server on :8080`.

- [ ] **Step 3: Verify `/health` in Docker**

```bash
curl http://localhost:8080/health
```
Expected: `{"status":"ok","version":"v1.0.0-<hash>"}` where hash is a real git hash (not `unknown`).

- [ ] **Step 4: Verify footer in browser**

Open `http://localhost:8080`. Footer should show `XAN-Argus v1.0.0-<hash>`.

- [ ] **Step 5: Test bump-patch**

```bash
make bump-patch
cat VERSION
git log --oneline -3
git tag -l
```
Expected:
- `VERSION` contains `1.0.1`
- Latest commit message is `chore: bump version to 1.0.1`
- Tag `v1.0.1` exists

Then reset for cleanliness (optional — only if you don't want to keep the bumped version):
```bash
git tag -d v1.0.1
git reset --soft HEAD~1
echo "1.0.0" > VERSION
git add VERSION
git commit -m "chore: reset version to 1.0.0 after bump test"
```

- [ ] **Step 6: Final commit (if any loose changes)**

```bash
git status
# commit anything uncommitted
```
