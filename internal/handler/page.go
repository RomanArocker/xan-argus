package handler

import (
	"net/http"

	"github.com/xan-com/xan-pythia/internal/model"
	"github.com/xan-com/xan-pythia/internal/repository"
)

type PageHandler struct {
	tmpl                *TemplateEngine
	customerRepo        *repository.CustomerRepository
	userRepo            *repository.UserRepository
	serviceRepo         *repository.ServiceRepository
	userAssignmentRepo  *repository.UserAssignmentRepository
	assetRepo           *repository.AssetRepository
	licenseRepo         *repository.LicenseRepository
	customerServiceRepo *repository.CustomerServiceRepository
}

func NewPageHandler(
	tmpl *TemplateEngine,
	customerRepo *repository.CustomerRepository,
	userRepo *repository.UserRepository,
	serviceRepo *repository.ServiceRepository,
	userAssignmentRepo *repository.UserAssignmentRepository,
	assetRepo *repository.AssetRepository,
	licenseRepo *repository.LicenseRepository,
	customerServiceRepo *repository.CustomerServiceRepository,
) *PageHandler {
	return &PageHandler{
		tmpl:                tmpl,
		customerRepo:        customerRepo,
		userRepo:            userRepo,
		serviceRepo:         serviceRepo,
		userAssignmentRepo:  userAssignmentRepo,
		assetRepo:           assetRepo,
		licenseRepo:         licenseRepo,
		customerServiceRepo: customerServiceRepo,
	}
}

func (h *PageHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /{$}", h.home)
	mux.HandleFunc("GET /customers", h.customerList)
	mux.HandleFunc("GET /customers/rows", h.customerListRows)
	mux.HandleFunc("GET /customers/new", h.customerForm)
	mux.HandleFunc("GET /customers/{id}", h.customerDetail)
	mux.HandleFunc("GET /customers/{id}/edit", h.customerEditForm)
}

func (h *PageHandler) home(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/customers", http.StatusFound)
}

func (h *PageHandler) customerList(w http.ResponseWriter, r *http.Request) {
	params := paginationParams(r)
	customers, err := h.customerRepo.List(r.Context(), params)
	if err != nil {
		http.Error(w, "failed to load customers", http.StatusInternalServerError)
		return
	}
	data := map[string]any{
		"Title":     "Customers",
		"Customers": customers,
		"Search":    params.Search,
	}
	h.tmpl.Render(w, "layout", data)
}

func (h *PageHandler) customerListRows(w http.ResponseWriter, r *http.Request) {
	params := paginationParams(r)
	customers, err := h.customerRepo.List(r.Context(), params)
	if err != nil {
		http.Error(w, "failed to load customers", http.StatusInternalServerError)
		return
	}
	h.tmpl.Render(w, "customer_rows", customers)
}

func (h *PageHandler) customerForm(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{
		"Title":    "New Customer",
		"Customer": model.Customer{},
		"IsNew":    true,
	}
	h.tmpl.Render(w, "layout", data)
}

func (h *PageHandler) customerDetail(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid ID", http.StatusBadRequest)
		return
	}
	customer, err := h.customerRepo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "customer not found", http.StatusNotFound)
		return
	}
	listParams := model.ListParams{Limit: 100}
	assets, _ := h.assetRepo.ListByCustomer(r.Context(), id, listParams)
	licenses, _ := h.licenseRepo.ListByCustomer(r.Context(), id, listParams)
	assignments, _ := h.userAssignmentRepo.ListByCustomer(r.Context(), id, listParams)
	customerServices, _ := h.customerServiceRepo.ListByCustomer(r.Context(), id, listParams)
	users, _ := h.userRepo.List(r.Context(), model.ListParams{Limit: 100})
	allServices, _ := h.serviceRepo.List(r.Context(), model.ListParams{Limit: 100})

	data := map[string]any{
		"Title":            customer.Name,
		"Customer":         customer,
		"Assets":           assets,
		"Licenses":         licenses,
		"Assignments":      assignments,
		"CustomerServices": customerServices,
		"Users":            users,
		"AllServices":      allServices,
	}
	h.tmpl.Render(w, "layout", data)
}

func (h *PageHandler) customerEditForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid ID", http.StatusBadRequest)
		return
	}
	customer, err := h.customerRepo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "customer not found", http.StatusNotFound)
		return
	}
	data := map[string]any{
		"Title":    "Edit Customer",
		"Customer": customer,
		"IsNew":    false,
	}
	h.tmpl.Render(w, "layout", data)
}
