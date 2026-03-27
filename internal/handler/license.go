package handler

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/xan-com/xan-pythia/internal/model"
	"github.com/xan-com/xan-pythia/internal/repository"
)

type LicenseHandler struct {
	repo *repository.LicenseRepository
}

func NewLicenseHandler(repo *repository.LicenseRepository) *LicenseHandler {
	return &LicenseHandler{repo: repo}
}

func (h *LicenseHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/customers/{customerId}/licenses", h.listByCustomer)
	mux.HandleFunc("POST /api/v1/customers/{customerId}/licenses", h.create)
	mux.HandleFunc("GET /api/v1/licenses/{id}", h.get)
	mux.HandleFunc("PUT /api/v1/licenses/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/licenses/{id}", h.delete)
}

func (h *LicenseHandler) listByCustomer(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseUUID(r.PathValue("customerId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}
	params := paginationParams(r)
	licenses, err := h.repo.ListByCustomer(r.Context(), customerID, params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list licenses")
		return
	}
	writeJSON(w, http.StatusOK, licenses)
}

func (h *LicenseHandler) create(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseUUID(r.PathValue("customerId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}
	var input model.CreateLicenseInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input.CustomerID = customerID
	if input.ProductName == "" {
		writeError(w, http.StatusBadRequest, "product_name is required")
		return
	}
	if input.Quantity <= 0 {
		writeError(w, http.StatusBadRequest, "quantity must be greater than 0")
		return
	}
	license, err := h.repo.Create(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, license)
}

func (h *LicenseHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid license ID")
		return
	}
	license, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "license not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get license")
		return
	}
	writeJSON(w, http.StatusOK, license)
}

func (h *LicenseHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid license ID")
		return
	}
	var input model.UpdateLicenseInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	license, err := h.repo.Update(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "license not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update license")
		return
	}
	writeJSON(w, http.StatusOK, license)
}

func (h *LicenseHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid license ID")
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		if isFKViolation(err) {
			writeError(w, http.StatusConflict, "license has dependent records")
			return
		}
		writeError(w, http.StatusNotFound, "license not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
