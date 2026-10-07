package menu

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func FetchShortLink(ctx context.Context, longURL string) (string, error) {
	u := strings.TrimSpace(longURL)
	if u == "" {
		return "⚠️ Masukkan tautan website yang ingin dipendekkan.\nContoh: *!short https://contoh-link-panjang.com*", nil
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		u = "https://" + u
	}

	endpoint := fmt.Sprintf("https://tinyurl.com/api-create.php?url=%s", url.QueryEscape(u))
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

	resStr := strings.TrimSpace(string(body))
	if !strings.HasPrefix(resStr, "http") {
		return "⚠️ Gagal memendekkan tautan. Pastikan link tujuan valid.", nil
	}

	return fmt.Sprintf("🔗 *TAUTAN PENDEK:*\n%s", resStr), nil
}
