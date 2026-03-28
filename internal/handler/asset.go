package handler

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/xan-com/xan-argus/internal/model"
	"github.com/xan-com/xan-argus/internal/repository"
)

type AssetHandler struct {
	repo    *repository.AssetRepository
	catRepo *repository.HardwareCategoryRepository
}

func NewAssetHandler(repo *repository.AssetRepository, catRepo *repository.HardwareCategoryRepository) *AssetHandler {
	return &AssetHandler{repo: repo, catRepo: catRepo}
}

func (h *AssetHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/customers/{customerId}/assets", h.listByCustomer)
	mux.HandleFunc("POST /api/v1/customers/{customerId}/assets", h.create)
	mux.HandleFunc("GET /api/v1/assets/{id}", h.get)
	mux.HandleFunc("PUT /api/v1/assets/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/assets/{id}", h.delete)
}

func (h *AssetHandler) listByCustomer(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseUUID(r.PathValue("customerId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}
	params := paginationParams(r)
	params.Filter = r.URL.Query().Get("category_id")
	assets, err := h.repo.ListByCustomer(r.Context(), customerID, params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list assets")
		return
	}
	writeJSON(w, http.StatusOK, assets)
}

func (h *AssetHandler) create(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseUUID(r.PathValue("customerId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}
	var input model.CreateAssetInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input.CustomerID = customerID
	if input.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	// Validate field_values against category's field definitions
	if input.CategoryID.Valid && len(input.FieldValues) > 0 && string(input.FieldValues) != "{}" {
		cat, err := h.catRepo.GetByID(r.Context(), input.CategoryID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid category_id")
			return
		}
		if msg := validateFieldValues(input.FieldValues, cat.Fields); msg != "" {
			writeError(w, http.StatusBadRequest, msg)
			return
		}
	}
	asset, err := h.repo.Create(r.Context(), input)
	if err != nil {
		if isFKViolation(err) {
			writeError(w, http.StatusConflict, "invalid user_assignment_id or category_id")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create asset")
		return
	}
	writeJSON(w, http.StatusCreated, asset)
}

func (h *AssetHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset ID")
		return
	}
	asset, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "asset not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get asset")
		return
	}
	resp := model.AssetResponse{Asset: asset}
	if asset.CategoryID.Valid {
		cat, err := h.catRepo.GetByID(r.Context(), asset.CategoryID)
		if err == nil {
			resp.Category = &cat
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *AssetHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset ID")
		return
	}
	var input model.UpdateAssetInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	// Validate field_values against category's field definitions
	if len(input.FieldValues) > 0 && string(input.FieldValues) != "{}" {
		var catID pgtype.UUID
		if input.CategoryID.Valid {
			catID = input.CategoryID
		} else {
			existing, err := h.repo.GetByID(r.Context(), id)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					writeError(w, http.StatusNotFound, "asset not found")
					return
				}
				writeError(w, http.StatusInternalServerError, "failed to get asset")
				return
			}
			catID = existing.CategoryID
		}
		if catID.Valid {
			cat, err := h.catRepo.GetByID(r.Context(), catID)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid category_id")
				return
			}
			if msg := validateFieldValues(input.FieldValues, cat.Fields); msg != "" {
				writeError(w, http.StatusBadRequest, msg)
				return
			}
		}
	}
	asset, err := h.repo.Update(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "asset not found")
			return
		}
		if isFKViolation(err) {
			writeError(w, http.StatusConflict, "invalid user_assignment_id or category_id")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update asset")
		return
	}
	writeJSON(w, http.StatusOK, asset)
}

func (h *AssetHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset ID")
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		if isFKViolation(err) {
			writeError(w, http.StatusConflict, "asset has dependent records")
			return
		}
		writeError(w, http.StatusNotFound, "asset not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
