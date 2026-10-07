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

type wikipediaResponse struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Extract     string `json:"extract"`
	ContentURLs struct {
		Desktop struct {
			Page string `json:"page"`
		} `json:"desktop"`
	} `json:"content_urls"`
}

func FetchWikipediaSummary(ctx context.Context, query string) (string, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return "⚠️ Masukkan topik yang ingin dicari di Wikipedia.\nContoh: *!wiki Teori Relativitas*", nil
	}

	endpoint := fmt.Sprintf("https://id.wikipedia.org/api/rest_v1/page/summary/%s", url.PathEscape(q))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "GoWA-Bot/1.0 (admin@local)")

	resp, err := GetFastClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Sprintf("⚠️ Topik *%s* tidak ditemukan di Wikipedia bahasa Indonesia.", q), nil
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var data wikipediaResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}

	if data.Extract == "" {
		return fmt.Sprintf("⚠️ Ringkasan untuk *%s* tidak tersedia di Wikipedia.", q), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📚 *WIKIPEDIA: %s*\n", data.Title))
	if data.Description != "" {
		sb.WriteString(fmt.Sprintf("_%s_\n\n", data.Description))
	} else {
		sb.WriteString("\n")
	}
	sb.WriteString(data.Extract)
	sb.WriteString("\n\n")
	if data.ContentURLs.Desktop.Page != "" {
		sb.WriteString(fmt.Sprintf("_Baca selengkapnya: %s_", data.ContentURLs.Desktop.Page))
	}

	return sb.String(), nil
}
