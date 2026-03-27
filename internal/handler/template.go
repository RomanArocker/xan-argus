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
	templates *template.Template
}

func NewTemplateEngine(templateDir string) (*TemplateEngine, error) {
	funcMap := template.FuncMap{
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
		"pgText": func(t pgtype.Text) string {
			if !t.Valid {
				return ""
			}
			return t.String
		},
		"uuidStr": func(u pgtype.UUID) string {
			if !u.Valid {
				return ""
			}
			b := u.Bytes
			return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
		},
		"lower": strings.ToLower,
		"default": func(def string, val string) string {
			if val == "" {
				return def
			}
			return val
		},
	}

	patterns := []string{
		filepath.Join(templateDir, "*.html"),
		filepath.Join(templateDir, "customers", "*.html"),
		filepath.Join(templateDir, "users", "*.html"),
		filepath.Join(templateDir, "services", "*.html"),
	}

	tmpl := template.New("").Funcs(funcMap)
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("globbing templates %s: %w", pattern, err)
		}
		if len(matches) > 0 {
			if _, err := tmpl.ParseFiles(matches...); err != nil {
				return nil, fmt.Errorf("parsing templates: %w", err)
			}
		}
	}

	return &TemplateEngine{templates: tmpl}, nil
}

func (e *TemplateEngine) Render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := e.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}
