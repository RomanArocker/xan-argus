package handler

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/xan-com/xan-pythia/internal/model"
	"github.com/xan-com/xan-pythia/internal/repository"
)

type UserAssignmentHandler struct {
	repo *repository.UserAssignmentRepository
}

func NewUserAssignmentHandler(repo *repository.UserAssignmentRepository) *UserAssignmentHandler {
	return &UserAssignmentHandler{repo: repo}
}

func (h *UserAssignmentHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/customers/{customerId}/user-assignments", h.listByCustomer)
	mux.HandleFunc("POST /api/v1/customers/{customerId}/user-assignments", h.create)
	mux.HandleFunc("GET /api/v1/user-assignments/{id}", h.get)
	mux.HandleFunc("PUT /api/v1/user-assignments/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/user-assignments/{id}", h.delete)
}

func (h *UserAssignmentHandler) listByCustomer(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseUUID(r.PathValue("customerId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}
	params := paginationParams(r)
	assignments, err := h.repo.ListByCustomer(r.Context(), customerID, params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list user assignments")
		return
	}
	writeJSON(w, http.StatusOK, assignments)
}

func (h *UserAssignmentHandler) create(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseUUID(r.PathValue("customerId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}
	var input model.CreateUserAssignmentInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input.CustomerID = customerID
	if !input.UserID.Valid {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	if input.Role == "" {
		writeError(w, http.StatusBadRequest, "role is required")
		return
	}
	assignment, err := h.repo.Create(r.Context(), input)
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "user is already assigned to this customer")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create user assignment")
		return
	}
	writeJSON(w, http.StatusCreated, assignment)
}

func (h *UserAssignmentHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user assignment ID")
		return
	}
	assignment, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "user assignment not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get user assignment")
		return
	}
	writeJSON(w, http.StatusOK, assignment)
}

func (h *UserAssignmentHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user assignment ID")
		return
	}
	var input model.UpdateUserAssignmentInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	assignment, err := h.repo.Update(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "user assignment not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update user assignment")
		return
	}
	writeJSON(w, http.StatusOK, assignment)
}

func (h *UserAssignmentHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user assignment ID")
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		if isFKViolation(err) {
			writeError(w, http.StatusConflict, "user assignment has dependent records")
			return
		}
		writeError(w, http.StatusNotFound, "user assignment not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
