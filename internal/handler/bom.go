package handler

import (
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/xan-com/xan-argus/internal/model"
	"github.com/xan-com/xan-argus/internal/repository"
)

var allowedCurrencies = map[string]bool{
	"CHF": true, "EUR": true, "USD": true,
	"GBP": true, "JPY": true, "CAD": true, "AUD": true,
}

type BOMHandler struct {
	repo *repository.BOMRepository
}

func NewBOMHandler(repo *repository.BOMRepository) *BOMHandler {
	return &BOMHandler{repo: repo}
}

func (h *BOMHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/assets/{assetId}/bom", h.list)
	mux.HandleFunc("POST /api/v1/assets/{assetId}/bom", h.create)
	mux.HandleFunc("GET /api/v1/assets/{assetId}/bom/{id}", h.get)
	mux.HandleFunc("PUT /api/v1/assets/{assetId}/bom/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/assets/{assetId}/bom/{id}", h.delete)
	mux.HandleFunc("PUT /api/v1/assets/{assetId}/bom/{id}/up", h.moveUp)
	mux.HandleFunc("PUT /api/v1/assets/{assetId}/bom/{id}/down", h.moveDown)
}

func (h *BOMHandler) list(w http.ResponseWriter, r *http.Request) {
	assetID, err := parseUUID(r.PathValue("assetId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset ID")
		return
	}
	items, err := h.repo.ListByAsset(r.Context(), assetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list bom items")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *BOMHandler) create(w http.ResponseWriter, r *http.Request) {
	assetID, err := parseUUID(r.PathValue("assetId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset ID")
		return
	}
	var input model.CreateBOMItemInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if !allowedCurrencies[input.Currency] {
		writeError(w, http.StatusBadRequest, "invalid currency code")
		return
	}
	if !input.Quantity.Valid {
		writeError(w, http.StatusBadRequest, "quantity is required")
		return
	}
	if !input.UnitPrice.Valid {
		writeError(w, http.StatusBadRequest, "unit_price is required")
		return
	}
	item, err := h.repo.Create(r.Context(), assetID, input)
	if err != nil {
		if isFKViolation(err) {
			writeError(w, http.StatusConflict, "asset or unit not found")
			return
		}
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "bom item with this name already exists for this asset")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create bom item")
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *BOMHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid bom item ID")
		return
	}
	item, err := h.repo.GetByID(r.Context(), id)
	if err == pgx.ErrNoRows {
		writeError(w, http.StatusNotFound, "bom item not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get bom item")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *BOMHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid bom item ID")
		return
	}
	var input model.UpdateBOMItemInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.Currency != nil && !allowedCurrencies[*input.Currency] {
		writeError(w, http.StatusBadRequest, "invalid currency code")
		return
	}
	item, err := h.repo.Update(r.Context(), id, input)
	if err == pgx.ErrNoRows {
		writeError(w, http.StatusNotFound, "bom item not found")
		return
	}
	if err != nil {
		if isFKViolation(err) {
			writeError(w, http.StatusConflict, "unit not found")
			return
		}
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "bom item with this name already exists for this asset")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update bom item")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *BOMHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid bom item ID")
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "bom item not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete bom item")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *BOMHandler) moveUp(w http.ResponseWriter, r *http.Request) {
	h.move(w, r, "up")
}

func (h *BOMHandler) moveDown(w http.ResponseWriter, r *http.Request) {
	h.move(w, r, "down")
}

func (h *BOMHandler) move(w http.ResponseWriter, r *http.Request, direction string) {
	assetID, err := parseUUID(r.PathValue("assetId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset ID")
		return
	}
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid bom item ID")
		return
	}
	if err := h.repo.SwapSortOrder(r.Context(), assetID, id, direction); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "bom item not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to reorder bom item")
		return
	}
	// Signal HTMX to refresh the bom rows partial
	w.Header().Set("HX-Trigger", "bom-reorder")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
