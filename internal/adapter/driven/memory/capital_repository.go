package memory

import (
	"sort"

	"github.com/matheus-aicorp/sample-test-project/internal/domain"
)

// The 27 Brazilian capitals with the coordinates of each municipal seat.
// Names stay in Portuguese because they are data, not code.
var capitals = []domain.Capital{
	{Slug: "aracaju", Name: "Aracaju", StateCode: "SE", Latitude: -10.9472, Longitude: -37.0731},
	{Slug: "belem", Name: "Belém", StateCode: "PA", Latitude: -1.4558, Longitude: -48.5024},
	{Slug: "belo-horizonte", Name: "Belo Horizonte", StateCode: "MG", Latitude: -19.9167, Longitude: -43.9345},
	{Slug: "boa-vista", Name: "Boa Vista", StateCode: "RR", Latitude: 2.8238, Longitude: -60.6753},
	{Slug: "brasilia", Name: "Brasília", StateCode: "DF", Latitude: -15.7939, Longitude: -47.8828},
	{Slug: "campo-grande", Name: "Campo Grande", StateCode: "MS", Latitude: -20.4428, Longitude: -54.6481},
	{Slug: "cuiaba", Name: "Cuiabá", StateCode: "MT", Latitude: -15.6014, Longitude: -56.0979},
	{Slug: "curitiba", Name: "Curitiba", StateCode: "PR", Latitude: -25.4284, Longitude: -49.2733},
	{Slug: "florianopolis", Name: "Florianópolis", StateCode: "SC", Latitude: -27.5954, Longitude: -48.5480},
	{Slug: "fortaleza", Name: "Fortaleza", StateCode: "CE", Latitude: -3.7319, Longitude: -38.5267},
	{Slug: "goiania", Name: "Goiânia", StateCode: "GO", Latitude: -16.6869, Longitude: -49.2648},
	{Slug: "joao-pessoa", Name: "João Pessoa", StateCode: "PB", Latitude: -7.1195, Longitude: -34.8450},
	{Slug: "macapa", Name: "Macapá", StateCode: "AP", Latitude: 0.0389, Longitude: -51.0664},
	{Slug: "maceio", Name: "Maceió", StateCode: "AL", Latitude: -9.6658, Longitude: -35.7350},
	{Slug: "manaus", Name: "Manaus", StateCode: "AM", Latitude: -3.1190, Longitude: -60.0217},
	{Slug: "natal", Name: "Natal", StateCode: "RN", Latitude: -5.7945, Longitude: -35.2110},
	{Slug: "palmas", Name: "Palmas", StateCode: "TO", Latitude: -10.2128, Longitude: -48.3603},
	{Slug: "porto-alegre", Name: "Porto Alegre", StateCode: "RS", Latitude: -30.0346, Longitude: -51.2177},
	{Slug: "porto-velho", Name: "Porto Velho", StateCode: "RO", Latitude: -8.7612, Longitude: -63.9004},
	{Slug: "recife", Name: "Recife", StateCode: "PE", Latitude: -8.0476, Longitude: -34.8770},
	{Slug: "rio-branco", Name: "Rio Branco", StateCode: "AC", Latitude: -9.9754, Longitude: -67.8249},
	{Slug: "rio-de-janeiro", Name: "Rio de Janeiro", StateCode: "RJ", Latitude: -22.9068, Longitude: -43.1729},
	{Slug: "salvador", Name: "Salvador", StateCode: "BA", Latitude: -12.9714, Longitude: -38.5014},
	{Slug: "sao-luis", Name: "São Luís", StateCode: "MA", Latitude: -2.5300, Longitude: -44.3028},
	{Slug: "sao-paulo", Name: "São Paulo", StateCode: "SP", Latitude: -23.5505, Longitude: -46.6333},
	{Slug: "teresina", Name: "Teresina", StateCode: "PI", Latitude: -5.0892, Longitude: -42.8016},
	{Slug: "vitoria", Name: "Vitória", StateCode: "ES", Latitude: -20.3155, Longitude: -40.3128},
}

// CapitalRepository is the in-memory adapter for domain.CapitalRepository.
// The slug index is built once at startup, so lookups are O(1) and the
// dataset stays immutable for the process lifetime.
type CapitalRepository struct {
	sorted []domain.Capital
	bySlug map[string]domain.Capital
}

var _ domain.CapitalRepository = (*CapitalRepository)(nil)

func NewCapitalRepository() *CapitalRepository {
	sorted := make([]domain.Capital, len(capitals))
	copy(sorted, capitals)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	bySlug := make(map[string]domain.Capital, len(sorted))
	for _, capital := range sorted {
		bySlug[capital.Slug] = capital
	}

	return &CapitalRepository{sorted: sorted, bySlug: bySlug}
}

func (r *CapitalRepository) All() []domain.Capital {
	out := make([]domain.Capital, len(r.sorted))
	copy(out, r.sorted)
	return out
}

func (r *CapitalRepository) BySlug(slug string) (domain.Capital, bool) {
	capital, ok := r.bySlug[slug]
	return capital, ok
}
