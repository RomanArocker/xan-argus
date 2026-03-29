package importer

func servicesConfig() *EntityConfig {
	return &EntityConfig{
		Name: "services", Table: "services",
		MatchKeys: []string{"name"},
		Columns: []ColumnConfig{
			{Header: "name", DBColumn: "name", Required: true, Type: "text"},
			{Header: "description", DBColumn: "description", Type: "text"},
		},
	}
}
