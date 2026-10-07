package menu

import (
	"context"
	"strings"
	"testing"
)

func TestMenuFeatures(t *testing.T) {
	ctx := context.Background()

	fact := FetchRandomFact(ctx)
	if !strings.Contains(fact, "TAHUKAH KAMU") {
		t.Fatalf("expected fact header, got: %s", fact)
	}

	factScience := FetchRandomFact(ctx, "sains")
	if !strings.Contains(factScience, "TAHUKAH KAMU") {
		t.Fatalf("expected filtered fact, got: %s", factScience)
	}

	factAnimal := FetchRandomFact(ctx, "hewan")
	if !strings.Contains(factAnimal, "TAHUKAH KAMU") {
		t.Fatalf("expected animal fact, got: %s", factAnimal)
	}

	pantun := FetchRandomPantun()
	if !strings.Contains(pantun, "PANTUN NUSANTARA") {
		t.Fatalf("expected pantun header, got: %s", pantun)
	}

	pantunJenaka := FetchRandomPantun("jenaka")
	if !strings.Contains(pantunJenaka, "JENAKA") {
		t.Fatalf("expected jenaka pantun, got: %s", pantunJenaka)
	}

	bt := FetchRandomBrainTeaser()
	if !strings.Contains(bt, "ASAH OTAK") {
		t.Fatalf("expected brain teaser header, got: %s", bt)
	}

	btLogic := FetchRandomBrainTeaser("logika")
	if !strings.Contains(btLogic, "LOGIKA") {
		t.Fatalf("expected logic brain teaser, got: %s", btLogic)
	}

	dua := FetchDailyDua("makan")
	if !strings.Contains(dua, "MAKAN") {
		t.Fatalf("expected makan dua, got: %s", dua)
	}

	recipe := FetchRecipe("rendang")
	if !strings.Contains(recipe, "RENDANG") {
		t.Fatalf("expected rendang recipe, got: %s", recipe)
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
