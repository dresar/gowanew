package menu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type cryptoSimpleResponse map[string]map[string]float64

func FetchCryptoPrice(ctx context.Context, coin string) (string, error) {
	coinClean := strings.ToLower(strings.TrimSpace(coin))
	if coinClean == "" {
		coinClean = "bitcoin"
	}

	aliases := map[string]string{
		"btc":  "bitcoin",
		"eth":  "ethereum",
		"sol":  "solana",
		"bnb":  "binancecoin",
		"xrp":  "ripple",
		"doge": "dogecoin",
		"ada":  "cardano",
		"trx":  "tron",
		"ton":  "the-open-network",
		"dot":  "polkadot",
	}
	if id, ok := aliases[coinClean]; ok {
		coinClean = id
	}

	endpoint := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=idr,usd&include_24hr_change=true", coinClean)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "GoWA-Bot/1.0")

	resp, err := GetFastClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var data cryptoSimpleResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}

	coinData, ok := data[coinClean]
	if !ok {
		return fmt.Sprintf("⚠️ Kripto *%s* belum ditemukan di CoinGecko.", coin), nil
	}

	idrVal := coinData["idr"]
	usdVal := coinData["usd"]
	change24h := coinData["usd_24h_change"]

	symbol := "📈"
	if change24h < 0 {
		symbol = "📉"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🪙 *HARGA KRIPTO: %s*\n\n", strings.ToUpper(coinClean)))
	sb.WriteString(fmt.Sprintf("• *IDR:* Rp %.0f\n", idrVal))
	sb.WriteString(fmt.Sprintf("• *USD:* $%.2f\n", usdVal))
	sb.WriteString(fmt.Sprintf("• *Perubahan 24 Jam:* %s %.2f%%\n\n", symbol, change24h))
	sb.WriteString("_Sumber: CoinGecko Market Data_")

	return sb.String(), nil
}
