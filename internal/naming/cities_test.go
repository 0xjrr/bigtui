package naming

import (
	"slices"
	"testing"
)

func TestCitiesFitTabNameLimit(t *testing.T) {
	for _, city := range Cities {
		if len(city) > 12 {
			t.Fatalf("city %q exceeds the tab-safe name limit", city)
		}
		if city == "" {
			t.Fatal("city names must not be empty")
		}
	}
}

func TestCitiesAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, city := range Cities {
		if seen[city] {
			t.Fatalf("duplicate city name: %q", city)
		}
		seen[city] = true
	}
}

func TestRandomCityComesFromTheCatalog(t *testing.T) {
	for index := 0; index < 50; index++ {
		if city := RandomCity(); !slices.Contains(Cities, city) {
			t.Fatalf("random city %q is not part of the catalog", city)
		}
	}
}
