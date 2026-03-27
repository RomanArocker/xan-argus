package model

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

type ListParams struct {
	Limit  int
	Offset int
	Search string // optional name/text search (?q=)
	Filter string // optional type filter (?type=)
}

func (p *ListParams) Normalize() {
	if p.Limit <= 0 {
		p.Limit = DefaultLimit
	}
	if p.Limit > MaxLimit {
		p.Limit = MaxLimit
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
}
