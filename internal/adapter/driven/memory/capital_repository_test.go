package memory_test

import (
	"testing"

	"github.com/matheus-aicorp/sample-test-project/internal/adapter/driven/memory"
	"github.com/matheus-aicorp/sample-test-project/internal/domain"
)

func TestAllReturnsEveryCapitalAlphabetically(t *testing.T) {
	repo := memory.NewCapitalRepository()
	all := repo.All()

	if len(all) != 27 {
		t.Fatalf("All() returned %d capitals, want 27 (26 states + DF)", len(all))
	}

	for i := 1; i < len(all); i++ {
		if all[i-1].Name > all[i].Name {
			t.Errorf("capitals not alphabetical: %q before %q", all[i-1].Name, all[i].Name)
		}
	}
}

func TestAllReturnsACopy(t *testing.T) {
	repo := memory.NewCapitalRepository()

	first := repo.All()
	first[0].Name = "corrupted"
	first[0].Slug = "corrupted"

	if repo.All()[0].Name == "corrupted" {
		t.Error("All() leaked internal state; callers can corrupt the dataset")
	}
}

func TestDatasetIntegrity(t *testing.T) {
	repo := memory.NewCapitalRepository()

	seenSlugs := make(map[string]struct{}, 27)
	seenStateCodes := make(map[string]struct{}, 27)

	for _, capital := range repo.All() {
		if got := domain.NormalizeSlug(capital.Slug); got != capital.Slug {
			t.Errorf("%s: slug %q is not canonical, want %q", capital.Name, capital.Slug, got)
		}
		if _, duplicate := seenSlugs[capital.Slug]; duplicate {
			t.Errorf("duplicate slug %q", capital.Slug)
		}
		seenSlugs[capital.Slug] = struct{}{}

		if len(capital.StateCode) != 2 {
			t.Errorf("%s: state code %q must have 2 chars", capital.Name, capital.StateCode)
		}
		if _, duplicate := seenStateCodes[capital.StateCode]; duplicate {
			t.Errorf("duplicate state code %q", capital.StateCode)
		}
		seenStateCodes[capital.StateCode] = struct{}{}

		if capital.Latitude < -34 || capital.Latitude > 6 {
			t.Errorf("%s: latitude %.4f is outside Brazil", capital.Name, capital.Latitude)
		}
		if capital.Longitude < -74 || capital.Longitude > -34 {
			t.Errorf("%s: longitude %.4f is outside Brazil", capital.Name, capital.Longitude)
		}
		if capital.Name == "" {
			t.Errorf("capital with slug %q has an empty name", capital.Slug)
		}
	}
}

func TestBySlug(t *testing.T) {
	repo := memory.NewCapitalRepository()

	t.Run("known slug", func(t *testing.T) {
		capital, ok := repo.BySlug("sao-paulo")
		if !ok {
			t.Fatal("expected sao-paulo to be found")
		}
		if capital.Name != "São Paulo" || capital.StateCode != "SP" {
			t.Errorf("got %+v", capital)
		}
	})

	t.Run("every listed capital is resolvable by its own slug", func(t *testing.T) {
		for _, capital := range repo.All() {
			if _, ok := repo.BySlug(capital.Slug); !ok {
				t.Errorf("BySlug(%q) not found", capital.Slug)
			}
		}
	})

	t.Run("unknown slug", func(t *testing.T) {
		if _, ok := repo.BySlug("campinas"); ok {
			t.Error("campinas is not a state capital and must not resolve")
		}
	})
}
