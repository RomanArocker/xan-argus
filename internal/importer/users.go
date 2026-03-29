package importer

func usersConfig() *EntityConfig {
	return &EntityConfig{
		Name: "users", Table: "users",
		MatchKeys: []string{"first_name", "last_name", "type"},
		Columns: []ColumnConfig{
			{Header: "type", DBColumn: "type", Required: true, Type: "text"},
			{Header: "first_name", DBColumn: "first_name", Required: true, Type: "text"},
			{Header: "last_name", DBColumn: "last_name", Required: true, Type: "text"},
		},
	}
}
