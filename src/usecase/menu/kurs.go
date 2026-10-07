package menu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type exchangeRateResponse struct {
	Result   string             `json:"result"`
	BaseCode string             `json:"base_code"`
	Rates    map[string]float64 `json:"rates"`
}

func FetchCurrencyExchange(ctx context.Context, amountStr, from, to string) (string, error) {
	amount := 1.0
	if parsed, err := strconv.ParseFloat(strings.TrimSpace(amountStr), 64); err == nil && parsed > 0 {
		amount = parsed
	}

	fromUpper := strings.ToUpper(strings.TrimSpace(from))
	toUpper := strings.ToUpper(strings.TrimSpace(to))
	if fromUpper == "" {
		fromUpper = "USD"
	}
	if toUpper == "" {
		toUpper = "IDR"
	}

	endpoint := fmt.Sprintf("https://open.er-api.com/v6/latest/%s", fromUpper)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}

	resp, err := GetFastClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var data exchangeRateResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}

	rate, ok := data.Rates[toUpper]
	if !ok {
		return fmt.Sprintf("⚠️ Mata uang tujuan *%s* tidak dikenali.", toUpper), nil
	}

	converted := amount * rate
	var sb strings.Builder
	sb.WriteString("💵 *KONVERSI KURS VALAS*\n\n")
	sb.WriteString(fmt.Sprintf("%.2f %s = *%.2f %s*\n\n", amount, fromUpper, converted, toUpper))
	sb.WriteString(fmt.Sprintf("_Kurs spot: 1 %s = %.4f %s_", fromUpper, rate, toUpper))

	return sb.String(), nil
}
