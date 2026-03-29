package importer

func hardwareCategoriesConfig() *EntityConfig {
	return &EntityConfig{
		Name: "hardware-categories", Table: "hardware_categories",
		MatchKeys: []string{"name"},
		Columns: []ColumnConfig{
			{Header: "name", DBColumn: "name", Required: true, Type: "text"},
			{Header: "description", DBColumn: "description", Type: "text"},
		},
	}
}
