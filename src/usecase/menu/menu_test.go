package menu

import (
	"context"
	"strings"
	"testing"
)

func TestMenuFeatures(t *testing.T) {
	pantun := FetchRandomPantun()
	if !strings.Contains(pantun, "PANTUN NUSANTARA") {
		t.Fatalf("expected pantun header, got: %s", pantun)
	}

	bt := FetchRandomBrainTeaser()
	if !strings.Contains(bt, "ASAH OTAK") {
		t.Fatalf("expected brain teaser header, got: %s", bt)
	}

	zodiac := FetchZodiac("aries")
	if !strings.Contains(zodiac, "ARIES") {
		t.Fatalf("expected aries zodiac, got: %s", zodiac)
	}

	quote := FetchRandomQuote()
	if !strings.Contains(quote, "KATA MUTIARA") {
		t.Fatalf("expected quote header, got: %s", quote)
	}

	quoteFilter := FetchRandomQuote("sukses")
	if !strings.Contains(quoteFilter, "KATA MUTIARA") {
		t.Fatalf("expected filtered quote header, got: %s", quoteFilter)
	}

	calc, err := SimpleCalculate("10 + 20")
	if err != nil || !strings.Contains(calc, "30") {
		t.Fatalf("expected 30, got: %s, err: %v", calc, err)
	}

	ctx := context.Background()
	testLocations := []string{"Jakarta", "Medan", "Torganda", "Bagan Baru", "Rokan Hulu"}
	for _, loc := range testLocations {
		code, _ := ResolveBMKGAdm4(ctx, loc)
		if code == "" {
			t.Fatalf("expected valid adm4 code for %s", loc)
		}
	}

	weather, err := FetchBMKGWeather(ctx, "Torganda")
	if err != nil {
		t.Fatalf("failed fetching weather for Torganda: %v", err)
	}
	if !strings.Contains(weather, "Torganda") || !strings.Contains(weather, "Torgamba") {
		t.Fatalf("expected weather to mention desa Torganda and kec Torgamba, got: %s", weather)
	}
}
