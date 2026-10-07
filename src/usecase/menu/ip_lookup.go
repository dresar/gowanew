package menu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type ipApiResponse struct {
	Status      string  `json:"status"`
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	RegionName  string  `json:"regionName"`
	City        string  `json:"city"`
	Zip         string  `json:"zip"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Timezone    string  `json:"timezone"`
	ISP         string  `json:"isp"`
	Org         string  `json:"org"`
	AS          string  `json:"as"`
	Query       string  `json:"query"`
	Message     string  `json:"message"`
}

func FetchIPLookup(ctx context.Context, ip string) (string, error) {
	cleanIP := strings.TrimSpace(ip)
	if cleanIP == "" {
		return "⚠️ Masukkan alamat IP yang ingin diperiksa.\nContoh: *!ip 103.253.213.185*", nil
	}

	endpoint := fmt.Sprintf("http://ip-api.com/json/%s", url.PathEscape(cleanIP))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}

	resp, err := GetFastClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var data ipApiResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}

	if data.Status != "success" {
		return fmt.Sprintf("⚠️ IP *%s* tidak valid atau gagal dideteksi (%s).", cleanIP, data.Message), nil
	}

	var sb strings.Builder
	sb.WriteString("🌐 *INFORMASI IP GEOLOCATION*\n\n")
	sb.WriteString(fmt.Sprintf("• *IP:* %s\n", data.Query))
	sb.WriteString(fmt.Sprintf("• *Negara:* %s (%s)\n", data.Country, data.CountryCode))
	sb.WriteString(fmt.Sprintf("• *Provinsi / Kota:* %s, %s\n", data.RegionName, data.City))
	sb.WriteString(fmt.Sprintf("• *ISP:* %s\n", data.ISP))
	sb.WriteString(fmt.Sprintf("• *Organisasi:* %s\n", data.Org))
	sb.WriteString(fmt.Sprintf("• *AS:* %s\n", data.AS))
	sb.WriteString(fmt.Sprintf("• *Zona Waktu:* %s\n", data.Timezone))
	sb.WriteString(fmt.Sprintf("• *Koordinat:* %.4f, %.4f\n", data.Lat, data.Lon))

	return sb.String(), nil
}
