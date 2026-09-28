package domain

import "context"

// Driven ports: contracts published by the core and implemented by outer
// adapters. The domain never learns about HTTP, Open-Meteo or storage.

// CapitalRepository resolves the capitals covered by the API.
//
// Slug keys are assumed to be in the canonical form produced by NormalizeSlug.
type CapitalRepository interface {
	// All returns every capital, alphabetically, as a copy the caller may modify.
	All() []Capital

	BySlug(slug string) (Capital, bool)
}

// WeatherProvider fetches current conditions for a set of capitals at once.
//
// Implementations must stay substitutable: return exactly one reading per
// capital in the same order, never error on valid capitals, and honour context
// cancellation. Batching is part of the contract because it is a domain
// capability ("fetch the weather for these capitals"), not a provider detail —
// a provider without batch support can fan out internally.
type WeatherProvider interface {
	CurrentWeather(ctx context.Context, capitals []Capital) ([]CurrentWeather, error)
}
