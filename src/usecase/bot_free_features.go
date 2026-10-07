package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
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

	client := &http.Client{Timeout: 6 * time.Second}
	resp, err := client.Do(req)
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

	client := &http.Client{Timeout: 6 * time.Second}
	resp, err := client.Do(req)
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

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
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

type exchangeRateResponse struct {
	Result   string             `json:"result"`
	BaseCode string             `json:"base_code"`
	Rates    map[string]float64 `json:"rates"`
}

func FetchCurrencyExchange(ctx context.Context, amountStr, from, to string) (string, error) {
	amount := 1.0
	if parsed, err := strconv.ParseFloat(strings.TrimSpace(amountStr), 64); err == nil && parsed > 0 {
		amount = parsed
	}

	fromUpper := strings.ToUpper(strings.TrimSpace(from))
	toUpper := strings.ToUpper(strings.TrimSpace(to))
	if fromUpper == "" {
		fromUpper = "USD"
	}
	if toUpper == "" {
		toUpper = "IDR"
	}

	endpoint := fmt.Sprintf("https://open.er-api.com/v6/latest/%s", fromUpper)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
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

	var data exchangeRateResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}

	rate, ok := data.Rates[toUpper]
	if !ok {
		return fmt.Sprintf("⚠️ Mata uang tujuan *%s* tidak dikenali.", toUpper), nil
	}

	converted := amount * rate
	var sb strings.Builder
	sb.WriteString("💵 *KONVERSI KURS VALAS*\n\n")
	sb.WriteString(fmt.Sprintf("%.2f %s = *%.2f %s*\n\n", amount, fromUpper, converted, toUpper))
	sb.WriteString(fmt.Sprintf("_Kurs spot: 1 %s = %.4f %s_", fromUpper, rate, toUpper))

	return sb.String(), nil
}

type cryptoSimpleResponse map[string]map[string]float64

func FetchCryptoPrice(ctx context.Context, coin string) (string, error) {
	coinClean := strings.ToLower(strings.TrimSpace(coin))
	if coinClean == "" {
		coinClean = "bitcoin"
	}

	aliases := map[string]string{
		"btc":  "bitcoin",
		"eth":  "ethereum",
		"sol":  "solana",
		"bnb":  "binancecoin",
		"xrp":  "ripple",
		"doge": "dogecoin",
		"ada":  "cardano",
		"trx":  "tron",
		"ton":  "the-open-network",
		"dot":  "polkadot",
	}
	if id, ok := aliases[coinClean]; ok {
		coinClean = id
	}

	endpoint := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=idr,usd&include_24hr_change=true", coinClean)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "GoWA-Bot/1.0")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var data cryptoSimpleResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}

	coinData, ok := data[coinClean]
	if !ok {
		return fmt.Sprintf("⚠️ Kripto *%s* belum ditemukan di CoinGecko.", coin), nil
	}

	idrVal := coinData["idr"]
	usdVal := coinData["usd"]
	change24h := coinData["usd_24h_change"]

	symbol := "📈"
	if change24h < 0 {
		symbol = "📉"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🪙 *HARGA KRIPTO: %s*\n\n", strings.ToUpper(coinClean)))
	sb.WriteString(fmt.Sprintf("• *IDR:* Rp %.0f\n", idrVal))
	sb.WriteString(fmt.Sprintf("• *USD:* $%.2f\n", usdVal))
	sb.WriteString(fmt.Sprintf("• *Perubahan 24 Jam:* %s %.2f%%\n\n", symbol, change24h))
	sb.WriteString("_Sumber: CoinGecko Market Data_")

	return sb.String(), nil
}

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
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
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
	respSched, err := client.Do(reqSched)
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

type equranAyatResponse struct {
	Code int `json:"code"`
	Data struct {
		Nomor      int    `json:"nomor"`
		NamaLatin  string `json:"namaLatin"`
		JumlahAyat int    `json:"jumlahAyat"`
		Ayat       []struct {
			NomorAyat     int    `json:"nomorAyat"`
			TeksArab      string `json:"teksArab"`
			TeksLatin     string `json:"teksLatin"`
			TeksIndonesia string `json:"teksIndonesia"`
		} `json:"ayat"`
	} `json:"data"`
}

func FetchQuranVerse(ctx context.Context, ref string) (string, error) {
	refClean := strings.TrimSpace(ref)
	if refClean == "" {
		return "⚠️ Masukkan nomor surah dan ayat.\nContoh: *!quran 1:1* atau *!quran 2:255*", nil
	}

	parts := strings.Split(refClean, ":")
	if len(parts) != 2 {
		return "⚠️ Format ayat salah. Gunakan format *nomor_surah:nomor_ayat*.\nContoh: *!quran 1:1*", nil
	}

	surahNum, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	ayatNum, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil || surahNum < 1 || surahNum > 114 {
		return "⚠️ Nomor surah (1-114) atau nomor ayat tidak valid.", nil
	}

	endpoint := fmt.Sprintf("https://equran.id/api/v2/surat/%d", surahNum)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 6 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var data equranAyatResponse
	if err := json.Unmarshal(body, &data); err != nil || data.Code != 200 {
		return "⚠️ Gagal mengambil data Al-Qur'an.", nil
	}

	for _, a := range data.Data.Ayat {
		if a.NomorAyat == ayatNum {
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("📖 *QS. %s : Ayat %d*\n\n", data.Data.NamaLatin, a.NomorAyat))
			sb.WriteString(fmt.Sprintf("%s\n\n", a.TeksArab))
			sb.WriteString(fmt.Sprintf("_%s_\n\n", strings.TrimSpace(a.TeksLatin)))
			sb.WriteString(fmt.Sprintf("\"%s\"\n\n", a.TeksIndonesia))
			sb.WriteString("_Sumber: Kementerian Agama RI_")
			return sb.String(), nil
		}
	}

	return fmt.Sprintf("⚠️ Surah %s hanya memiliki %d ayat.", data.Data.NamaLatin, data.Data.JumlahAyat), nil
}

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

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
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

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
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

var pantunList = []string{
	"Beli pulsa di toko Pak Rahmat,\nPulang ke rumah disambut senyuman.\nAwali hari dengan semangat,\nSemoga sukses dalam genggaman.",
	"Jalan-jalan ke Kota Blitar,\nJangan lupa membeli sukun.\nJika kamu ingin pintar,\nBelajarlah dengan tekun.",
	"Pohon beringin rindang daunnya,\nTempat berteduh sang gembala.\nBekerjalah dengan sepenuh jiwa,\nRezeki halal membawa pahala.",
	"Pergi ke pasar membeli nangka,\nNangka dibelah manis rasanya.\nTetaplah ramah kepada sesama,\nHidup bahagia damai selamanya.",
	"Burung dara terbang melayang,\nHinggap sebentar di dahan cemara.\nKepada sahabat selalu sayang,\nHati tenang tiada duka.",
}

func FetchRandomPantun() string {
	idx := rand.Intn(len(pantunList))
	return fmt.Sprintf("📜 *PANTUN NUSANTARA:*\n\n%s", pantunList[idx])
}

var brainTeasers = []struct {
	Question string
	Answer   string
}{
	{Question: "Apa yang selalu datang tapi tidak pernah tiba?", Answer: "Besok (hari esok)"},
	{Question: "Memiliki banyak gigi tapi tidak bisa menggigit, benda apakah itu?", Answer: "Sisir rambut"},
	{Question: "Benda apa yang jika diisi semakin ringan?", Answer: "Balon udara"},
	{Question: "Semakin dipotong semakin panjang, apakah itu?", Answer: "Celana panjang yang dipotong bagian bawahnya (atau tali)"},
	{Question: "Benda apa yang selalu berjalan tapi tidak punya kaki?", Answer: "Jam dinding / Jam tangan"},
}

func FetchRandomBrainTeaser() string {
	idx := rand.Intn(len(brainTeasers))
	bt := brainTeasers[idx]
	return fmt.Sprintf("🧠 *KUIS ASAH OTAK:*\n\n*%s*\n\n_Kunci Jawaban: %s_", bt.Question, bt.Answer)
}

var zodiacData = map[string]string{
	"aries":       "♈ *ARIES (21 Mar - 19 Apr)*\n• Karir: Energi kepemimpinanmu terpancar, ambil inisiatif baru.\n• Asmara: Beri kejutan kecil untuk orang tersayang.\n• Angka Hoki: 9, 18, 27",
	"taurus":      "♉ *TAURUS (20 Apr - 20 Mei)*\n• Karir: Stabilitas finansialmu membaik, kelola aset dengan bijak.\n• Asmara: Kesabaranmu menjadi perekat hubungan.\n• Angka Hoki: 6, 15, 24",
	"gemini":      "♊ *GEMINI (21 Mei - 20 Jun)*\n• Karir: Komunikasi lancar membuka peluang kolaborasi emas.\n• Asmara: Obrolan santai membawa chemistry baru.\n• Angka Hoki: 5, 14, 23",
	"cancer":      "♋ *CANCER (21 Jun - 22 Jul)*\n• Karir: Percayai intuisi dalam membuat keputusan penting.\n• Asmara: Kehangatan keluarga memberi ketenangan jiwa.\n• Angka Hoki: 2, 7, 16",
	"leo":         "♌ *LEO (23 Jul - 22 Agu)*\n• Karir: Kepercayaan dirimu menginspirasi tim di sekitarmu.\n• Asmara: Sikap tulusmu meluluhkan keraguan pasangan.\n• Angka Hoki: 1, 10, 19",
	"virgo":       "♍ *VIRGO (23 Agu - 22 Sep)*\n• Karir: Ketelitianmu menyelesaikan persoalan rumit secara efisien.\n• Asmara: Hindari overthinking, nikmati momen kebersamaan.\n• Angka Hoki: 3, 12, 21",
	"libra":       "♎ *LIBRA (23 Sep - 22 Okt)*\n• Karir: Keseimbangan kerja dan istirahat menjaga produktivitas.\n• Asmara: Harmoni hubungan semakin erat dan harmonis.\n• Angka Hoki: 4, 13, 22",
	"scorpio":     "♏ *SCORPIO (23 Okt - 21 Nov)*\n• Karir: Fokusmu sedang di puncak, target tertunda akan tuntas.\n• Asmara: Kejujuran adalah kunci hubungan langgeng.\n• Angka Hoki: 8, 17, 26",
	"sagitarius":  "♐ *SAGITARIUS (22 Nov - 21 Des)*\n• Karir: Jiwa petualangmu menemukan ide kreatif bernilai tinggi.\n• Asmara: Spontanitas membakar kembali bara asmara.\n• Angka Hoki: 7, 16, 25",
	"capricorn":   "♑ *CAPRICORN (22 Des - 19 Jan)*\n• Karir: Disiplin dan konsistensimu mulai membuahkan hasil nyata.\n• Asmara: Komitmen saling percaya semakin kuat.\n• Angka Hoki: 4, 8, 13",
	"aquarius":    "♒ *AQUARIUS (20 Jan - 18 Feb)*\n• Karir: Solusi inovatifmu mendapat apresiasi dari kolega.\n• Asmara: Keterbukaan pikiran membuat suasana nyaman.\n• Angka Hoki: 11, 22, 33",
	"pisces":      "♓ *PISCES (19 Feb - 20 Mar)*\n• Karir: Imajinasi dan rasa empatimu melahirkan karya bermakna.\n• Asmara: Perhatian kecilmu begitu berharga bagi pasangan.\n• Angka Hoki: 3, 9, 12",
}

func FetchZodiac(sign string) string {
	s := strings.ToLower(strings.TrimSpace(sign))
	if val, ok := zodiacData[s]; ok {
		return val
	}
	return "⚠️ Nama zodiak tidak dikenali.\nContoh: *!zodiak aries*, *!zodiak scorpio*, *!zodiak leo*"
}

var dailyDuas = map[string]string{
	"makan":        "🤲 *DOA SEBELUM MAKAN*\n\nاَللّٰهُمَّ بَارِكْ لَنَا فِيْمَا رَزَقْتَنَا وَقِنَا عَذَابَ النَّارِ\n_Allāhumma bārik lanā fīmā razaqtanā wa qinā 'ażāban-nār_\n\n\"Ya Allah, berkahilah kami dalam rezeki yang telah Engkau berikan dan peliharalah kami dari siksa api neraka.\"",
	"tidur":        "🤲 *DOA SEBELUM TIDUR*\n\nبِاسْمِكَ اللّٰهُمَّ اَحْيَا وَبِاسْمِكَ اَمُوْتُ\n_Bismika Allāhumma aḥyā wa bismika amūt_\n\n\"Dengan nama-Mu ya Allah aku hidup dan dengan nama-Mu aku mati.\"",
	"keluar rumah": "🤲 *DOA KELUAR RUMAH*\n\nبِسْمِ اللَّهِ تَوَكَّلْتُ عَلَى اللَّهِ لَا حَوْلَ وَلَا قُوَّةَ إِلَّا بِاللَّهِ\n_Bismillāhi tawakkaltu 'alallāh, lā ḥawla wa lā quwwata illā billāh_\n\n\"Dengan nama Allah, aku berserah diri kepada Allah. Tidak ada daya dan kekuatan kecuali dengan pertolongan Allah.\"",
	"orang tua":    "🤲 *DOA UNTUK KEDUA ORANG TUA*\n\nرَبِّ اغْفِرْ لِيْ وَلِوَالِدَيَّ وَارْحَمْهُمَا كَمَا رَبَّيَانِيْ صَغِيْرًا\n_Rabbigfir lī wa liwālidayya warḥamhumā kamā rabbayānī ṣagīrā_\n\n\"Wahai Tuhanku, ampunilah aku dan kedua orang tuaku, dan sayangilah keduanya sebagaimana mereka menyayangiku di waktu kecil.\"",
	"sapujagat":    "🤲 *DOA SAPUJAGAT (KESELAMATAN DUNIA AKHIRAT)*\n\nرَبَّنَا آتِنَا فِي الدُّنْيَا حَسَنَةً وَفِي الآخِرَةِ حَسَنَةً وَقِنَا عَذَابَ النَّارِ\n_Rabbanā ātinā fid-dunyā ḥasanah, wa fil-ākhirati ḥasanah, wa qinā 'ażāban-nār_\n\n\"Ya Tuhan kami, berilah kami kebaikan di dunia dan kebaikan di akhirat, dan lindungilah kami dari siksa api neraka.\"",
}

func FetchDailyDua(query string) string {
	q := strings.ToLower(strings.TrimSpace(query))
	for k, v := range dailyDuas {
		if strings.Contains(q, k) || strings.Contains(k, q) {
			return v
		}
	}
	return "🤲 *KUMPULAN DOA HARIAN*\n\nKetik kata kunci doa yang diinginkan:\n• *!doa makan*\n• *!doa tidur*\n• *!doa keluar rumah*\n• *!doa orang tua*\n• *!doa sapujagat*"
}

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

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
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
