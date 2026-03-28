package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/xan-com/xan-argus/internal/model"
	"github.com/xan-com/xan-argus/internal/repository"
)

type PageHandler struct {
	tmpl                 *TemplateEngine
	customerRepo         *repository.CustomerRepository
	userRepo             *repository.UserRepository
	serviceRepo          *repository.ServiceRepository
	userAssignmentRepo   *repository.UserAssignmentRepository
	assetRepo            *repository.AssetRepository
	licenseRepo          *repository.LicenseRepository
	customerServiceRepo  *repository.CustomerServiceRepository
	hardwareCategoryRepo *repository.HardwareCategoryRepository
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
	hardwareCategoryRepo *repository.HardwareCategoryRepository,
) *PageHandler {
	return &PageHandler{
		tmpl:                 tmpl,
		customerRepo:         customerRepo,
		userRepo:             userRepo,
		serviceRepo:          serviceRepo,
		userAssignmentRepo:   userAssignmentRepo,
		assetRepo:            assetRepo,
		licenseRepo:          licenseRepo,
		customerServiceRepo:  customerServiceRepo,
		hardwareCategoryRepo: hardwareCategoryRepo,
	}
}

func (h *PageHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /{$}", h.home)
	mux.HandleFunc("GET /customers", h.customerList)
	mux.HandleFunc("GET /customers/rows", h.customerListRows)
	mux.HandleFunc("GET /customers/new", h.customerForm)
	mux.HandleFunc("GET /customers/{id}", h.customerDetail)
	mux.HandleFunc("GET /customers/{id}/edit", h.customerEditForm)
	mux.HandleFunc("GET /customers/{customerId}/assets/{assetId}", h.assetDetail)
	mux.HandleFunc("GET /users", h.userList)
	mux.HandleFunc("GET /users/rows", h.userListRows)
	mux.HandleFunc("GET /users/new", h.userForm)
	mux.HandleFunc("GET /users/{id}/edit", h.userEditForm)
	mux.HandleFunc("GET /services", h.serviceList)
	mux.HandleFunc("GET /services/rows", h.serviceListRows)
	mux.HandleFunc("GET /services/new", h.serviceForm)
	mux.HandleFunc("GET /services/{id}/edit", h.serviceEditForm)
	mux.HandleFunc("GET /categories", h.categoryList)
	mux.HandleFunc("GET /categories/new", h.categoryForm)
	mux.HandleFunc("GET /categories/{id}/edit", h.categoryEditForm)
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
	h.tmpl.RenderPage(w, "customers/list", map[string]any{
		"Title":     "Customers",
		"Customers": customers,
		"Search":    params.Search,
	})
}

func (h *PageHandler) customerListRows(w http.ResponseWriter, r *http.Request) {
	params := paginationParams(r)
	customers, err := h.customerRepo.List(r.Context(), params)
	if err != nil {
		http.Error(w, "failed to load customers", http.StatusInternalServerError)
		return
	}
	h.tmpl.RenderPartial(w, "customer_rows", customers)
}

func (h *PageHandler) customerForm(w http.ResponseWriter, r *http.Request) {
	h.tmpl.RenderPage(w, "customers/form", map[string]any{
		"Title":    "New Customer",
		"Customer": model.Customer{},
		"IsNew":    true,
	})
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
	categories, err := h.hardwareCategoryRepo.List(r.Context())
	if err != nil {
		http.Error(w, "failed to load categories", http.StatusInternalServerError)
		return
	}
	categoryMap := make(map[string]string)
	for _, cat := range categories {
		categoryMap[uuidToStr(cat.ID)] = cat.Name
	}

	h.tmpl.RenderPage(w, "customers/detail", map[string]any{
		"Title":            customer.Name,
		"Customer":         customer,
		"Assets":           assets,
		"Licenses":         licenses,
		"Assignments":      assignments,
		"CustomerServices": customerServices,
		"Users":            users,
		"AllServices":      allServices,
		"AllCategories":    categories,
		"CategoryMap":      categoryMap,
	})
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
	h.tmpl.RenderPage(w, "customers/form", map[string]any{
		"Title":    "Edit Customer",
		"Customer": customer,
		"IsNew":    false,
	})
}

func (h *PageHandler) userList(w http.ResponseWriter, r *http.Request) {
	params := paginationParams(r)
	users, err := h.userRepo.List(r.Context(), params)
	if err != nil {
		http.Error(w, "failed to load users", http.StatusInternalServerError)
		return
	}
	h.tmpl.RenderPage(w, "users/list", map[string]any{
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
	h.tmpl.RenderPartial(w, "user_rows", users)
}

func (h *PageHandler) userForm(w http.ResponseWriter, r *http.Request) {
	h.tmpl.RenderPage(w, "users/form", map[string]any{
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
	h.tmpl.RenderPage(w, "users/form", map[string]any{
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
	h.tmpl.RenderPage(w, "services/list", map[string]any{
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
	h.tmpl.RenderPartial(w, "service_rows", services)
}

func (h *PageHandler) serviceForm(w http.ResponseWriter, r *http.Request) {
	h.tmpl.RenderPage(w, "services/form", map[string]any{
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
	h.tmpl.RenderPage(w, "services/form", map[string]any{
		"Title":   "Edit Service",
		"Service": svc,
		"IsNew":   false,
	})
}

func (h *PageHandler) categoryList(w http.ResponseWriter, r *http.Request) {
	categories, err := h.hardwareCategoryRepo.List(r.Context())
	if err != nil {
		http.Error(w, "failed to load categories", http.StatusInternalServerError)
		return
	}
	for i, cat := range categories {
		fields, err := h.hardwareCategoryRepo.ListFields(r.Context(), cat.ID)
		if err != nil {
			http.Error(w, "failed to load fields", http.StatusInternalServerError)
			return
		}
		categories[i].Fields = fields
	}
	h.tmpl.RenderPage(w, "categories/list", map[string]any{
		"Title":      "Hardware Categories",
		"Categories": categories,
	})
}

func (h *PageHandler) categoryForm(w http.ResponseWriter, r *http.Request) {
	h.tmpl.RenderPage(w, "categories/form", map[string]any{
		"Title":    "New Category",
		"Category": model.HardwareCategory{},
		"IsNew":    true,
	})
}

func (h *PageHandler) categoryEditForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid ID", http.StatusBadRequest)
		return
	}
	cat, err := h.hardwareCategoryRepo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "category not found", http.StatusNotFound)
		return
	}
	h.tmpl.RenderPage(w, "categories/form", map[string]any{
		"Title":    "Edit Category",
		"Category": cat,
		"IsNew":    false,
	})
}

// AssetFieldDisplay holds a pre-resolved label+value pair for template rendering.
type AssetFieldDisplay struct {
	Name  string
	Value string
}

func (h *PageHandler) assetDetail(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseUUID(r.PathValue("customerId"))
	if err != nil {
		http.Error(w, "invalid customer ID", http.StatusBadRequest)
		return
	}
	assetID, err := parseUUID(r.PathValue("assetId"))
	if err != nil {
		http.Error(w, "invalid asset ID", http.StatusBadRequest)
		return
	}
	customer, err := h.customerRepo.GetByID(r.Context(), customerID)
	if err != nil {
		http.Error(w, "customer not found", http.StatusNotFound)
		return
	}
	asset, err := h.assetRepo.GetByID(r.Context(), assetID)
	if err != nil {
		http.Error(w, "asset not found", http.StatusNotFound)
		return
	}
	if uuidToStr(asset.CustomerID) != uuidToStr(customerID) {
		http.Error(w, "asset not found", http.StatusNotFound)
		return
	}

	var cat *model.HardwareCategory
	var fields []AssetFieldDisplay
	if asset.CategoryID.Valid {
		// If category was deleted, err != nil → cat stays nil, fields stays empty.
		// This is intentional: show asset without category fields rather than erroring.
		c, err := h.hardwareCategoryRepo.GetByID(r.Context(), asset.CategoryID)
		if err == nil {
			cat = &c
			// c.Fields is pre-sorted by sort_order, name from the repository query
			var vals map[string]interface{}
			if err := json.Unmarshal(asset.FieldValues, &vals); err == nil {
				for _, fd := range c.Fields {
					v := "—"
					// field_values keys are field definition UUIDs, not names
					if raw, ok := vals[uuidToStr(fd.ID)]; ok && raw != nil {
						v = fmt.Sprintf("%v", raw)
					}
					fields = append(fields, AssetFieldDisplay{Name: fd.Name, Value: v})
				}
			}
		}
	}

	h.tmpl.RenderPage(w, "customers/asset_detail", map[string]any{
		"Title":    asset.Name + " — " + customer.Name,
		"Customer": customer,
		"Asset":    asset,
		"Category": cat,
		"Fields":   fields,
	})
}
