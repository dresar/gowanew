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

	calc, err := SimpleCalculate("10 + 20")
	if err != nil || !strings.Contains(calc, "30") {
		t.Fatalf("expected 30, got: %s, err: %v", calc, err)
	}

	code, name := ResolveBMKGAdm4(context.Background(), "Jakarta")
	if code == "" || name == "" {
		t.Fatalf("expected valid code and name for Jakarta, got: %s, %s", code, name)
	}
}
