package importer

func licensesConfig() *EntityConfig {
	return &EntityConfig{
		Name: "licenses", Table: "licenses",
		MatchKeys: []string{"product_name", "customer_id", "license_key"},
		Columns: []ColumnConfig{
			{Header: "product_name", DBColumn: "product_name", Required: true, Type: "text"},
			{Header: "license_key", DBColumn: "license_key", Required: true, Type: "text"},
			{Header: "customer", DBColumn: "customer_id", Required: true, Type: "uuid",
				FK: &FKConfig{Table: "customers", LookupCol: "name", Strategy: "name"}},
			{Header: "user_assignment_id", DBColumn: "user_assignment_id", Type: "uuid",
				FK: &FKConfig{Table: "user_assignments", LookupCol: "id", Strategy: "uuid"}},
			{Header: "quantity", DBColumn: "quantity", Type: "int"},
			{Header: "valid_from", DBColumn: "valid_from", Type: "date"},
			{Header: "valid_until", DBColumn: "valid_until", Type: "date"},
		},
	}
}
