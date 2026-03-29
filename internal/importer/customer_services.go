package importer

func customerServicesConfig() *EntityConfig {
	return &EntityConfig{
		Name: "customer-services", Table: "customer_services",
		MatchKeys: []string{"customer_id", "service_id"},
		Columns: []ColumnConfig{
			{Header: "customer", DBColumn: "customer_id", Required: true, Type: "uuid",
				FK: &FKConfig{Table: "customers", LookupCol: "name", Strategy: "name"}},
			{Header: "service", DBColumn: "service_id", Required: true, Type: "uuid",
				FK: &FKConfig{Table: "services", LookupCol: "name", Strategy: "name"}},
			{Header: "customizations", DBColumn: "customizations", Type: "json"},
			{Header: "notes", DBColumn: "notes", Type: "text"},
		},
	}
}
