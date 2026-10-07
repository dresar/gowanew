package menu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type newsResponse struct {
	Status       string `json:"status"`
	TotalResults int    `json:"totalResults"`
	Articles     []struct {
		Source struct {
			Name string `json:"name"`
		} `json:"source"`
		Title       string `json:"title"`
		Description string `json:"description"`
		URL         string `json:"url"`
	} `json:"articles"`
}

func FetchNewsHeadlines(ctx context.Context, category, apiKey string) (string, error) {
	cat := strings.TrimSpace(category)
	if cat == "" {
		cat = "general"
	}

	key := strings.TrimSpace(apiKey)
	if key == "" {
		return "📰 *BERITA TERKINI*\n\n1. Pembaruan Sistem Layanan Publik Digital Terkini.\n2. Laporan Pertumbuhan Ekonomi dan Valas Kuartal Ini.\n3. Inovasi Teknologi dan Infrastruktur Komunikasi.\n\n_Untuk siaran berita dinamis real-time, silakan atur NewsAPI Key di Pengaturan Menu._", nil
	}

	endpoint := fmt.Sprintf("https://newsapi.org/v2/top-headlines?country=id&category=%s&apiKey=%s&pageSize=4", cat, key)
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

	var data newsResponse
	if err := json.Unmarshal(body, &data); err != nil || len(data.Articles) == 0 {
		return "⚠️ Belum ada berita terkini pada kategori tersebut.", nil
	}

	var sb strings.Builder
	sb.WriteString("📰 *BERITA TERKINI INDONESIA*\n\n")
	for i, a := range data.Articles {
		sb.WriteString(fmt.Sprintf("%d. *%s*\n", i+1, a.Title))
		if a.Source.Name != "" {
			sb.WriteString(fmt.Sprintf("   _Sumber: %s_\n", a.Source.Name))
		}
		if a.URL != "" {
			sb.WriteString(fmt.Sprintf("   %s\n", a.URL))
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}
