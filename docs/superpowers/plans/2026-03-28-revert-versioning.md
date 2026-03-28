# Revert Versioning Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove all versioning functionality (VERSION file, Makefile, ldflags, build args, version display) — it's overengineered for an MVP running locally via Docker Compose.

**Architecture:** Pure removal/simplification. The `/health` endpoint stays but returns only `{"status":"ok"}`. No new functionality.

**Tech Stack:** Go, Docker, docker-compose

---

### Task 1: Delete VERSION file and Makefile

**Files:**
- Delete: `VERSION`
- Delete: `Makefile`

- [ ] **Step 1: Delete files**

```bash
rm VERSION Makefile
```

- [ ] **Step 2: Commit**

```bash
git add VERSION Makefile
git commit -m "chore: remove VERSION file and Makefile"
```

---

### Task 2: Simplify Dockerfile and docker-compose.yml

**Files:**
- Modify: `Dockerfile:6-8` — remove ARG lines, simplify build command
- Modify: `docker-compose.yml:21-23` — remove build args

- [ ] **Step 1: Simplify Dockerfile**

Replace lines 6-8:

```dockerfile
ARG VERSION=dev
ARG GIT_COMMIT=unknown
RUN CGO_ENABLED=0 go build -ldflags "-X main.version=${VERSION} -X main.gitCommit=${GIT_COMMIT}" -o /xan-argus ./cmd/server/
```

With:

```dockerfile
RUN CGO_ENABLED=0 go build -o /xan-argus ./cmd/server/
```

- [ ] **Step 2: Remove build args from docker-compose.yml**

Replace the `build:` section (lines 19-23):

```yaml
    build:
      context: .
      args:
        VERSION: ${VERSION:-dev}
        GIT_COMMIT: ${GIT_COMMIT:-unknown}
```

With:

```yaml
    build: .
```

- [ ] **Step 3: Verify Docker build works**

```bash
docker compose build app
```

Expected: Build succeeds without version args.

- [ ] **Step 4: Commit**

```bash
git add Dockerfile docker-compose.yml
git commit -m "chore: remove version build args from Docker"
```

---

### Task 3: Simplify main.go — remove version vars, simplify /health

**Files:**
- Modify: `cmd/server/main.go:2-4` — remove `fmt` from imports
- Modify: `cmd/server/main.go:16-19` — remove version/gitCommit vars
- Modify: `cmd/server/main.go:61-66` — simplify /health handler
- Modify: `cmd/server/main.go:73` — remove `tmpl.Version = appVersion`

- [ ] **Step 1: Remove version variables (lines 16-19)**

Remove:

```go
var (
	version   = "dev"
	gitCommit = "unknown"
)
```

- [ ] **Step 2: Simplify /health handler (lines 61-66)**

Replace:

```go
	appVersion := "v" + version + "-" + gitCommit
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","version":%q}`, appVersion)
	})
```

With:

```go
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
```

- [ ] **Step 3: Remove tmpl.Version line (line 73)**

Remove:

```go
	tmpl.Version = appVersion
```

- [ ] **Step 4: Remove `fmt` from imports if unused**

Check if `fmt` is still used elsewhere in main.go. If not, remove it from the import block.

- [ ] **Step 5: Run `go vet ./...`**

Expected: No errors.

- [ ] **Step 6: Commit**

```bash
git add cmd/server/main.go
git commit -m "chore: remove version vars and simplify /health endpoint"
```

---

### Task 4: Remove Version from TemplateEngine and layout

**Files:**
- Modify: `internal/handler/template.go:17` — remove `Version` field
- Modify: `internal/handler/template.go:121` — remove `data["Version"]` line
- Modify: `web/templates/layout.html:27-31` — remove footer

- [ ] **Step 1: Remove `Version` field from TemplateEngine struct (line 17)**

Remove:

```go
	Version  string                        // app version string, injected into every page render
```

So the struct becomes:

```go
type TemplateEngine struct {
	pages    map[string]*template.Template // page templates (layout + content)
	partials *template.Template            // standalone partials (rows, etc.)
}
```

- [ ] **Step 2: Remove `data["Version"]` from RenderPage (line 121)**

Remove:

```go
	data["Version"] = e.Version
```

- [ ] **Step 3: Remove version footer from layout.html (lines 27-31)**

Remove:

```html
    <footer>
        <div class="container">
            <span class="text-muted">XAN-Argus {{.Version}}</span>
        </div>
    </footer>
```

- [ ] **Step 4: Run `go vet ./...`**

Expected: No errors.

- [ ] **Step 5: Commit**

```bash
git add internal/handler/template.go web/templates/layout.html
git commit -m "chore: remove version display from templates and footer"
```

---

### Task 5: Archive versioning docs

**Files:**
- Move: `docs/superpowers/specs/2026-03-28-versioning-design.md` → `docs/superpowers/specs/archive/`
- Move: `docs/superpowers/plans/2026-03-28-versioning.md` → `docs/superpowers/plans/archive/`

- [ ] **Step 1: Move spec to archive**

```bash
mkdir -p docs/superpowers/specs/archive
mv docs/superpowers/specs/2026-03-28-versioning-design.md docs/superpowers/specs/archive/
```

- [ ] **Step 2: Move plan to archive**

```bash
mkdir -p docs/superpowers/plans/archive
mv docs/superpowers/plans/2026-03-28-versioning.md docs/superpowers/plans/archive/
```

- [ ] **Step 3: Commit**

```bash
git add docs/superpowers/
git commit -m "chore: archive versioning spec and plan"
```

---

### Task 6: Full stack verification

- [ ] **Step 1: Run `go vet ./...`**

Expected: No errors.

- [ ] **Step 2: Run `docker compose up --build`**

Expected: App starts, no version-related output.

- [ ] **Step 3: Test /health endpoint**

```bash
curl http://localhost:8080/health
```

Expected: `{"status":"ok"}` (no version field).

- [ ] **Step 4: Verify footer is gone**

Open `http://localhost:8080` in browser — no version text in footer.
