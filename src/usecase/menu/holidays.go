package menu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type holidayItem struct {
	HolidayDate string `json:"holiday_date"`
	HolidayName string `json:"holiday_name"`
	IsNational  bool   `json:"is_national_holiday"`
}

func FetchNationalHolidays(ctx context.Context) (string, error) {
	endpoint := "https://api-harilibur.vercel.app/api"
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

	var items []holidayItem
	if err := json.Unmarshal(body, &items); err != nil {
		return "", err
	}

	nowStr := time.Now().Format("2006-01-02")
	var upcoming []holidayItem
	for _, it := range items {
		if it.HolidayDate >= nowStr {
			upcoming = append(upcoming, it)
			if len(upcoming) >= 6 {
				break
			}
		}
	}

	if len(upcoming) == 0 {
		return "📅 *HARI LIBUR NASIONAL*\nBelum ada tanggal merah yang terdaftar dalam waktu dekat.", nil
	}

	var sb strings.Builder
	sb.WriteString("📅 *HARI LIBUR NASIONAL MENDATANG:*\n\n")
	for _, it := range upcoming {
		sb.WriteString(fmt.Sprintf("• *%s*: %s\n", it.HolidayDate, it.HolidayName))
	}
	sb.WriteString("\n_Sumber: SKB 3 Menteri / Kalender Resmi RI_")

	return sb.String(), nil
}
