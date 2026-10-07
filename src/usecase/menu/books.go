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

type googleBooksResponse struct {
	TotalItems int `json:"totalItems"`
	Items      []struct {
		VolumeInfo struct {
			Title         string   `json:"title"`
			Authors       []string `json:"authors"`
			Publisher     string   `json:"publisher"`
			PublishedDate string   `json:"publishedDate"`
			PageCount     int      `json:"pageCount"`
			Description   string   `json:"description"`
		} `json:"volumeInfo"`
	} `json:"items"`
}

func FetchBookInfo(ctx context.Context, title string) (string, error) {
	t := strings.TrimSpace(title)
	if t == "" {
		return "⚠️ Masukkan judul buku yang ingin dicari.\nContoh: *!buku Filosofi Teras*", nil
	}

	endpoint := fmt.Sprintf("https://www.googleapis.com/books/v1/volumes?q=%s&maxResults=1", url.QueryEscape(t))
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

	var data googleBooksResponse
	if err := json.Unmarshal(body, &data); err != nil || len(data.Items) == 0 {
		return fmt.Sprintf("⚠️ Buku dengan judul *%s* tidak ditemukan di Google Books.", t), nil
	}

	v := data.Items[0].VolumeInfo
	authors := strings.Join(v.Authors, ", ")
	if authors == "" {
		authors = "Anonim"
	}

	desc := v.Description
	if len(desc) > 300 {
		desc = desc[:300] + "..."
	}
	if desc == "" {
		desc = "Sinopsis tidak tersedia."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📚 *BUKU: %s*\n\n", v.Title))
	sb.WriteString(fmt.Sprintf("• *Penulis:* %s\n", authors))
	if v.Publisher != "" {
		sb.WriteString(fmt.Sprintf("• *Penerbit:* %s\n", v.Publisher))
	}
	if v.PublishedDate != "" {
		sb.WriteString(fmt.Sprintf("• *Tahun:* %s\n", v.PublishedDate))
	}
	if v.PageCount > 0 {
		sb.WriteString(fmt.Sprintf("• *Tebal:* %d Halaman\n\n", v.PageCount))
	} else {
		sb.WriteString("\n")
	}
	sb.WriteString(fmt.Sprintf("*Sinopsis:*\n%s\n\n", desc))
	sb.WriteString("_Sumber: Google Books Library_")

	return sb.String(), nil
}
