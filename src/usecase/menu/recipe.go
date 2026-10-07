package menu

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

type RecipeItem struct {
	Name        string
	Category    string
	Ingredients []string
	Steps       []string
}

var recipesDatabase = []RecipeItem{
	{
		Name:     "Nasi Goreng Kampung",
		Category: "Nasi",
		Ingredients: []string{
			"2 piring nasi putih dingin (pera)",
			"3 siung bawang merah, iris halus",
			"2 siung bawang putih, cincang",
			"3 buah cabai rawit merah, iris",
			"1 butir telur ayam",
			"1 sdm kecap manis",
			"1/2 sdt terasi bakar",
			"Garam, merica, dan minyak secukupnya",
		},
		Steps: []string{
			"Panaskan sedikit minyak, tumis bawang merah, bawang putih, cabai, dan terasi hingga harum.",
			"Sisihkan bumbu ke tepi wajan, masukkan telur lalu orak-arik hingga matang.",
			"Masukkan nasi putih, aduk rata dengan bumbu di atas api sedang-besar.",
			"Tambahkan kecap manis, garam, dan merica. Aduk cepat hingga beraroma smokey.",
			"Angkat dan sajikan selagi hangat dengan taburan bawang goreng dan kerupuk.",
		},
	},
	{
		Name:     "Rendang Daging Sapi",
		Category: "Daging",
		Ingredients: []string{
			"500 gram daging sapi gandik, potong dadu besar",
			"1 liter santan kental dari 2 butir kelapa",
			"500 ml santan encer",
			"2 batang serai, memarkan",
			"3 lembar daun jeruk & 2 lembar daun kunyit",
			"2 buah asam kandis",
			"Bumbu halus: 8 bawang merah, 4 bawang putih, 10 cabai merah keriting, 2 cm jahe, 2 cm lengkuas, ketumbar, pala, garam",
		},
		Steps: []string{
			"Campurkan santan encer, bumbu halus, serai, daun kunyit, daun jeruk, dan asam kandis ke dalam wajan besar.",
			"Masak dengan api sedang sambil diaduk perlahan hingga santan mengeluarkan minyak.",
			"Masukkan potongan daging sapi, aduk rata dan kecilkan api.",
			"Masak selama 2-3 jam sambil sesekali diaduk agar bagian bawah tidak gosong.",
			"Saat kuah mulai mengering (kalio), masukkan santan kental dan masak terus hingga bumbu berwarna cokelat gelap berminyak.",
		},
	},
	{
		Name:     "Soto Ayam Lamongan",
		Category: "Kuah",
		Ingredients: []string{
			"1/2 ekor ayam kampung",
			"1,5 liter air kaldu",
			"Bumbu halus: 6 bawang merah, 4 bawang putih, 3 butir kemiri sangrai, 2 cm kunyit bakar, 1 cm jahe",
			"Pelengkap: suun, tauge, kol iris, telur rebus, seledri, bubuk koya gurih (kerupuk udang + bawang putih)",
		},
		Steps: []string{
			"Rebus ayam bersama air kaldu, serai, dan daun jeruk hingga empuk. Angkat ayam, suwir dagingnya.",
			"Tumis bumbu halus hingga harum dan matang sempurna, lalu masukkan ke dalam air rebusan kaldu ayam.",
			"Bumbui dengan garam, gula, dan merica. Masak kuah soto hingga mendidih dan bumbu meresap.",
			"Tata suun, kol, tauge, dan suwiran ayam di mangkuk saji.",
			"Siram dengan kuah soto panas dan taburi bubuk koya, seledri, serta perasan jeruk nipis.",
		},
	},
	{
		Name:     "Sate Ayam Madura",
		Category: "Ayam",
		Ingredients: []string{
			"500 gram dada/paha ayam fillet, potong dadu",
			"Tusuk sate secukupnya",
			"Bumbu marinasi: 2 sdm kecap manis, 1 sdm minyak goreng, 1/2 sdt ketumbar bubuk",
			"Bumbu kacang: 150 gram kacang tanah goreng (haluskan), 3 bawang putih, 4 cabai merah, gula merah, kecap manis, air",
		},
		Steps: []string{
			"Tusuk potongan ayam ke tusuk sate (3-4 potong per tusuk). Lumuri dengan bumbu marinasi dan diamkan 15 menit.",
			"Masak bumbu kacang bersama sedikit air, gula merah, dan garam hingga mengental dan keluar minyak.",
			"Celupkan sate ke sedikit bumbu kacang dan kecap sebelum dibakar.",
			"Bakar sate di atas panggangan arang/grill pan hingga matang kecokelatan merata.",
			"Sajikan dengan siraman bumbu kacang gurih, kecap manis, irisan bawang merah, dan cabai rawit.",
		},
	},
	{
		Name:     "Rawon Daging Khas Jawa Timur",
		Category: "Kuah",
		Ingredients: []string{
			"500 gram daging sandung lamur sapi",
			"4 buah kluwek kualitas bagus (rendam air hangat, haluskan isinya)",
			"Bumbu halus: 7 bawang merah, 4 bawang putih, 3 kemiri, 2 cm kunyit, 2 cm jahe, 1 sdt ketumbar",
			"Pelengkap: tauge pendek, sambal terasi, telur asin, kerupuk udang",
		},
		Steps: []string{
			"Rebus daging sapi hingga empuk, lalu potong dadu. Saring kaldunya untuk kuah.",
			"Tumis bumbu halus bersama kluwek, serai, daun jeruk, dan lengkuas hingga matang dan harum pekat.",
			"Masukkan tumisan bumbu ke dalam panci kaldu daging.",
			"Masak dengan api kecil hingga warna kuah hitam pekat dan bumbu meresap sempurna ke daging.",
			"Sajikan selagi panas bersama nasi putih, tauge pendek mentah, telur asin, dan sambal terasi.",
		},
	},
	{
		Name:     "Ayam Geprek Sambal Bawang",
		Category: "Ayam",
		Ingredients: []string{
			"2 potong ayam crispy (tepung krispi)",
			"10 buah cabai rawit merah",
			"2 siung bawang putih",
			"1/4 sdt garam dan kaldu bubuk",
			"2 sdm minyak goreng panas sisa menggoreng",
		},
		Steps: []string{
			"Ulek kasar cabai rawit merah, bawang putih, garam, dan kaldu bubuk di cobek batu.",
			"Siram sambal mentah dengan minyak goreng yang masih mendidih, aduk rata.",
			"Letakkan ayam krispi di atas sambal, lalu geprek atau tekan dengan ulekan hingga remuk.",
			"Aduk rata ayam dengan sambal bawang dan sajikan langsung bersama nasi putih hangat.",
		},
	},
	{
		Name:     "Sayur Asem Segar",
		Category: "Sayur",
		Ingredients: []string{
			"1 ikat kacang panjang, potong-potong",
			"1 buah jagung manis, potong melingkar",
			"50 gram melinjo & daun melinjo",
			"1 buah labu siam, potong dadu",
			"3 buah asam jawa muda & 2 lembar daun salam",
			"Bumbu halus: 5 bawang merah, 2 bawang putih, 3 cabai merah, 2 kemiri, 1 sdt terasi",
		},
		Steps: []string{
			"Rebus air bersama bumbu halus, daun salam, lengkuas, dan asam jawa hingga mendidih dan harum.",
			"Masukkan jagung dan melinjo terlebih dahulu hingga setengah matang.",
			"Masukkan labu siam dan kacang panjang.",
			"Bumbui dengan garam dan gula pasir secukupnya (rasa seimbang antara asam, manis, dan gurih).",
			"Terakhir masukkan daun melinjo, masak sebentar lalu angkat dan sajikan.",
		},
	},
	{
		Name:     "Bakso Sapi Kuah Gurih",
		Category: "Kuah",
		Ingredients: []string{
			"20 butir bakso sapi",
			"1,5 liter air kaldu tulang sapi",
			"4 siung bawang putih goreng, haluskan",
			"2 batang seledri dan daun bawang",
			"Garam, merica bubuk, dan kaldu sapi secukupnya",
			"Pelengkap: mie kuning, bihun, sawi hijau, bawang goreng",
		},
		Steps: []string{
			"Didihkan air kaldu tulang sapi di panci.",
			"Masukkan bawang putih goreng yang sudah dihaluskan, garam, merica, dan kaldu sapi.",
			"Masukkan butiran bakso sapi ke dalam kuah mendidih hingga mengapung matang.",
			"Tata mie, bihun, dan sawi hijau di mangkuk.",
			"Siram dengan kuah kaldu panas dan bakso, taburi seledri dan bawang goreng renyah.",
		},
	},
}

func FetchRecipe(query ...string) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	filter := ""
	if len(query) > 0 && strings.TrimSpace(query[0]) != "" {
		filter = strings.ToLower(strings.TrimSpace(query[0]))
	}

	var pool []RecipeItem
	if filter != "" {
		for _, rec := range recipesDatabase {
			if strings.Contains(strings.ToLower(rec.Name), filter) ||
				strings.Contains(strings.ToLower(rec.Category), filter) {
				pool = append(pool, rec)
			}
		}
	}

	if len(pool) == 0 && filter != "" {
		return fmt.Sprintf("🍳 Resep *%s* belum ditemukan di database koleksi resep nusantara.\n\nContoh resep tersedia:\n• *!resep rendang*\n• *!resep nasi goreng*\n• *!resep soto ayam*\n• *!resep sate*\n• *!resep rawon*\n• *!resep ayam geprek*\n• *!resep sayur asem*\n• *!resep bakso*", strings.Title(filter))
	}

	if len(pool) == 0 {
		pool = recipesDatabase
	}

	idx := r.Intn(len(pool))
	rec := pool[idx]

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🍳 *RESEP MASAKAN: %s*\n", strings.ToUpper(rec.Name)))
	sb.WriteString(fmt.Sprintf("🏷️ Kategori: %s\n\n", rec.Category))

	sb.WriteString("*Bahan-Bahan:*\n")
	for _, ing := range rec.Ingredients {
		sb.WriteString(fmt.Sprintf("• %s\n", ing))
	}

	sb.WriteString("\n*Langkah Memasak:*\n")
	for i, step := range rec.Steps {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
	}

	sb.WriteString("\n_Ketik *!resep <nama_makanan>* untuk mencari menu hidangan lainnya_")
	return sb.String()
}
