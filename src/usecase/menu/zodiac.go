package menu

import (
	"strings"
)

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
