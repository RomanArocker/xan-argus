package importer

func userAssignmentsConfig() *EntityConfig {
	return &EntityConfig{
		Name: "user-assignments", Table: "user_assignments",
		MatchKeys: []string{"user_id", "customer_id"},
		Columns: []ColumnConfig{
			{Header: "user_id", DBColumn: "user_id", Required: true, Type: "uuid",
				FK: &FKConfig{Table: "users", LookupCol: "id", Strategy: "uuid"}},
			{Header: "customer", DBColumn: "customer_id", Required: true, Type: "uuid",
				FK: &FKConfig{Table: "customers", LookupCol: "name", Strategy: "name"}},
			{Header: "role", DBColumn: "role", Type: "text"},
			{Header: "email", DBColumn: "email", Type: "text"},
			{Header: "phone", DBColumn: "phone", Type: "text"},
			{Header: "notes", DBColumn: "notes", Type: "text"},
		},
	}
}
