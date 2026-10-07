package menu

import (
	"fmt"
	"strings"
)

type DuaItem struct {
	Title      string
	Arabic     string
	Latin      string
	Indonesian string
	Keywords   []string
}

var dailyDuasCollection = []DuaItem{
	{
		Title:      "DOA SEBELUM MAKAN",
		Arabic:     "اَللّٰهُمَّ بَارِكْ لَنَا فِيْمَا رَزَقْتَنَا وَقِنَا عَذَابَ النَّارِ",
		Latin:      "Allāhumma bārik lanā fīmā razaqtanā wa qinā 'ażāban-nār",
		Indonesian: "Ya Allah, berkahilah kami dalam rezeki yang telah Engkau berikan kepada kami dan peliharalah kami dari siksa api neraka.",
		Keywords:   []string{"makan", "sebelum makan", "rezeki"},
	},
	{
		Title:      "DOA SESUDAH MAKAN",
		Arabic:     "اَلْحَمْدُ لِلّٰهِ الَّذِيْ أَطْعَمَنَا وَسَقَانَا وَجَعَلَنَا مُسْلِمِيْنَ",
		Latin:      "Al-ḥamdu lillāhillażī aṭ'amanā wa saqānā wa ja'alanā muslimīn",
		Indonesian: "Segala puji bagi Allah yang telah memberi makan dan minum kepada kami serta menjadikan kami termasuk orang-orang muslim.",
		Keywords:   []string{"sesudah makan", "selesai makan", "setelah makan"},
	},
	{
		Title:      "DOA SEBELUM TIDUR",
		Arabic:     "بِاسْمِكَ اللّٰهُمَّ اَحْيَا وَبِاسْمِكَ اَمُوْتُ",
		Latin:      "Bismika Allāhumma aḥyā wa bismika amūt",
		Indonesian: "Dengan nama-Mu ya Allah aku hidup dan dengan nama-Mu aku mati.",
		Keywords:   []string{"tidur", "sebelum tidur"},
	},
	{
		Title:      "DOA BANGUN TIDUR",
		Arabic:     "اَلْحَمْدُ لِلّٰهِ الَّذِيْ أَحْيَانَا بَعْدَ مَا أَمَاتَنَا وَإِلَيْهِ النُّشُوْرُ",
		Latin:      "Al-ḥamdu lillāhillażī aḥyānā ba'da mā amātanā wa ilaihin-nusyūr",
		Indonesian: "Segala puji bagi Allah yang telah menghidupkan kami setelah mematikan kami dan kepada-Nya lah kami dibangkitkan.",
		Keywords:   []string{"bangun tidur", "pagi", "bangun"},
	},
	{
		Title:      "DOA KELUAR RUMAH",
		Arabic:     "بِسْمِ اللَّهِ تَوَكَّلْتُ عَلَى اللَّهِ لَا حَوْلَ وَلَا قُوَّةَ إِلَّا بِاللَّهِ",
		Latin:      "Bismillāhi tawakkaltu 'alallāh, lā ḥawla wa lā quwwata illā billāh",
		Indonesian: "Dengan nama Allah, aku berserah diri kepada Allah. Tidak ada daya dan kekuatan kecuali dengan pertolongan Allah.",
		Keywords:   []string{"keluar rumah", "pergi", "keluar"},
	},
	{
		Title:      "DOA MASUK RUMAH",
		Arabic:     "بِسْمِ اللّٰهِ وَلَجْنَا، وَبِسْمِ اللّٰهِ خَرَجْنَا، وَعَلَى رَبِّنَا تَوَكَّلْنَا",
		Latin:      "Bismillāhi walajnā, wa bismillāhi kharajnā, wa 'alā rabbinā tawakkalnā",
		Indonesian: "Dengan nama Allah kami masuk, dan dengan nama Allah kami keluar, dan kepada Tuhan kami, kami bertawakal.",
		Keywords:   []string{"masuk rumah", "pulang"},
	},
	{
		Title:      "DOA UNTUK KEDUA ORANG TUA",
		Arabic:     "رَبِّ اغْفِرْ لِيْ وَلِوَالِدَيَّ وَارْحَمْهُمَا كَمَا رَبَّيَانِيْ صَغِيْرًا",
		Latin:      "Rabbigfir lī wa liwālidayya warḥamhumā kamā rabbayānī ṣagīrā",
		Indonesian: "Wahai Tuhanku, ampunilah aku dan kedua orang tuaku, dan sayangilah keduanya sebagaimana mereka menyayangiku di waktu kecil.",
		Keywords:   []string{"orang tua", "ibu", "ayah", "bapak"},
	},
	{
		Title:      "DOA SAPUJAGAT (KESELAMATAN DUNIA AKHIRAT)",
		Arabic:     "رَبَّنَا آتِنَا فِي الدُّنْيَا حَسَنَةً وَفِي الآخِرَةِ حَسَنَةً وَقِنَا عَذَابَ النَّارِ",
		Latin:      "Rabbanā ātinā fid-dunyā ḥasanah, wa fil-ākhirati ḥasanah, wa qinā 'ażāban-nār",
		Indonesian: "Ya Tuhan kami, berilah kami kebaikan di dunia dan kebaikan di akhirat, dan lindungilah kami dari siksa api neraka.",
		Keywords:   []string{"sapujagat", "selamat", "keselamatan", "dunia akhirat"},
	},
	{
		Title:      "DOA KETIKA TURUN HUJAN",
		Arabic:     "اَللّٰهُمَّ صَيِّبًا نَافِعًا",
		Latin:      "Allāhumma ṣayyiban nāfi'ā",
		Indonesian: "Ya Allah, turunkanlah hujan yang membawa manfaat.",
		Keywords:   []string{"hujan", "gerimis"},
	},
	{
		Title:      "DOA MEMOHON KELAPANGAN DADA & KEMUDAHAN URUSAN",
		Arabic:     "رَبِّ اشْرَحْ لِي صَدْرِي وَيَسِّرْ لِي أَمْرِي وَاحْلُلْ عُقْدَةً مِّن لِّسَانِي يَفْقَهُوا قَوْلِي",
		Latin:      "Rabbisyraḥ lī ṣadrī wa yassir lī amrī waḥlul 'uqdatam mil-lisānī yafqahū qaulī",
		Indonesian: "Ya Tuhanku, lapangkanlah dadaku, dan mudahkanlah urusanku, dan lepaskanlah kekakuan dari lidahku, agar mereka mengerti perkataanku.",
		Keywords:   []string{"lapang dada", "kemudahan", "ujian", "bicara", "presentasi"},
	},
	{
		Title:      "DOA MASUK KAMAR MANDI / WC",
		Arabic:     "اَللّٰهُمَّ إِنِّي أَعُوْذُ بِكَ مِنَ الْخُبُثِ وَالْخَبَائِثِ",
		Latin:      "Allāhumma innī a'ūżu bika minal-khubuṡi wal-khabā'iṡ",
		Indonesian: "Ya Allah, sesungguhnya aku berlindung kepada-Mu dari godaan setan laki-laki dan setan perempuan.",
		Keywords:   []string{"kamar mandi", "wc", "toilet"},
	},
	{
		Title:      "DOA KELUAR KAMAR MANDI / WC",
		Arabic:     "غُفْرَانَكَ، اَلْحَمْدُ لِلّٰهِ الَّذِيْ أَذْهَبَ عَنِّي الْأَذَى وَعَافَانِيْ",
		Latin:      "Gufrānaka, al-ḥamdu lillāhillażī ażhaba 'annil-ażā wa 'āfānī",
		Indonesian: "Aku memohon ampunan-Mu. Segala puji bagi Allah yang telah menghilangkan penyakit dari tubuhku dan memberikanku kesehatan.",
		Keywords:   []string{"keluar wc", "keluar toilet", "keluar kamar mandi"},
	},
	{
		Title:      "DOA BERCERMIN",
		Arabic:     "اَللّٰهُمَّ كَمَا حَسَّنْتَ خَلْقِيْ فَحَسِّنْ خُلُقِيْ",
		Latin:      "Allāhumma kamā ḥassanta khalqī fa ḥassin khuluqī",
		Indonesian: "Ya Allah, sebagaimana Engkau telah membaguskan rupa fisikku, maka baguskanlah pula akhlak budi pekertiku.",
		Keywords:   []string{"cermin", "bercermin", "kaca"},
	},
	{
		Title:      "DOA NAIK KENDARAAN / BEPERGIAN",
		Arabic:     "سُبْحَانَ الَّذِي سَخَّرَ لَنَا هَذَا وَمَا كُنَّا لَهُ مُقْرِنِينَ وَإِنَّا إِلَى رَبِّنَا لَمُنْقَلِبُونَ",
		Latin:      "Subḥānallażī sakhkhara lanā hāżā wa mā kunnā lahū muqrinīn, wa innā ilā rabbinā lamunqalibūn",
		Indonesian: "Maha Suci Allah yang telah menundukkan semua ini bagi kami padahal sebelumnya kami tidak mampu menguasainya, dan sesungguhnya kami akan kembali kepada Tuhan kami.",
		Keywords:   []string{"kendaraan", "perjalanan", "bepergian", "safar", "mobil", "motor"},
	},
	{
		Title:      "DOA KELANCARAN REZEKI",
		Arabic:     "اَللّٰهُمَّ اكْفِنِيْ بِحَلَالِكَ عَنْ حَرَامِكَ وَأَغْنِنِيْ بِفَضْلِكَ عَمَّنْ سِوَاكَ",
		Latin:      "Allāhummakfinī biḥalālika 'an ḥarāmika wa agninī bifaḍlika 'amman siwāka",
		Indonesian: "Ya Allah, cukupkanlah aku dengan rezeki-Mu yang halal hingga aku terhindar dari yang haram, dan perkaya lah aku dengan karunia-Mu hingga aku tidak bergantung kepada selain-Mu.",
		Keywords:   []string{"rezeki", "harta", "usaha", "bisnis", "utang"},
	},
}

func FetchDailyDua(query string) string {
	q := strings.ToLower(strings.TrimSpace(query))
	if q != "" {
		for _, item := range dailyDuasCollection {
			if strings.Contains(strings.ToLower(item.Title), q) {
				return formatDuaOutput(item)
			}
			for _, kw := range item.Keywords {
				if strings.Contains(q, kw) || strings.Contains(kw, q) {
					return formatDuaOutput(item)
				}
			}
		}
	}

	return "🤲 *KUMPULAN DOA HARIAN ISLAM*\n\nKetik kata kunci doa yang dicari:\n• *!doa makan* / *!doa sesudah makan*\n• *!doa tidur* / *!doa bangun tidur*\n• *!doa keluar rumah* / *!doa masuk rumah*\n• *!doa orang tua* (ayah & ibu)\n• *!doa sapujagat* (dunia akhirat)\n• *!doa rezeki* / *!doa kemudahan*\n• *!doa perjalanan* (naik kendaraan)\n• *!doa hujan* / *!doa cermin* / *!doa toilet*"
}

func formatDuaOutput(d DuaItem) string {
	return fmt.Sprintf("🤲 *%s*\n\n%s\n\n_%s_\n\n\"%s\"", d.Title, d.Arabic, d.Latin, d.Indonesian)
}
