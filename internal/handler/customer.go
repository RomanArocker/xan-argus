package handler

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/xan-com/xan-pythia/internal/model"
	"github.com/xan-com/xan-pythia/internal/repository"
)

type CustomerHandler struct {
	repo *repository.CustomerRepository
}

func NewCustomerHandler(repo *repository.CustomerRepository) *CustomerHandler {
	return &CustomerHandler{repo: repo}
}

func (h *CustomerHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/customers", h.list)
	mux.HandleFunc("POST /api/v1/customers", h.create)
	mux.HandleFunc("GET /api/v1/customers/{id}", h.get)
	mux.HandleFunc("PUT /api/v1/customers/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/customers/{id}", h.delete)
}

func (h *CustomerHandler) list(w http.ResponseWriter, r *http.Request) {
	params := paginationParams(r)
	customers, err := h.repo.List(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list customers")
		return
	}
	writeJSON(w, http.StatusOK, customers)
}

func (h *CustomerHandler) create(w http.ResponseWriter, r *http.Request) {
	var input model.CreateCustomerInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	customer, err := h.repo.Create(r.Context(), input)
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "customer with this name already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create customer")
		return
	}
	writeJSON(w, http.StatusCreated, customer)
}

func (h *CustomerHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}
	customer, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "customer not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get customer")
		return
	}
	writeJSON(w, http.StatusOK, customer)
}

func (h *CustomerHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}
	var input model.UpdateCustomerInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	customer, err := h.repo.Update(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "customer not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update customer")
		return
	}
	writeJSON(w, http.StatusOK, customer)
}

func (h *CustomerHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		if isFKViolation(err) {
			writeError(w, http.StatusConflict, "customer has dependent records")
			return
		}
		writeError(w, http.StatusNotFound, "customer not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
