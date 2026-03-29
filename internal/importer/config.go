package importer

type EntityConfig struct {
	Name      string
	Table     string
	MatchKeys []string
	Columns   []ColumnConfig
}

type ColumnConfig struct {
	Header   string
	DBColumn string
	Required bool
	Type     string
	FK       *FKConfig
}

type FKConfig struct {
	Table     string
	LookupCol string
	Strategy  string
}

type ImportResult struct {
	Total   int           `json:"total"`
	Created int           `json:"created"`
	Updated int           `json:"updated"`
	Errors  []ImportError `json:"errors"`
}

type ImportError struct {
	Row     int    `json:"row"`
	Column  string `json:"column"`
	Message string `json:"message"`
}
