package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type TemplateEngine struct {
	pages    map[string]*template.Template // page templates (layout + content)
	partials *template.Template            // standalone partials (rows, etc.)
}

func newFuncMap() template.FuncMap {
	return template.FuncMap{
		"formatDate": func(t time.Time) string {
			return t.Format("02.01.2006")
		},
		"formatDateTime": func(t time.Time) string {
			return t.Format("02.01.2006 15:04")
		},
		"formatPgDate": func(d pgtype.Date) string {
			if !d.Valid {
				return "—"
			}
			return d.Time.Format("02.01.2006")
		},
		"formatInputDate": func(d pgtype.Date) string {
			if !d.Valid {
				return ""
			}
			return d.Time.Format("2006-01-02")
		},
		"pgText": func(t pgtype.Text) string {
			if !t.Valid {
				return ""
			}
			return t.String
		},
		"uuidStr": uuidToStr,
		"mapGet": func(m map[string]string, key string) string {
			if v, ok := m[key]; ok {
				return v
			}
			return ""
		},
		"lower": strings.ToLower,
		"default": func(def string, val string) string {
			if val == "" {
				return def
			}
			return val
		},
	}
}

func NewTemplateEngine(templateDir string) (*TemplateEngine, error) {
	layoutFile := filepath.Join(templateDir, "layout.html")
	pages := make(map[string]*template.Template)

	// Collect all partial files (list_rows.html) — these define named templates
	// that page templates reference via {{template "customer_rows" ...}}
	partialFiles := []string{
		filepath.Join(templateDir, "customers", "list_rows.html"),
		filepath.Join(templateDir, "users", "list_rows.html"),
		filepath.Join(templateDir, "services", "list_rows.html"),
		filepath.Join(templateDir, "categories", "list_rows.html"),
		filepath.Join(templateDir, "assets", "fields_partial.html"),
	}

	// Page templates: each gets layout + all partials + its own content
	// This avoids "content" name collisions between pages while giving
	// each page access to all partial templates it might reference
	pageFiles := []string{
		filepath.Join(templateDir, "customers", "list.html"),
		filepath.Join(templateDir, "customers", "detail.html"),
		filepath.Join(templateDir, "customers", "form.html"),
		filepath.Join(templateDir, "customers", "asset_detail.html"),
		filepath.Join(templateDir, "users", "list.html"),
		filepath.Join(templateDir, "users", "form.html"),
		filepath.Join(templateDir, "services", "list.html"),
		filepath.Join(templateDir, "services", "form.html"),
		filepath.Join(templateDir, "categories", "list.html"),
		filepath.Join(templateDir, "categories", "form.html"),
		filepath.Join(templateDir, "assets", "form.html"),
		filepath.Join(templateDir, "licenses", "detail.html"),
		filepath.Join(templateDir, "licenses", "form.html"),
	}

	for _, pf := range pageFiles {
		// Parse layout + all partials + this page's content template
		files := append([]string{layoutFile}, partialFiles...)
		files = append(files, pf)
		t, err := template.New("").Funcs(newFuncMap()).ParseFiles(files...)
		if err != nil {
			return nil, fmt.Errorf("parsing page template %s: %w", pf, err)
		}
		name := filepath.Base(filepath.Dir(pf)) + "/" + strings.TrimSuffix(filepath.Base(pf), ".html")
		pages[name] = t
	}

	// Standalone partials for HTMX row responses (no layout wrapper)
	partials := template.New("").Funcs(newFuncMap())
	for _, pf := range partialFiles {
		if _, err := partials.ParseFiles(pf); err != nil {
			return nil, fmt.Errorf("parsing partial %s: %w", pf, err)
		}
	}

	return &TemplateEngine{pages: pages, partials: partials}, nil
}

// RenderPage renders a page template (layout + content) by page key (e.g., "customers/list")
func (e *TemplateEngine) RenderPage(w http.ResponseWriter, page string, data map[string]any) {
	t, ok := e.pages[page]
	if !ok {
		http.Error(w, "page not found: "+page, http.StatusInternalServerError)
		return
	}
	if data == nil {
		data = map[string]any{}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// RenderPartial renders a standalone partial template (e.g., "customer_rows")
func (e *TemplateEngine) RenderPartial(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := e.partials.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}
