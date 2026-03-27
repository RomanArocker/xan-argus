package handler

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/xan-com/xan-argus/internal/model"
	"github.com/xan-com/xan-argus/internal/repository"
)

type HardwareCategoryHandler struct {
	repo *repository.HardwareCategoryRepository
}

func NewHardwareCategoryHandler(repo *repository.HardwareCategoryRepository) *HardwareCategoryHandler {
	return &HardwareCategoryHandler{repo: repo}
}

func (h *HardwareCategoryHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/hardware-categories", h.list)
	mux.HandleFunc("POST /api/v1/hardware-categories", h.create)
	mux.HandleFunc("GET /api/v1/hardware-categories/{id}", h.get)
	mux.HandleFunc("PUT /api/v1/hardware-categories/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/hardware-categories/{id}", h.delete)
	mux.HandleFunc("POST /api/v1/hardware-categories/{id}/fields", h.createField)
	mux.HandleFunc("PUT /api/v1/hardware-categories/{id}/fields/{fieldId}", h.updateField)
	mux.HandleFunc("DELETE /api/v1/hardware-categories/{id}/fields/{fieldId}", h.deleteField)
}

// --- Category CRUD ---

func (h *HardwareCategoryHandler) list(w http.ResponseWriter, r *http.Request) {
	categories, err := h.repo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list categories")
		return
	}
	writeJSON(w, http.StatusOK, categories)
}

func (h *HardwareCategoryHandler) create(w http.ResponseWriter, r *http.Request) {
	var input model.CreateHardwareCategoryInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	category, err := h.repo.Create(r.Context(), input)
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "category name already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create category")
		return
	}
	writeJSON(w, http.StatusCreated, category)
}

func (h *HardwareCategoryHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category ID")
		return
	}
	category, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "category not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get category")
		return
	}
	writeJSON(w, http.StatusOK, category)
}

func (h *HardwareCategoryHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category ID")
		return
	}
	var input model.UpdateHardwareCategoryInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	category, err := h.repo.Update(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "category not found")
			return
		}
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "category name already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update category")
		return
	}
	writeJSON(w, http.StatusOK, category)
}

func (h *HardwareCategoryHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category ID")
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "category not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete category")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Field Definition CRUD ---

func (h *HardwareCategoryHandler) createField(w http.ResponseWriter, r *http.Request) {
	categoryID, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category ID")
		return
	}
	var input model.CreateFieldDefinitionInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input.CategoryID = categoryID
	if input.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	validTypes := map[string]bool{"text": true, "number": true, "date": true, "boolean": true}
	if !validTypes[input.FieldType] {
		writeError(w, http.StatusBadRequest, "field_type must be 'text', 'number', 'date', or 'boolean'")
		return
	}
	field, err := h.repo.CreateField(r.Context(), input)
	if err != nil {
		if isFKViolation(err) {
			writeError(w, http.StatusNotFound, "category not found")
			return
		}
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "field name already exists in this category")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create field definition")
		return
	}
	writeJSON(w, http.StatusCreated, field)
}

func (h *HardwareCategoryHandler) updateField(w http.ResponseWriter, r *http.Request) {
	fieldID, err := parseUUID(r.PathValue("fieldId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid field ID")
		return
	}
	var input model.UpdateFieldDefinitionInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	field, err := h.repo.UpdateField(r.Context(), fieldID, input)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "field definition not found")
			return
		}
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "field name already exists in this category")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update field definition")
		return
	}
	writeJSON(w, http.StatusOK, field)
}

func (h *HardwareCategoryHandler) deleteField(w http.ResponseWriter, r *http.Request) {
	fieldID, err := parseUUID(r.PathValue("fieldId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid field ID")
		return
	}
	if err := h.repo.DeleteField(r.Context(), fieldID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "field definition not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete field definition")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
