package menu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

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

	resp, err := GetFastClient().Do(req)
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
