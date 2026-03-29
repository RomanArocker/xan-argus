package importer

func fieldDefinitionsConfig() *EntityConfig {
	return &EntityConfig{
		Name: "field-definitions", Table: "category_field_definitions",
		MatchKeys: []string{"category_id", "name"},
		Columns: []ColumnConfig{
			{Header: "category", DBColumn: "category_id", Required: true, Type: "uuid",
				FK: &FKConfig{Table: "hardware_categories", LookupCol: "name", Strategy: "name"}},
			{Header: "name", DBColumn: "name", Required: true, Type: "text"},
			{Header: "field_type", DBColumn: "field_type", Required: true, Type: "text"},
			{Header: "required", DBColumn: "required", Type: "bool"},
			{Header: "sort_order", DBColumn: "sort_order", Type: "int"},
		},
	}
}
