package domain

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrCapitalNotFound    = errors.New("capital not found")
	ErrWeatherUnavailable = errors.New("weather unavailable from upstream source")
)

// InvalidRequestError is a request that breaks a domain input rule.
type InvalidRequestError struct {
	Message string
}

func (e *InvalidRequestError) Error() string { return e.Message }

// UnknownCapitalsError carries the valid slugs so the HTTP layer can answer
// with an actionable message without querying the repository again.
type UnknownCapitalsError struct {
	Slugs      []string
	ValidSlugs []string
}

func (e *UnknownCapitalsError) Error() string {
	return fmt.Sprintf("%s: %s", ErrCapitalNotFound.Error(), strings.Join(e.Slugs, ", "))
}

func (e *UnknownCapitalsError) Unwrap() error { return ErrCapitalNotFound }
