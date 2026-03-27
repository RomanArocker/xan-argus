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
	// User routes
	mux.HandleFunc("GET /users", h.userList)
	mux.HandleFunc("GET /users/rows", h.userListRows)
	mux.HandleFunc("GET /users/new", h.userForm)
	mux.HandleFunc("GET /users/{id}/edit", h.userEditForm)
	// Service routes
	mux.HandleFunc("GET /services", h.serviceList)
	mux.HandleFunc("GET /services/rows", h.serviceListRows)
	mux.HandleFunc("GET /services/new", h.serviceForm)
	mux.HandleFunc("GET /services/{id}/edit", h.serviceEditForm)
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

func (h *PageHandler) userList(w http.ResponseWriter, r *http.Request) {
	params := paginationParams(r)
	users, err := h.userRepo.List(r.Context(), params)
	if err != nil {
		http.Error(w, "failed to load users", http.StatusInternalServerError)
		return
	}
	h.tmpl.Render(w, "layout", map[string]any{
		"Title":  "Users",
		"Users":  users,
		"Search": params.Search,
		"Filter": params.Filter,
	})
}

func (h *PageHandler) userListRows(w http.ResponseWriter, r *http.Request) {
	params := paginationParams(r)
	users, err := h.userRepo.List(r.Context(), params)
	if err != nil {
		http.Error(w, "failed to load users", http.StatusInternalServerError)
		return
	}
	h.tmpl.Render(w, "user_rows", users)
}

func (h *PageHandler) userForm(w http.ResponseWriter, r *http.Request) {
	h.tmpl.Render(w, "layout", map[string]any{
		"Title": "New User",
		"User":  model.User{},
		"IsNew": true,
	})
}

func (h *PageHandler) userEditForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid ID", http.StatusBadRequest)
		return
	}
	user, err := h.userRepo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	h.tmpl.Render(w, "layout", map[string]any{
		"Title": "Edit User",
		"User":  user,
		"IsNew": false,
	})
}

func (h *PageHandler) serviceList(w http.ResponseWriter, r *http.Request) {
	params := paginationParams(r)
	services, err := h.serviceRepo.List(r.Context(), params)
	if err != nil {
		http.Error(w, "failed to load services", http.StatusInternalServerError)
		return
	}
	h.tmpl.Render(w, "layout", map[string]any{
		"Title":    "Services",
		"Services": services,
		"Search":   params.Search,
	})
}

func (h *PageHandler) serviceListRows(w http.ResponseWriter, r *http.Request) {
	params := paginationParams(r)
	services, err := h.serviceRepo.List(r.Context(), params)
	if err != nil {
		http.Error(w, "failed to load services", http.StatusInternalServerError)
		return
	}
	h.tmpl.Render(w, "service_rows", services)
}

func (h *PageHandler) serviceForm(w http.ResponseWriter, r *http.Request) {
	h.tmpl.Render(w, "layout", map[string]any{
		"Title":   "New Service",
		"Service": model.Service{},
		"IsNew":   true,
	})
}

func (h *PageHandler) serviceEditForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid ID", http.StatusBadRequest)
		return
	}
	svc, err := h.serviceRepo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "service not found", http.StatusNotFound)
		return
	}
	h.tmpl.Render(w, "layout", map[string]any{
		"Title":   "Edit Service",
		"Service": svc,
		"IsNew":   false,
	})
}
