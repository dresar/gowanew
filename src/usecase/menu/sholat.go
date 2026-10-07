package menu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type myQuranCitySearchResponse struct {
	Status bool `json:"status"`
	Data   []struct {
		ID     string `json:"id"`
		Lokasi string `json:"lokasi"`
	} `json:"data"`
}

type myQuranScheduleResponse struct {
	Status bool `json:"status"`
	Data   struct {
		Lokasi string `json:"lokasi"`
		Daerah string `json:"daerah"`
		Jadwal struct {
			Tanggal string `json:"tanggal"`
			Imsak   string `json:"imsak"`
			Subuh   string `json:"subuh"`
			Terbit  string `json:"terbit"`
			Dhuha   string `json:"dhuha"`
			Dzuhur  string `json:"dzuhur"`
			Ashar   string `json:"ashar"`
			Maghrib string `json:"maghrib"`
			Isya    string `json:"isya"`
		} `json:"jadwal"`
	} `json:"data"`
}

func FetchPrayerTimes(ctx context.Context, city string) (string, error) {
	cityClean := strings.TrimSpace(city)
	if cityClean == "" {
		cityClean = "jakarta"
	}

	searchURL := fmt.Sprintf("https://api.myquran.com/v2/sholat/kota/cari/%s", url.PathEscape(cityClean))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
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

	var searchRes myQuranCitySearchResponse
	if err := json.Unmarshal(body, &searchRes); err != nil || len(searchRes.Data) == 0 {
		return fmt.Sprintf("⚠️ Kota/Kabupaten *%s* tidak ditemukan di database jadwal sholat Kemenag.", cityClean), nil
	}

	cityID := searchRes.Data[0].ID
	now := time.Now()
	schedURL := fmt.Sprintf("https://api.myquran.com/v2/sholat/jadwal/%s/%04d/%02d/%02d", cityID, now.Year(), now.Month(), now.Day())

	reqSched, err := http.NewRequestWithContext(ctx, http.MethodGet, schedURL, nil)
	if err != nil {
		return "", err
	}
	respSched, err := GetFastClient().Do(reqSched)
	if err != nil {
		return "", err
	}
	defer respSched.Body.Close()

	bodySched, err := io.ReadAll(respSched.Body)
	if err != nil {
		return "", err
	}

	var schedRes myQuranScheduleResponse
	if err := json.Unmarshal(bodySched, &schedRes); err != nil || !schedRes.Status {
		return fmt.Sprintf("⚠️ Gagal mengambil jadwal sholat untuk *%s*.", cityClean), nil
	}

	j := schedRes.Data.Jadwal
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🕌 *JADWAL SHOLAT %s*\n", schedRes.Data.Lokasi))
	sb.WriteString(fmt.Sprintf("📅 %s\n\n", j.Tanggal))
	sb.WriteString(fmt.Sprintf("• Imsak:   %s WIB\n", j.Imsak))
	sb.WriteString(fmt.Sprintf("• Subuh:   %s WIB\n", j.Subuh))
	sb.WriteString(fmt.Sprintf("• Terbit:  %s WIB\n", j.Terbit))
	sb.WriteString(fmt.Sprintf("• Dhuha:   %s WIB\n", j.Dhuha))
	sb.WriteString(fmt.Sprintf("• Dzuhur:  %s WIB\n", j.Dzuhur))
	sb.WriteString(fmt.Sprintf("• Ashar:   %s WIB\n", j.Ashar))
	sb.WriteString(fmt.Sprintf("• Maghrib: %s WIB\n", j.Maghrib))
	sb.WriteString(fmt.Sprintf("• Isya:    %s WIB\n\n", j.Isya))
	sb.WriteString("_Sumber: Bimas Islam Kementerian Agama RI_")

	return sb.String(), nil
}
