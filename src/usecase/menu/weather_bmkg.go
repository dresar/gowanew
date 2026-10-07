package menu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
)

var (
	bmkgAdm4Pattern = regexp.MustCompile(`^\d{2}\.\d{2}\.\d{2}\.\d{4}$`)
	adm4Cache       sync.Map

	bmkgCityDirectory = map[string]string{
		"torganda":                 "12.22.03.2011",
		"desa torganda":            "12.22.03.2011",
		"torgamba":                 "12.22.03.2009",
		"kecamatan torgamba":       "12.22.03.2009",
		"bagan baru":               "12.19.12.2011",
		"desa bagan baru":          "12.19.12.2011",
		"rokan hulu":               "14.06.03.1001",
		"kabupaten rokan hulu":     "14.06.03.1001",
		"pasir pengaraian":         "14.06.03.1001",
		"pasir pengarayan":         "14.06.03.1001",
		"rambah":                   "14.06.03.1001",
		"jakarta":                  "31.71.01.1001",
		"jakarta pusat":            "31.71.01.1001",
		"gambir":                   "31.71.01.1001",
		"kemayoran":                "31.71.03.1001",
		"jakarta selatan":          "31.74.04.1001",
		"kebayoran":                "31.74.04.1001",
		"jakarta barat":            "31.73.02.1001",
		"jakarta timur":            "31.75.03.1001",
		"jakarta utara":            "31.72.02.1001",
		"bandung":                  "32.73.01.1001",
		"bogor":                    "32.71.01.1001",
		"depok":                    "32.76.01.1001",
		"bekasi":                   "32.75.01.1001",
		"tangerang":                "36.71.01.1001",
		"tangsel":                  "36.74.01.1001",
		"tangerang selatan":        "36.74.01.1001",
		"cirebon":                  "32.74.01.1001",
		"sukabumi":                 "32.72.01.1001",
		"tasikmalaya":              "32.78.01.1001",
		"semarang":                 "33.74.01.1001",
		"solo":                     "33.72.01.1001",
		"surakarta":                "33.72.01.1001",
		"magelang":                 "33.71.01.1001",
		"pekalongan":               "33.75.01.1001",
		"tegal":                    "33.76.01.1001",
		"yogyakarta":               "34.71.01.1001",
		"jogja":                    "34.71.01.1001",
		"surabaya":                 "35.78.01.1001",
		"malang":                   "35.73.01.1001",
		"kediri":                   "35.71.01.1001",
		"blitar":                   "35.72.01.1001",
		"madiun":                   "35.77.01.1001",
		"probolinggo":              "35.74.01.1001",
		"pasuruan":                 "35.75.01.1001",
		"denpasar":                 "51.71.01.1001",
		"bali":                     "51.71.01.1001",
		"mataram":                  "52.71.01.1001",
		"lombok":                   "52.71.01.1001",
		"kupang":                   "53.71.01.1001",
		"banda aceh":               "11.71.01.1001",
		"aceh":                     "11.71.01.1001",
		"medan":                    "12.71.01.1001",
		"padang":                   "13.71.01.1001",
		"pekanbaru":                "14.71.01.1001",
		"riau":                     "14.71.01.1001",
		"batam":                    "21.71.01.1001",
		"tanjung pinang":           "21.72.01.1001",
		"jambi":                    "15.71.01.1001",
		"palembang":                "16.71.01.1001",
		"bengkulu":                 "17.71.01.1001",
		"bandar lampung":           "18.71.01.1001",
		"lampung":                  "18.71.01.1001",
		"pangkal pinang":           "19.71.01.1001",
		"pontianak":                "61.71.01.1001",
		"banjarmasin":              "63.71.01.1001",
		"banjarbaru":               "63.72.01.1001",
		"samarinda":                "64.71.01.1001",
		"balikpapan":               "64.71.02.1001",
		"palangkaraya":             "62.71.01.1001",
		"tarakan":                  "65.71.01.1001",
		"manado":                   "71.71.01.1001",
		"palu":                     "72.71.01.1001",
		"makassar":                 "73.71.01.1001",
		"kendari":                  "74.71.01.1001",
		"gorontalo":                "75.71.01.1001",
		"mamuju":                   "76.02.01.1001",
		"ambon":                    "81.71.01.1001",
		"ternate":                  "82.71.01.1001",
		"jayapura":                 "91.71.01.1001",
		"sorong":                   "92.71.01.1001",
		"manokwari":                "92.02.01.1001",
		"merauke":                  "93.01.01.1001",
		"labuhanbatu selatan":      "12.22.01.1001",
		"labuhanbatu":              "12.10.01.1001",
		"labuhanbatu utara":        "12.23.01.1001",
		"asahan":                   "12.09.01.1001",
		"batu bara":                "12.19.01.1001",
		"deli serdang":             "12.07.01.1001",
		"langkat":                  "12.05.01.1001",
		"simalungun":               "12.08.01.1001",
		"karo":                     "12.06.01.1001",
		"dairi":                    "12.11.01.1001",
		"toba":                     "12.12.01.1001",
		"samosir":                  "12.17.01.1001",
		"serdang bedagai":          "12.18.01.1001",
		"binjai":                   "12.75.01.1001",
		"tebing tinggi":            "12.76.01.1001",
		"pematang siantar":         "12.72.01.1001",
		"siantar":                  "12.72.01.1001",
		"sibolga":                  "12.73.01.1001",
		"tanjung balai":            "12.74.01.1001",
		"padangsidimpuan":          "12.77.01.1001",
		"gunungsitoli":             "12.78.01.1001",
		"serang":                   "36.73.01.1001",
		"cilegon":                  "36.72.01.1001",
		"karawang":                 "32.15.01.1001",
		"purwakarta":               "32.14.01.1001",
		"subang":                   "32.13.01.1001",
		"garut":                    "32.05.01.1001",
		"ciamis":                   "32.07.01.1001",
		"kuningan":                 "32.08.01.1001",
		"majalengka":               "32.10.01.1001",
		"indramayu":                "32.12.01.1001",
		"sumedang":                 "32.11.01.1001",
		"cianjur":                  "32.03.01.1001",
		"banyumas":                 "33.02.01.1001",
		"purwokerto":               "33.02.01.1001",
		"cilacap":                  "33.01.01.1001",
		"kebumen":                  "33.05.01.1001",
		"purworejo":                "33.06.01.1001",
		"wonosobo":                 "33.07.01.1001",
		"klaten":                   "33.10.01.1001",
		"sukoharjo":                "33.11.01.1001",
		"wonogiri":                 "33.12.01.1001",
		"karanganyar":              "33.13.01.1001",
		"sragen":                   "33.14.01.1001",
		"grobogan":                 "33.15.01.1001",
		"blora":                    "33.16.01.1001",
		"rembang":                  "33.17.01.1001",
		"pati":                     "33.18.01.1001",
		"kudus":                    "33.19.01.1001",
		"jepara":                   "33.20.01.1001",
		"demak":                    "33.21.01.1001",
		"kendal":                   "33.24.01.1001",
		"batang":                   "33.25.01.1001",
		"pemalang":                 "33.27.01.1001",
		"brebes":                   "33.29.01.1001",
		"salatiga":                 "33.73.01.1001",
		"banyuwangi":               "35.10.01.1001",
		"jember":                   "35.09.01.1001",
		"lumajang":                 "35.08.01.1001",
		"bondowoso":                "35.11.01.1001",
		"situbondo":                "35.12.01.1001",
		"sidoarjo":                 "35.15.01.1001",
		"mojokerto":                "35.76.01.1001",
		"jombang":                  "35.17.01.1001",
		"nganjuk":                  "35.18.01.1001",
		"trenggalek":               "35.03.01.1001",
		"tulungagung":              "35.04.01.1001",
		"ponorogo":                 "35.02.01.1001",
		"pacitan":                  "35.01.01.1001",
		"magetan":                  "35.20.01.1001",
		"ngawi":                    "35.21.01.1001",
		"bojonegoro":               "35.22.01.1001",
		"tuban":                    "35.23.01.1001",
		"lamongan":                 "35.24.01.1001",
		"gresik":                   "35.25.01.1001",
		"bangkalan":                "35.26.01.1001",
		"sampang":                  "35.27.01.1001",
		"pamekasan":                "35.28.01.1001",
		"sumenep":                  "35.29.01.1001",
		"batu":                     "35.79.01.1001",
	}
)

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

type onlineKodeposResponse struct {
	StatusCode int    `json:"statusCode"`
	Code       string `json:"code"`
	Data       []struct {
		Village  string `json:"village"`
		District string `json:"district"`
		Regency  string `json:"regency"`
		Province string `json:"province"`
	} `json:"data"`
}

type wilayahItem struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type wilayahGenericResponse struct {
	Data []wilayahItem `json:"data"`
}

func lookupDynamicWilayah(ctx context.Context, term string) (string, string) {
	if term == "" {
		return "", ""
	}

	searchURL := fmt.Sprintf("https://kodepos.vercel.app/search?q=%s", url.QueryEscape(term))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return "", ""
	}

	resp, err := GetFastClient().Do(req)
	if err != nil {
		return "", ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", ""
	}

	var res onlineKodeposResponse
	if err := json.Unmarshal(body, &res); err != nil || len(res.Data) == 0 {
		return "", ""
	}

	item := res.Data[0]
	targetProv := strings.ToLower(strings.TrimSpace(item.Province))
	targetReg := strings.ToLower(strings.TrimSpace(item.Regency))
	targetDist := strings.ToLower(strings.TrimSpace(item.District))
	targetVill := strings.ToLower(strings.TrimSpace(item.Village))

	provCode := resolveProvinceCode(ctx, targetProv)
	if provCode == "" {
		return "", ""
	}

	regCode := resolveRegencyCode(ctx, provCode, targetReg)
	if regCode == "" {
		return "", ""
	}

	distCode := resolveDistrictCode(ctx, regCode, targetDist)
	if distCode == "" {
		return regCode + ".01.1001", item.Regency
	}

	villCode := resolveVillageCode(ctx, distCode, targetVill)
	if villCode != "" {
		return villCode, item.Village
	}

	return distCode + ".1001", item.District
}

func resolveProvinceCode(ctx context.Context, target string) string {
	url := "https://wilayah.id/api/provinces.json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ""
	}

	resp, err := GetFastClient().Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var res wilayahGenericResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return ""
	}

	for _, p := range res.Data {
		pName := strings.ToLower(p.Name)
		if pName == target || strings.Contains(pName, target) || strings.Contains(target, pName) {
			return p.Code
		}
	}
	return ""
}

func resolveRegencyCode(ctx context.Context, provCode, target string) string {
	url := fmt.Sprintf("https://wilayah.id/api/regencies/%s.json", provCode)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ""
	}

	resp, err := GetFastClient().Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var res wilayahGenericResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return ""
	}

	cleanTarget := strings.TrimPrefix(strings.TrimPrefix(target, "kabupaten "), "kota ")
	cleanTarget = strings.TrimSpace(cleanTarget)

	for _, r := range res.Data {
		rName := strings.ToLower(r.Name)
		cleanRName := strings.TrimPrefix(strings.TrimPrefix(rName, "kabupaten "), "kota ")
		cleanRName = strings.TrimSpace(cleanRName)

		if cleanRName == cleanTarget || strings.Contains(cleanRName, cleanTarget) || strings.Contains(cleanTarget, cleanRName) {
			return r.Code
		}
	}
	return ""
}

func resolveDistrictCode(ctx context.Context, regCode, target string) string {
	url := fmt.Sprintf("https://wilayah.id/api/districts/%s.json", regCode)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ""
	}

	resp, err := GetFastClient().Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var res wilayahGenericResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return ""
	}

	cleanTarget := strings.TrimPrefix(target, "kecamatan ")
	cleanTarget = strings.TrimSpace(cleanTarget)

	for _, d := range res.Data {
		dName := strings.ToLower(d.Name)
		if dName == cleanTarget || strings.Contains(dName, cleanTarget) || strings.Contains(cleanTarget, dName) {
			return d.Code
		}
	}
	return ""
}

func resolveVillageCode(ctx context.Context, distCode, target string) string {
	url := fmt.Sprintf("https://wilayah.id/api/villages/%s.json", distCode)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ""
	}

	resp, err := GetFastClient().Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var res wilayahGenericResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return ""
	}

	cleanTarget := strings.TrimPrefix(strings.TrimPrefix(target, "desa "), "kelurahan ")
	cleanTarget = strings.TrimSpace(cleanTarget)

	for _, v := range res.Data {
		vName := strings.ToLower(v.Name)
		if vName == cleanTarget || strings.Contains(vName, cleanTarget) || strings.Contains(cleanTarget, vName) {
			return v.Code
		}
	}
	return ""
}

func ResolveBMKGAdm4(ctx context.Context, query string) (string, string) {
	q := strings.TrimSpace(query)
	if q == "" {
		return "31.71.01.1001", "Jakarta"
	}
	if bmkgAdm4Pattern.MatchString(q) {
		return q, q
	}
	lower := strings.ToLower(q)
	cleaned := lower
	prefixes := []string{
		"desa ", "kelurahan ", "kel. ", "kecamatan ", "kec. ",
		"kota ", "kabupaten ", "kab. ", "wilayah ",
	}
	for _, p := range prefixes {
		cleaned = strings.TrimPrefix(cleaned, p)
	}
	cleaned = strings.TrimSpace(cleaned)

	if val, ok := adm4Cache.Load(cleaned); ok {
		if pair, ok2 := val.([2]string); ok2 {
			return pair[0], pair[1]
		}
	}
	if val, ok := adm4Cache.Load(lower); ok {
		if pair, ok2 := val.([2]string); ok2 {
			return pair[0], pair[1]
		}
	}

	if code, ok := bmkgCityDirectory[cleaned]; ok {
		return code, q
	}
	if code, ok := bmkgCityDirectory[lower]; ok {
		return code, q
	}

	for k, v := range bmkgCityDirectory {
		if k == cleaned || k == lower {
			return v, q
		}
	}

	for k, v := range bmkgCityDirectory {
		if strings.Contains(k, cleaned) || strings.Contains(cleaned, k) {
			return v, q
		}
	}

	onlineCode, onlineName := lookupDynamicWilayah(ctx, cleaned)
	if onlineCode == "" && cleaned != lower {
		onlineCode, onlineName = lookupDynamicWilayah(ctx, lower)
	}
	if onlineCode != "" {
		adm4Cache.Store(cleaned, [2]string{onlineCode, onlineName})
		adm4Cache.Store(lower, [2]string{onlineCode, onlineName})
		return onlineCode, onlineName
	}

	return "", q
}

func FetchBMKGWeather(ctx context.Context, query string) (string, error) {
	adm4Code, displayCity := ResolveBMKGAdm4(ctx, query)
	if adm4Code == "" {
		return fmt.Sprintf("⚠️ Wilayah *%s* belum ditemukan dalam direktori BMKG.\n\nContoh pencarian bebas:\n• Desa: *!cuaca Torganda*, *!cuaca Bagan Baru*\n• Kecamatan: *!cuaca Torgamba*, *!cuaca Kemayoran*\n• Kota / Kab: *!cuaca Medan*, *!cuaca Jakarta*, *!cuaca Rokan Hulu*\n\n_Sumber: data.bmkg.go.id_", displayCity), nil
	}

	endpoint := fmt.Sprintf("https://api.bmkg.go.id/publik/prakiraan-cuaca?adm4=%s", adm4Code)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "GoWA-Weather-Bot/1.0 (+https://data.bmkg.go.id)")

	resp, err := GetStandardClient().Do(req)
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

	var locParts []string
	if loc.Desa != "" && !strings.EqualFold(loc.Desa, loc.Kecamatan) && !strings.EqualFold(loc.Desa, loc.Kotkab) {
		locParts = append(locParts, fmt.Sprintf("Desa %s", loc.Desa))
	} else if loc.Desa != "" {
		locParts = append(locParts, loc.Desa)
	}
	if loc.Kecamatan != "" && !strings.EqualFold(loc.Kecamatan, loc.Kotkab) {
		locParts = append(locParts, fmt.Sprintf("Kec. %s", loc.Kecamatan))
	}
	if loc.Kotkab != "" {
		locParts = append(locParts, loc.Kotkab)
	}
	locHeader := strings.Join(locParts, ", ")
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
