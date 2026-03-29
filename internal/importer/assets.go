package importer

func assetsConfig() *EntityConfig {
	return &EntityConfig{
		Name: "assets", Table: "assets",
		MatchKeys: []string{"name", "customer_id"},
		Columns: []ColumnConfig{
			{Header: "name", DBColumn: "name", Required: true, Type: "text"},
			{Header: "description", DBColumn: "description", Type: "text"},
			{Header: "customer", DBColumn: "customer_id", Required: true, Type: "uuid",
				FK: &FKConfig{Table: "customers", LookupCol: "name", Strategy: "name"}},
			{Header: "category", DBColumn: "category_id", Type: "uuid",
				FK: &FKConfig{Table: "hardware_categories", LookupCol: "name", Strategy: "name"}},
			{Header: "user_assignment_id", DBColumn: "user_assignment_id", Type: "uuid",
				FK: &FKConfig{Table: "user_assignments", LookupCol: "id", Strategy: "uuid"}},
			{Header: "metadata", DBColumn: "metadata", Type: "json"},
			{Header: "field_values", DBColumn: "field_values", Type: "json"},
		},
	}
}
