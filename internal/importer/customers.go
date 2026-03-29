package importer

func customersConfig() *EntityConfig {
	return &EntityConfig{
		Name: "customers", Table: "customers",
		MatchKeys: []string{"name"},
		Columns: []ColumnConfig{
			{Header: "name", DBColumn: "name", Required: true, Type: "text"},
			{Header: "contact_email", DBColumn: "contact_email", Type: "text"},
			{Header: "notes", DBColumn: "notes", Type: "text"},
		},
	}
}
