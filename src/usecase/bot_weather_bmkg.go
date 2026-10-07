package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var bmkgAdm4Pattern = regexp.MustCompile(`^\d{2}\.\d{2}\.\d{2}\.\d{4}$`)

var bmkgCityDirectory = map[string]string{
	"jakarta":           "31.71.01.1001",
	"jakarta pusat":     "31.71.01.1001",
	"gambir":            "31.71.01.1001",
	"kemayoran":         "31.71.03.1001",
	"jakarta selatan":   "31.74.04.1001",
	"kebayoran":         "31.74.04.1001",
	"jakarta barat":     "31.73.02.1001",
	"jakarta timur":     "31.75.03.1001",
	"jakarta utara":     "31.72.02.1001",
	"bandung":           "32.73.01.1001",
	"bogor":             "32.71.01.1001",
	"depok":             "32.76.01.1001",
	"bekasi":            "32.75.01.1001",
	"tangerang":         "36.71.01.1001",
	"tangsel":           "36.74.01.1001",
	"tangerang selatan": "36.74.01.1001",
	"cirebon":           "32.74.01.1001",
	"sukabumi":          "32.72.01.1001",
	"tasikmalaya":       "32.78.01.1001",
	"semarang":          "33.74.01.1001",
	"solo":              "33.72.01.1001",
	"surakarta":         "33.72.01.1001",
	"magelang":          "33.71.01.1001",
	"pekalongan":        "33.75.01.1001",
	"tegal":             "33.76.01.1001",
	"yogyakarta":        "34.71.01.1001",
	"jogja":             "34.71.01.1001",
	"surabaya":          "35.78.01.1001",
	"malang":            "35.73.01.1001",
	"kediri":            "35.71.01.1001",
	"blitar":            "35.72.01.1001",
	"madiun":            "35.77.01.1001",
	"probolinggo":       "35.74.01.1001",
	"pasuruan":          "35.75.01.1001",
	"denpasar":          "51.71.01.1001",
	"bali":              "51.71.01.1001",
	"mataram":           "52.71.01.1001",
	"lombok":            "52.71.01.1001",
	"kupang":            "53.71.01.1001",
	"banda aceh":        "11.71.01.1001",
	"aceh":              "11.71.01.1001",
	"medan":             "12.71.01.1001",
	"padang":            "13.71.01.1001",
	"pekanbaru":         "14.71.01.1001",
	"riau":              "14.71.01.1001",
	"batam":             "21.71.01.1001",
	"tanjung pinang":    "21.72.01.1001",
	"jambi":             "15.71.01.1001",
	"palembang":         "16.71.01.1001",
	"bengkulu":          "17.71.01.1001",
	"bandar lampung":    "18.71.01.1001",
	"lampung":           "18.71.01.1001",
	"pangkal pinang":    "19.71.01.1001",
	"pontianak":         "61.71.01.1001",
	"banjarmasin":       "63.71.01.1001",
	"banjarbaru":        "63.72.01.1001",
	"samarinda":         "64.71.01.1001",
	"balikpapan":        "64.71.02.1001",
	"palangkaraya":      "62.71.01.1001",
	"tarakan":           "65.71.01.1001",
	"manado":            "71.71.01.1001",
	"palu":              "72.71.01.1001",
	"makassar":          "73.71.01.1001",
	"kendari":           "74.71.01.1001",
	"gorontalo":         "75.71.01.1001",
	"mamuju":            "76.02.01.1001",
	"ambon":             "81.71.01.1001",
	"ternate":           "82.71.01.1001",
	"jayapura":          "91.71.01.1001",
	"sorong":            "92.71.01.1001",
	"manokwari":         "92.02.01.1001",
	"merauke":           "93.01.01.1001",
}

type bmkgLocation struct {
	Adm1      string `json:"adm1"`
	Adm2      string `json:"adm2"`
	Adm3      string `json:"adm3"`
	Adm4      string `json:"adm4"`
	Provinsi  string `json:"provinsi"`
	Kotkab    string `json:"kotkab"`
	Kecamatan string `json:"kecamatan"`
	Desa      string `json:"desa"`
	Timezone  string `json:"timezone"`
}

type bmkgWeatherSlot struct {
	Datetime      string  `json:"datetime"`
	T             float64 `json:"t"`
	HU            float64 `json:"hu"`
	WeatherDesc   string  `json:"weather_desc"`
	WeatherDescEn string  `json:"weather_desc_en"`
	WS            float64 `json:"ws"`
	WD            string  `json:"wd"`
	TCC           float64 `json:"tcc"`
	LocalDatetime string  `json:"local_datetime"`
}

type bmkgApiResponse struct {
	Lokasi bmkgLocation `json:"lokasi"`
	Data   []struct {
		Lokasi bmkgLocation        `json:"lokasi"`
		Cuaca  [][]bmkgWeatherSlot `json:"cuaca"`
	} `json:"data"`
}

func ResolveBMKGAdm4(query string) (string, string) {
	q := strings.TrimSpace(query)
	if q == "" {
		return "31.71.01.1001", "Jakarta"
	}
	if bmkgAdm4Pattern.MatchString(q) {
		return q, q
	}
	lower := strings.ToLower(q)
	lower = strings.TrimPrefix(lower, "kota ")
	lower = strings.TrimPrefix(lower, "kab. ")
	lower = strings.TrimPrefix(lower, "kabupaten ")
	lower = strings.TrimSpace(lower)

	if code, ok := bmkgCityDirectory[lower]; ok {
		return code, q
	}

	for k, v := range bmkgCityDirectory {
		if strings.Contains(k, lower) || strings.Contains(lower, k) {
			return v, q
		}
	}

	return "", q
}

func FetchBMKGWeather(ctx context.Context, query string) (string, error) {
	adm4Code, displayCity := ResolveBMKGAdm4(query)
	if adm4Code == "" {
		return fmt.Sprintf("⚠️ Wilayah *%s* belum ditemukan dalam direktori BMKG.\n\nContoh penggunaan:\n• *!cuaca Jakarta*\n• *!cuaca Bandung*\n• *!cuaca Surabaya*\n• *!cuaca Medan*\n• Atau kode adm4: *!cuaca 31.71.01.1001*\n\n_Sumber: data.bmkg.go.id_", displayCity), nil
	}

	url := fmt.Sprintf("https://api.bmkg.go.id/publik/prakiraan-cuaca?adm4=%s", adm4Code)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "GoWA-Weather-Bot/1.0 (+https://data.bmkg.go.id)")

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status code BMKG: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var parsed bmkgApiResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}

	if len(parsed.Data) == 0 || len(parsed.Data[0].Cuaca) == 0 || len(parsed.Data[0].Cuaca[0]) == 0 {
		return fmt.Sprintf("⚠️ Data prakiraan cuaca BMKG untuk *%s* sedang diperbarui.\n_Sumber: data.bmkg.go.id_", displayCity), nil
	}

	loc := parsed.Lokasi
	slots := parsed.Data[0].Cuaca[0]
	current := slots[0]

	locHeader := loc.Kotkab
	if loc.Kecamatan != "" && loc.Kecamatan != loc.Kotkab {
		locHeader += fmt.Sprintf(", Kec. %s", loc.Kecamatan)
	}
	if loc.Provinsi != "" {
		locHeader += fmt.Sprintf(" (%s)", loc.Provinsi)
	}

	var sb strings.Builder
	sb.WriteString("🌤️ *PRAKIRAAN CUACA BMKG*\n")
	sb.WriteString(fmt.Sprintf("📍 *Lokasi:* %s\n", locHeader))
	if current.LocalDatetime != "" {
		sb.WriteString(fmt.Sprintf("⏱️ *Waktu:* %s\n", current.LocalDatetime))
	}
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("🌡️ *Suhu:* %.0f°C\n", current.T))
	sb.WriteString(fmt.Sprintf("💧 *Kelembapan:* %.0f%%\n", current.HU))
	sb.WriteString(fmt.Sprintf("☁️ *Kondisi:* %s\n", current.WeatherDesc))
	sb.WriteString(fmt.Sprintf("💨 *Angin:* %.1f km/jam (%s)\n", current.WS, current.WD))
	sb.WriteString(fmt.Sprintf("☁️ *Tutupan Awan:* %.0f%%\n", current.TCC))

	if len(slots) > 1 {
		sb.WriteString("\n*Prakiraan Waktu Berikutnya:*\n")
		limit := 3
		if len(slots) < limit {
			limit = len(slots)
		}
		for i := 1; i < limit; i++ {
			s := slots[i]
			timeStr := s.LocalDatetime
			if len(timeStr) >= 16 {
				timeStr = timeStr[11:16]
			}
			sb.WriteString(fmt.Sprintf("• %s : %s (%.0f°C)\n", timeStr, s.WeatherDesc, s.T))
		}
	}

	sb.WriteString("\n_Sumber: BMKG (Badan Meteorologi, Klimatologi, dan Geofisika)_")

	return sb.String(), nil
}
