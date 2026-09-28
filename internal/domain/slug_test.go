package domain_test

import (
	"testing"

	"github.com/matheus-aicorp/sample-test-project/internal/domain"
)

func TestNormalizeSlug(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "already canonical", raw: "sao-paulo", want: "sao-paulo"},
		{name: "upper case", raw: "SAO-PAULO", want: "sao-paulo"},
		{name: "mixed case", raw: "Rio-De-Janeiro", want: "rio-de-janeiro"},
		{name: "spaces become hyphens", raw: "rio de janeiro", want: "rio-de-janeiro"},
		{name: "underscores become hyphens", raw: "rio_de_janeiro", want: "rio-de-janeiro"},
		{name: "surrounding whitespace trimmed", raw: "  curitiba \t", want: "curitiba"},
		{name: "empty stays empty", raw: "   ", want: ""},
		{name: "idempotent", raw: domain.NormalizeSlug("Belo Horizonte"), want: "belo-horizonte"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := domain.NormalizeSlug(tt.raw); got != tt.want {
				t.Errorf("NormalizeSlug(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}
