package menu

import (
	"strings"
)

var dailyDuas = map[string]string{
	"makan":        "🤲 *DOA SEBELUM MAKAN*\n\nاَللّٰهُمَّ بَارِكْ لَنَا فِيْمَا رَزَقْتَنَا وَقِنَا عَذَابَ النَّارِ\n_Allāhumma bārik lanā fīmā razaqtanā wa qinā 'ażāban-nār_\n\n\"Ya Allah, berkahilah kami dalam rezeki yang telah Engkau berikan dan peliharalah kami dari siksa api neraka.\"",
	"tidur":        "🤲 *DOA SEBELUM TIDUR*\n\nبِاسْمِكَ اللّٰهُمَّ اَحْيَا وَبِاسْمِكَ اَمُوْتُ\n_Bismika Allāhumma aḥyā wa bismika amūt_\n\n\"Dengan nama-Mu ya Allah aku hidup dan dengan nama-Mu aku mati.\"",
	"keluar rumah": "🤲 *DOA KELUAR RUMAH*\n\nبِسْمِ اللَّهِ تَوَكَّلْتُ عَلَى اللَّهِ لَا حَوْلَ وَلَا قُوَّةَ إِلَّا بِاللَّهِ\n_Bismillāhi tawakkaltu 'alallāh, lā ḥawla wa lā quwwata illā billāh_\n\n\"Dengan nama Allah, aku berserah diri kepada Allah. Tidak ada daya dan kekuatan kecuali dengan pertolongan Allah.\"",
	"orang tua":    "🤲 *DOA UNTUK KEDUA ORANG TUA*\n\nرَبِّ اغْفِرْ لِيْ وَلِوَالِدَيَّ وَارْحَمْهُمَا كَمَا رَبَّيَانِيْ صَغِيْرًا\n_Rabbigfir lī wa liwālidayya warḥamhumā kamā rabbayānī ṣagīrā_\n\n\"Wahai Tuhanku, ampunilah aku dan kedua orang tuaku, dan sayangilah keduanya sebagaimana mereka menyayangiku di waktu kecil.\"",
	"sapujagat":    "🤲 *DOA SAPUJAGAT (KESELAMATAN DUNIA AKHIRAT)*\n\nرَبَّنَا آتِنَا فِي الدُّنْيَا حَسَنَةً وَفِي الآخِرَةِ حَسَنَةً وَقِنَا عَذَابَ النَّارِ\n_Rabbanā ātinā fid-dunyā ḥasanah, wa fil-ākhirati ḥasanah, wa qinā 'ażāban-nār_\n\n\"Ya Tuhan kami, berilah kami kebaikan di dunia dan kebaikan di akhirat, dan lindungilah kami dari siksa api neraka.\"",
}

func FetchDailyDua(query string) string {
	q := strings.ToLower(strings.TrimSpace(query))
	if q != "" {
		for k, v := range dailyDuas {
			if strings.Contains(q, k) || strings.Contains(k, q) {
				return v
			}
		}
	}
	return "🤲 *KUMPULAN DOA HARIAN*\n\nKetik kata kunci doa yang diinginkan:\n• *!doa makan*\n• *!doa tidur*\n• *!doa keluar rumah*\n• *!doa orang tua*\n• *!doa sapujagat*"
}
