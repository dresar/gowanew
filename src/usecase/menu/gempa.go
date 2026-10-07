package menu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type bmkGEarthquakeResponse struct {
	Infogempa struct {
		Gempa struct {
			Tanggal     string `json:"Tanggal"`
			Jam         string `json:"Jam"`
			DateTime    string `json:"DateTime"`
			Coordinates string `json:"Coordinates"`
			Lintang     string `json:"Lintang"`
			Bujur       string `json:"Bujur"`
			Magnitude   string `json:"Magnitude"`
			Kedalaman   string `json:"Kedalaman"`
			Wilayah     string `json:"Wilayah"`
			Potensi     string `json:"Potensi"`
			Dirasakan   string `json:"Dirasakan"`
		} `json:"gempa"`
	} `json:"Infogempa"`
}

func FetchBMKGEarthquake(ctx context.Context) (string, error) {
	url := "https://data.bmkg.go.id/DataMKG/TEWS/autogempa.json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "GoWA-Bot/1.0 (+https://data.bmkg.go.id)")

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

	var data bmkGEarthquakeResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}

	g := data.Infogempa.Gempa
	var sb strings.Builder
	sb.WriteString("🔴 *INFO GEMPA TERKINI (BMKG)*\n\n")
	sb.WriteString(fmt.Sprintf("📍 *Pusat:* %s\n", g.Wilayah))
	sb.WriteString(fmt.Sprintf("📈 *Magnitudo:* %s SR\n", g.Magnitude))
	sb.WriteString(fmt.Sprintf("🌊 *Kedalaman:* %s\n", g.Kedalaman))
	sb.WriteString(fmt.Sprintf("⏱️ *Waktu:* %s %s\n", g.Tanggal, g.Jam))
	sb.WriteString(fmt.Sprintf("🧭 *Koordinat:* %s (%s, %s)\n", g.Coordinates, g.Lintang, g.Bujur))
	if g.Dirasakan != "" {
		sb.WriteString(fmt.Sprintf("📢 *Dirasakan:* %s\n", g.Dirasakan))
	}
	sb.WriteString(fmt.Sprintf("⚠️ *Status:* %s\n\n", g.Potensi))
	sb.WriteString("_Sumber: BMKG Indonesia (TEWS)_")

	return sb.String(), nil
}
