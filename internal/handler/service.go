package handler

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/xan-com/xan-argus/internal/model"
	"github.com/xan-com/xan-argus/internal/repository"
)

type ServiceHandler struct {
	repo *repository.ServiceRepository
}

func NewServiceHandler(repo *repository.ServiceRepository) *ServiceHandler {
	return &ServiceHandler{repo: repo}
}

func (h *ServiceHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/services", h.list)
	mux.HandleFunc("POST /api/v1/services", h.create)
	mux.HandleFunc("GET /api/v1/services/{id}", h.get)
	mux.HandleFunc("PUT /api/v1/services/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/services/{id}", h.delete)
}

func (h *ServiceHandler) list(w http.ResponseWriter, r *http.Request) {
	params := paginationParams(r)
	services, err := h.repo.List(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list services")
		return
	}
	writeJSON(w, http.StatusOK, services)
}

func (h *ServiceHandler) create(w http.ResponseWriter, r *http.Request) {
	var input model.CreateServiceInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	svc, err := h.repo.Create(r.Context(), input)
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "service with this name already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create service")
		return
	}
	writeJSON(w, http.StatusCreated, svc)
}

func (h *ServiceHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid service ID")
		return
	}
	svc, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "service not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get service")
		return
	}
	writeJSON(w, http.StatusOK, svc)
}

func (h *ServiceHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid service ID")
		return
	}
	var input model.UpdateServiceInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	svc, err := h.repo.Update(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "service not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update service")
		return
	}
	writeJSON(w, http.StatusOK, svc)
}

func (h *ServiceHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid service ID")
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		if isFKViolation(err) {
			writeError(w, http.StatusConflict, "service has dependent records")
			return
		}
		writeError(w, http.StatusNotFound, "service not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
