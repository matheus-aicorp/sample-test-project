package domain_test

import (
	"errors"
	"testing"

	"github.com/matheus-aicorp/sample-test-project/internal/domain"
)

func TestUnknownCapitalsError(t *testing.T) {
	err := &domain.UnknownCapitalsError{
		Slugs:      []string{"sao-pauloo", "xyz"},
		ValidSlugs: []string{"sao-paulo"},
	}

	want := "capital not found: sao-pauloo, xyz"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}

	if !errors.Is(err, domain.ErrCapitalNotFound) {
		t.Error("error should unwrap to ErrCapitalNotFound")
	}
}

func TestInvalidRequestError(t *testing.T) {
	err := &domain.InvalidRequestError{Message: "too many capitals"}

	if got := err.Error(); got != "too many capitals" {
		t.Errorf("Error() = %q, want %q", got, "too many capitals")
	}
	if errors.Is(err, domain.ErrCapitalNotFound) {
		t.Error("InvalidRequestError must not match ErrCapitalNotFound")
	}
}
