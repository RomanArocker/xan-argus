# Semantic HTML Refactoring Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace generic `<div>` elements with appropriate HTML5 semantic tags across all Go templates to improve accessibility and document structure.

**Architecture:** Mechanical find-and-replace across 14 templates. CSS uses class selectors (`.page-header`, `.section`, `.search-bar`) so element changes require no CSS updates. JS uses `getElementById`/`querySelector` which are element-agnostic — no JS changes needed for structural replacements. JS-generated alert markup gets `role="alert"` added.

**Tech Stack:** HTML5, Go templates, CSS, HTMX

---

### Replacement Map

| Current | Replacement | Reason |
|---|---|---|
| `<div class="page-header">` | `<header class="page-header">` | Introductory content for a section |
| `<div class="section">` | `<section class="section">` | Thematic grouping of content |
| `<div class="search-bar">` | `<search class="search-bar">` | HTML5 `<search>` element (Chrome 118+, FF 118+, Safari 17+) |
| `<div id="form-message">` | `<div id="form-message" aria-live="polite">` | Announces form feedback to screen readers |
| `<div id="form-msg-*">` | `<div id="form-msg-*" aria-live="polite">` | Same as above for inline form messages |
| JS: `'<div class="alert alert-error">'` | JS: `'<div class="alert alert-error" role="alert">'` | Urgent feedback announced immediately by screen readers |

**Not changed** (correctly generic):
- `<div class="container">` — layout wrapper
- `<div class="card">` — visual styling container
- `<div class="flex gap-1">` — layout utility
- `<div class="form-group">` — form layout grouping

---

### Task 1: Update CSS for new elements

**Files:**
- Modify: `web/static/css/style.css`

- [ ] **Step 1: Add element resets for `<header>`, `<section>`, `<search>`**

These elements have default browser styles. Add resets so they behave like `<div>`:

```css
header, section, search {
  display: block;
}
```

Add this right after the `* { box-sizing: border-box; ... }` rule.

- [ ] **Step 2: Verify no element-specific selectors exist**

Confirm that all selectors in `style.css` use class names (`.page-header`, `.section`, `.search-bar`), not element-qualified selectors like `div.page-header`. This is already the case — no changes needed, just verify.

- [ ] **Step 3: Commit**

```bash
git add web/static/css/style.css
git commit -m "refactor: add element resets for semantic HTML tags"
```

---

### Task 2: Update list templates — `<header>` and `<search>`

**Files:**
- Modify: `web/templates/customers/list.html`
- Modify: `web/templates/users/list.html`
- Modify: `web/templates/services/list.html`
- Modify: `web/templates/categories/list.html`

- [ ] **Step 1: Replace `<div class="page-header">` with `<header class="page-header">` in all 4 list templates**

In each file, change:
```html
<div class="page-header">
    <h1>...</h1>
    <a href="..." class="btn btn-primary">+ New ...</a>
</div>
```
to:
```html
<header class="page-header">
    <h1>...</h1>
    <a href="..." class="btn btn-primary">+ New ...</a>
</header>
```

- [ ] **Step 2: Replace `<div class="search-bar">` with `<search class="search-bar">` in customers, users, and services list templates**

In `customers/list.html`, `users/list.html`, `services/list.html`, change:
```html
<div class="search-bar">
```
to:
```html
<search class="search-bar">
```
And the corresponding closing `</div>` to `</search>`.

Note: `categories/list.html` has no search bar.

- [ ] **Step 3: Commit**

```bash
git add web/templates/customers/list.html web/templates/users/list.html web/templates/services/list.html web/templates/categories/list.html
git commit -m "refactor: use semantic header and search elements in list templates"
```

---

### Task 3: Update form templates — `<header>`, `aria-live`, `role="alert"`

**Files:**
- Modify: `web/templates/customers/form.html`
- Modify: `web/templates/users/form.html`
- Modify: `web/templates/services/form.html`
- Modify: `web/templates/categories/form.html`

- [ ] **Step 1: Replace `<div class="page-header">` with `<header class="page-header">` in all 4 form templates**

Same pattern as list templates — change opening `<div class="page-header">` to `<header class="page-header">` and closing `</div>` to `</header>`.

- [ ] **Step 2: Add `aria-live="polite"` to form message containers**

In each form template, change:
```html
<div id="form-message"></div>
```
to:
```html
<div id="form-message" aria-live="polite"></div>
```

In `categories/form.html`, also change:
```html
<div id="form-msg-field"></div>
```
to:
```html
<div id="form-msg-field" aria-live="polite"></div>
```

- [ ] **Step 3: Add `role="alert"` to JS-generated error alerts**

In each form template's `<script>` block, the responseError handler generates alert HTML. Change all instances of:
```javascript
'<div class="alert alert-error">' + msg + '</div>'
```
to:
```javascript
'<div class="alert alert-error" role="alert">' + msg + '</div>'
```

Files affected: `customers/form.html`, `users/form.html`, `services/form.html`, `categories/form.html` (2 instances in categories — one for category-form, one for add-field-form).

- [ ] **Step 4: Commit**

```bash
git add web/templates/customers/form.html web/templates/users/form.html web/templates/services/form.html web/templates/categories/form.html
git commit -m "refactor: use semantic header in forms, add aria-live and role=alert for accessibility"
```

---

### Task 4: Update detail template — `<header>`, `<section>`, `aria-live`, `role="alert"`

**Files:**
- Modify: `web/templates/customers/detail.html`

This is the most complex template with multiple sections.

- [ ] **Step 1: Replace the main `<div class="page-header">` with `<header class="page-header">`**

Change:
```html
<div class="page-header">
    <h1>{{.Customer.Name}}</h1>
    <div class="flex gap-1">
        <a href="/customers/{{uuidStr .Customer.ID}}/edit" class="btn btn-primary">Edit</a>
    </div>
</div>
```
to:
```html
<header class="page-header">
    <h1>{{.Customer.Name}}</h1>
    <div class="flex gap-1">
        <a href="/customers/{{uuidStr .Customer.ID}}/edit" class="btn btn-primary">Edit</a>
    </div>
</header>
```

- [ ] **Step 2: Replace all 4 `<div class="section">` with `<section class="section">`**

The 4 sections are: Assets, Licenses, Assigned Users, Subscribed Services. For each, change:
```html
<div class="section">
```
to:
```html
<section class="section">
```
And the corresponding closing `</div>` to `</section>`.

- [ ] **Step 3: Replace sub-section `<div class="page-header">` with `<header class="page-header">`**

Inside each `<section>`, these are single-line elements. Change all 4:
```html
<div class="page-header"><h2>Assets</h2></div>
<div class="page-header"><h2>Licenses</h2></div>
<div class="page-header"><h2>Assigned Users</h2></div>
<div class="page-header"><h2>Subscribed Services</h2></div>
```
to:
```html
<header class="page-header"><h2>Assets</h2></header>
<header class="page-header"><h2>Licenses</h2></header>
<header class="page-header"><h2>Assigned Users</h2></header>
<header class="page-header"><h2>Subscribed Services</h2></header>
```

- [ ] **Step 4: Add `aria-live="polite"` to all inline form message containers**

Change these 4 divs:
```html
<div id="form-msg-asset"></div>
<div id="form-msg-license"></div>
<div id="form-msg-assign"></div>
<div id="form-msg-cs"></div>
```
to:
```html
<div id="form-msg-asset" aria-live="polite"></div>
<div id="form-msg-license" aria-live="polite"></div>
<div id="form-msg-assign" aria-live="polite"></div>
<div id="form-msg-cs" aria-live="polite"></div>
```

- [ ] **Step 5: Add `role="alert"` to JS-generated error alert in the responseError handler**

In the `<script>` block at the bottom, change:
```javascript
'<div class="alert alert-error">' + msg + '</div>'
```
to:
```javascript
'<div class="alert alert-error" role="alert">' + msg + '</div>'
```

- [ ] **Step 6: Commit**

```bash
git add web/templates/customers/detail.html
git commit -m "refactor: use semantic section/header in customer detail, add aria attributes"
```

---

### Task 5: Update categories/form.html section

**Files:**
- Modify: `web/templates/categories/form.html`

- [ ] **Step 1: Replace `<div class="section mt-1">` with `<section class="section mt-1">`**

Change:
```html
<div class="section mt-1">
```
to:
```html
<section class="section mt-1">
```
And the corresponding closing `</div>` to `</section>`.

- [ ] **Step 2: Replace sub-section `<div class="page-header">` with `<header class="page-header">`**

Change:
```html
<div class="page-header"><h2>Field Definitions</h2></div>
```
to:
```html
<header class="page-header"><h2>Field Definitions</h2></header>
```

- [ ] **Step 3: Commit**

```bash
git add web/templates/categories/form.html
git commit -m "refactor: use semantic section/header in category field definitions"
```

---

### Task 6: Visual verification

- [ ] **Step 1: Build and run the application**

```bash
docker compose up --build
```

- [ ] **Step 2: Verify each page renders correctly**

Visit each page and confirm no visual regressions:
- `/customers` — list with search bar
- `/customers/new` — form
- `/customers/{id}` — detail with all 4 sections
- `/users` — list with search/filter
- `/users/new` — form
- `/services` — list with search
- `/services/new` — form
- `/categories` — list
- `/categories/new` — form
- `/categories/{id}/edit` — form with field definitions section

- [ ] **Step 3: Validate HTML**

Open browser DevTools on a few pages and confirm no HTML parsing errors or warnings. Check that `<search>`, `<header>`, `<section>` elements appear correctly in the DOM tree.
