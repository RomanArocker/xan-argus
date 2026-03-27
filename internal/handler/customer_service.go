package handler

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/xan-com/xan-argus/internal/model"
	"github.com/xan-com/xan-argus/internal/repository"
)

type CustomerServiceHandler struct {
	repo *repository.CustomerServiceRepository
}

func NewCustomerServiceHandler(repo *repository.CustomerServiceRepository) *CustomerServiceHandler {
	return &CustomerServiceHandler{repo: repo}
}

func (h *CustomerServiceHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/customers/{customerId}/services", h.listByCustomer)
	mux.HandleFunc("POST /api/v1/customers/{customerId}/services", h.create)
	mux.HandleFunc("GET /api/v1/customer-services/{id}", h.get)
	mux.HandleFunc("PUT /api/v1/customer-services/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/customer-services/{id}", h.delete)
}

func (h *CustomerServiceHandler) listByCustomer(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseUUID(r.PathValue("customerId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}
	params := paginationParams(r)
	services, err := h.repo.ListByCustomer(r.Context(), customerID, params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list customer services")
		return
	}
	writeJSON(w, http.StatusOK, services)
}

func (h *CustomerServiceHandler) create(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseUUID(r.PathValue("customerId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}
	var input model.CreateCustomerServiceInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input.CustomerID = customerID
	if !input.ServiceID.Valid {
		writeError(w, http.StatusBadRequest, "service_id is required")
		return
	}
	cs, err := h.repo.Create(r.Context(), input)
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "customer already subscribed to this service")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create customer service")
		return
	}
	writeJSON(w, http.StatusCreated, cs)
}

func (h *CustomerServiceHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer service ID")
		return
	}
	cs, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "customer service not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get customer service")
		return
	}
	writeJSON(w, http.StatusOK, cs)
}

func (h *CustomerServiceHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer service ID")
		return
	}
	var input model.UpdateCustomerServiceInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	cs, err := h.repo.Update(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "customer service not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update customer service")
		return
	}
	writeJSON(w, http.StatusOK, cs)
}

func (h *CustomerServiceHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer service ID")
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		if isFKViolation(err) {
			writeError(w, http.StatusConflict, "customer service has dependent records")
			return
		}
		writeError(w, http.StatusNotFound, "customer service not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
