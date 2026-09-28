package domain

// Capital is a Brazilian state capital known to the domain.
type Capital struct {
	Slug      string
	Name      string
	StateCode string
	Latitude  float64
	Longitude float64
}
