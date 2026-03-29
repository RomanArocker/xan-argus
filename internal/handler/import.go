package handler

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/xan-com/xan-argus/internal/importer"
)

type ImportHandler struct {
	engine   *importer.Engine
	exporter *importer.Exporter
	registry *importer.Registry
}

func NewImportHandler(engine *importer.Engine, exporter *importer.Exporter, registry *importer.Registry) *ImportHandler {
	return &ImportHandler{engine: engine, exporter: exporter, registry: registry}
}

func (h *ImportHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/import/{entity}", h.importCSV)
	mux.HandleFunc("GET /api/v1/import/{entity}/template", h.template)
	mux.HandleFunc("GET /api/v1/export/{entity}", h.export)
}

func (h *ImportHandler) importCSV(w http.ResponseWriter, r *http.Request) {
	entityName := r.PathValue("entity")
	if _, err := h.registry.Get(entityName); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("unknown entity: %s", entityName))
		return
	}
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing 'file' field")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read file")
		return
	}
	result, err := h.engine.Import(r.Context(), entityName, data)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("import failed: %v", err))
		return
	}
	if len(result.Errors) > 0 {
		writeJSON(w, http.StatusBadRequest, result)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *ImportHandler) template(w http.ResponseWriter, r *http.Request) {
	entityName := r.PathValue("entity")
	cfg, err := h.registry.Get(entityName)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("unknown entity: %s", entityName))
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s_template.csv"`, entityName))
	if err := h.exporter.WriteTemplate(w, cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate template")
	}
}

func (h *ImportHandler) export(w http.ResponseWriter, r *http.Request) {
	entityName := r.PathValue("entity")
	if _, err := h.registry.Get(entityName); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("unknown entity: %s", entityName))
		return
	}
	filename := fmt.Sprintf("%s_%s.csv", entityName, time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	if err := h.exporter.Export(r.Context(), w, entityName); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("export failed: %v", err))
	}
}
