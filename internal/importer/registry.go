package importer

import "fmt"

type Registry struct {
	configs map[string]*EntityConfig
	order   []string
}

func NewRegistry() *Registry {
	r := &Registry{
		configs: make(map[string]*EntityConfig),
		order: []string{
			"customers", "hardware-categories", "services", "users",
			"field-definitions", "user-assignments", "customer-services",
			"assets", "licenses",
		},
	}
	r.register(customersConfig())
	r.register(hardwareCategoriesConfig())
	r.register(servicesConfig())
	r.register(usersConfig())
	r.register(fieldDefinitionsConfig())
	r.register(userAssignmentsConfig())
	r.register(customerServicesConfig())
	r.register(assetsConfig())
	r.register(licensesConfig())
	return r
}

func (r *Registry) register(cfg *EntityConfig) {
	r.configs[cfg.Name] = cfg
}

func (r *Registry) Get(name string) (*EntityConfig, error) {
	cfg, ok := r.configs[name]
	if !ok {
		return nil, fmt.Errorf("unknown entity: %s", name)
	}
	return cfg, nil
}

func (r *Registry) OrderedNames() []string {
	return r.order
}
