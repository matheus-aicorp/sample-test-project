package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/matheus-aicorp/sample-test-project/internal/application"
	"github.com/matheus-aicorp/sample-test-project/internal/domain"
	"github.com/matheus-aicorp/sample-test-project/internal/mocks"
)

var (
	saoPaulo = domain.Capital{Slug: "sao-paulo", Name: "São Paulo", StateCode: "SP", Latitude: -23.5505, Longitude: -46.6333}
	curitiba = domain.Capital{Slug: "curitiba", Name: "Curitiba", StateCode: "PR", Latitude: -25.4284, Longitude: -49.2733}
	recife   = domain.Capital{Slug: "recife", Name: "Recife", StateCode: "PE", Latitude: -8.0476, Longitude: -34.8770}

	frozenNow = time.Date(2026, 9, 27, 20, 30, 0, 0, time.UTC)
)

type fixture struct {
	repo     *mocks.MockCapitalRepository
	provider *mocks.MockWeatherProvider
	clock    *mocks.MockClock
	useCase  *application.GetCapitalsWeather
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	ctrl := gomock.NewController(t)
	f := &fixture{
		repo:     mocks.NewMockCapitalRepository(ctrl),
		provider: mocks.NewMockWeatherProvider(ctrl),
		clock:    mocks.NewMockClock(ctrl),
	}
	f.useCase = application.NewGetCapitalsWeather(f.repo, f.provider, f.clock)
	return f
}

// captureCapitals records what the use case hands to the provider so tests can
// assert resolution order and de-duplication.
func (f *fixture) captureCapitals() *[]domain.Capital {
	var received []domain.Capital
	f.provider.EXPECT().
		CurrentWeather(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, capitals []domain.Capital) ([]domain.CurrentWeather, error) {
			received = capitals
			return nil, nil
		}).
		AnyTimes()
	// A successful provider call makes the use case stamp RetrievedAt.
	f.clock.EXPECT().Now().Return(frozenNow).AnyTimes()
	return &received
}

func slugsOf(capitals []domain.Capital) []string {
	out := make([]string, 0, len(capitals))
	for _, capital := range capitals {
		out = append(out, capital.Slug)
	}
	return out
}

func TestExecuteWithoutFilterUsesEveryCapital(t *testing.T) {
	f := newFixture(t)
	all := []domain.Capital{curitiba, recife, saoPaulo}
	readings := []domain.CurrentWeather{{Capital: saoPaulo, TemperatureCelsius: 21.5}}

	f.repo.EXPECT().All().Return(all)
	f.provider.EXPECT().CurrentWeather(gomock.Any(), all).Return(readings, nil)
	f.clock.EXPECT().Now().Return(frozenNow)

	result, err := f.useCase.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Readings) != 1 || result.Readings[0].TemperatureCelsius != 21.5 {
		t.Errorf("Readings = %+v", result.Readings)
	}
	if !result.RetrievedAt.Equal(frozenNow) {
		t.Errorf("RetrievedAt = %v, want %v", result.RetrievedAt, frozenNow)
	}
}

func TestExecuteConvertsRetrievedAtToUTC(t *testing.T) {
	f := newFixture(t)
	saoPauloZone := time.FixedZone("America/Sao_Paulo", -3*60*60)
	local := time.Date(2026, 9, 27, 17, 30, 0, 0, saoPauloZone)

	f.repo.EXPECT().All().Return([]domain.Capital{saoPaulo})
	f.provider.EXPECT().CurrentWeather(gomock.Any(), gomock.Any()).Return(nil, nil)
	f.clock.EXPECT().Now().Return(local)

	result, err := f.useCase.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RetrievedAt.Location() != time.UTC {
		t.Errorf("RetrievedAt location = %v, want UTC", result.RetrievedAt.Location())
	}
	if want := time.Date(2026, 9, 27, 20, 30, 0, 0, time.UTC); !result.RetrievedAt.Equal(want) {
		t.Errorf("RetrievedAt = %v, want %v", result.RetrievedAt, want)
	}
}

func TestExecuteKeepsRequestOrderAndFilters(t *testing.T) {
	f := newFixture(t)
	received := f.captureCapitals()

	f.repo.EXPECT().BySlug("recife").Return(recife, true)
	f.repo.EXPECT().BySlug("sao-paulo").Return(saoPaulo, true)

	if _, err := f.useCase.Execute(context.Background(), []string{"recife", "sao-paulo"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := slugsOf(*received); len(got) != 2 || got[0] != "recife" || got[1] != "sao-paulo" {
		t.Errorf("provider received %v, want [recife sao-paulo] in that order", got)
	}
}

func TestExecuteCollapsesDuplicatesAfterNormalization(t *testing.T) {
	f := newFixture(t)
	received := f.captureCapitals()

	f.repo.EXPECT().BySlug("sao-paulo").Return(saoPaulo, true).Times(1)

	if _, err := f.useCase.Execute(context.Background(), []string{"SAO PAULO", "sao-paulo", " Sao_Paulo "}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(*received) != 1 {
		t.Errorf("duplicates not collapsed: %v", slugsOf(*received))
	}
}

func TestExecuteIgnoresBlankSlugs(t *testing.T) {
	f := newFixture(t)
	received := f.captureCapitals()

	f.repo.EXPECT().BySlug("sao-paulo").Return(saoPaulo, true)

	if _, err := f.useCase.Execute(context.Background(), []string{"", "   ", "sao-paulo"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(*received) != 1 {
		t.Errorf("blank slugs should be dropped, got %v", slugsOf(*received))
	}
}

func TestExecuteUnknownCapital(t *testing.T) {
	f := newFixture(t)

	f.repo.EXPECT().BySlug("sao-paulo").Return(saoPaulo, true)
	f.repo.EXPECT().BySlug("campinas").Return(domain.Capital{}, false)
	f.repo.EXPECT().BySlug("xyz").Return(domain.Capital{}, false)
	f.repo.EXPECT().All().Return([]domain.Capital{curitiba, saoPaulo})
	// No provider expectation: gomock fails the test if it is called anyway.

	_, err := f.useCase.Execute(context.Background(), []string{"sao-paulo", "campinas", "xyz"})

	var unknown *domain.UnknownCapitalsError
	if !errors.As(err, &unknown) {
		t.Fatalf("expected *domain.UnknownCapitalsError, got %v", err)
	}
	if !errors.Is(err, domain.ErrCapitalNotFound) {
		t.Error("error should match ErrCapitalNotFound")
	}
	if len(unknown.Slugs) != 2 || unknown.Slugs[0] != "campinas" || unknown.Slugs[1] != "xyz" {
		t.Errorf("Slugs = %v, want [campinas xyz]", unknown.Slugs)
	}
	if len(unknown.ValidSlugs) != 2 {
		t.Errorf("ValidSlugs = %v, want the 2 known slugs", unknown.ValidSlugs)
	}
}

func TestExecuteRejectsOversizedRequests(t *testing.T) {
	f := newFixture(t)

	slugs := make([]string, 51)
	for i := range slugs {
		slugs[i] = "sao-paulo"
	}

	_, err := f.useCase.Execute(context.Background(), slugs)

	var invalid *domain.InvalidRequestError
	if !errors.As(err, &invalid) {
		t.Fatalf("expected *domain.InvalidRequestError, got %v", err)
	}
}

func TestExecuteWrapsProviderFailure(t *testing.T) {
	f := newFixture(t)
	upstream := errors.New("connection refused")

	f.repo.EXPECT().All().Return([]domain.Capital{saoPaulo})
	f.provider.EXPECT().CurrentWeather(gomock.Any(), gomock.Any()).Return(nil, upstream)

	_, err := f.useCase.Execute(context.Background(), nil)

	if !errors.Is(err, domain.ErrWeatherUnavailable) {
		t.Errorf("expected ErrWeatherUnavailable, got %v", err)
	}
	if !errors.Is(err, upstream) {
		t.Errorf("original cause should stay reachable, got %v", err)
	}
}

func TestExecutePropagatesContextToProvider(t *testing.T) {
	f := newFixture(t)

	type ctxKey string
	ctx := context.WithValue(context.Background(), ctxKey("request-id"), "abc-123")

	f.repo.EXPECT().All().Return([]domain.Capital{saoPaulo})
	f.provider.EXPECT().
		CurrentWeather(gomock.Any(), gomock.Any()).
		DoAndReturn(func(got context.Context, _ []domain.Capital) ([]domain.CurrentWeather, error) {
			if value, _ := got.Value(ctxKey("request-id")).(string); value != "abc-123" {
				t.Errorf("context not propagated to provider, got %v", got)
			}
			return nil, nil
		})
	f.clock.EXPECT().Now().Return(frozenNow)

	if _, err := f.useCase.Execute(ctx, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
