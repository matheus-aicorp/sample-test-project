package application

import (
	"context"
	"fmt"
	"time"

	"github.com/matheus-aicorp/sample-test-project/internal/domain"
)

// The valid set has 27 capitals, so anything above this is payload abuse.
const maxSlugsPerRequest = 50

// GetCapitalsWeather is the use case behind the API's only endpoint.
type GetCapitalsWeather struct {
	capitals domain.CapitalRepository
	provider domain.WeatherProvider
	clock    domain.Clock
}

// Result carries the readings plus the instant they were produced, so the
// transport layer never needs a clock of its own.
type Result struct {
	Readings    []domain.CurrentWeather
	RetrievedAt time.Time
}

func NewGetCapitalsWeather(
	capitals domain.CapitalRepository,
	provider domain.WeatherProvider,
	clock domain.Clock,
) *GetCapitalsWeather {
	return &GetCapitalsWeather{capitals: capitals, provider: provider, clock: clock}
}

// Execute returns current weather for the requested capitals. An empty slugs
// slice means every capital; duplicates collapse, preserving request order.
func (u *GetCapitalsWeather) Execute(ctx context.Context, slugs []string) (*Result, error) {
	capitals, err := u.resolve(slugs)
	if err != nil {
		return nil, err
	}

	readings, err := u.provider.CurrentWeather(ctx, capitals)
	if err != nil {
		// Re-chained onto a domain sentinel so the transport layer maps it to
		// 502 without knowing which adapter failed.
		return nil, fmt.Errorf("%w: %w", domain.ErrWeatherUnavailable, err)
	}

	return &Result{Readings: readings, RetrievedAt: u.clock.Now().UTC()}, nil
}

func (u *GetCapitalsWeather) resolve(slugs []string) ([]domain.Capital, error) {
	if len(slugs) == 0 {
		return u.capitals.All(), nil
	}

	if len(slugs) > maxSlugsPerRequest {
		return nil, &domain.InvalidRequestError{
			Message: fmt.Sprintf("at most %d capitals per request, got %d", maxSlugsPerRequest, len(slugs)),
		}
	}

	var (
		resolved []domain.Capital
		unknown  []string
		seen     = make(map[string]struct{}, len(slugs))
	)

	for _, raw := range slugs {
		slug := domain.NormalizeSlug(raw)
		if slug == "" {
			continue
		}
		if _, repeated := seen[slug]; repeated {
			continue
		}
		seen[slug] = struct{}{}

		capital, ok := u.capitals.BySlug(slug)
		if !ok {
			unknown = append(unknown, slug)
			continue
		}
		resolved = append(resolved, capital)
	}

	if len(unknown) > 0 {
		return nil, &domain.UnknownCapitalsError{
			Slugs:      unknown,
			ValidSlugs: validSlugs(u.capitals),
		}
	}

	return resolved, nil
}

func validSlugs(repo domain.CapitalRepository) []string {
	all := repo.All()
	slugs := make([]string, 0, len(all))
	for _, capital := range all {
		slugs = append(slugs, capital.Slug)
	}
	return slugs
}
