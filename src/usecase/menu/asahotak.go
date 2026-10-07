package menu

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

type BrainTeaserItem struct {
	Category string
	Question string
	Answer   string
	Clue     string
}

var brainTeasersCatalog = []BrainTeaserItem{
	{Category: "logika", Question: "Apa yang selalu datang tapi tidak pernah tiba?", Answer: "Hari esok (besok)", Clue: "Waktu masa depan"},
	{Category: "benda", Question: "Memiliki banyak gigi tapi tidak bisa menggigit, benda apakah itu?", Answer: "Sisir rambut", Clue: "Dipakai setelah mandi"},
	{Category: "logika", Question: "Benda apa yang jika diisi justru semakin ringan?", Answer: "Balon gas / Balon udara", Clue: "Terbang ke atas"},
	{Category: "logika", Question: "Semakin dipotong semakin panjang, apakah itu?", Answer: "Tali atau lubang galian", Clue: "Bisa juga celana yang dipotong bawahnya"},
	{Category: "benda", Question: "Benda apa yang selalu berjalan tapi tidak punya kaki?", Answer: "Jam dinding / Jam tangan", Clue: "Berdetak setiap detik"},
	{Category: "logika", Question: "Punya satu mata tapi tidak bisa melihat, benda apakah itu?", Answer: "Jarum jahit", Clue: "Alat menjahit"},
	{Category: "logika", Question: "Apa yang bisa dipecahkan tanpa pernah disentuh sama sekali?", Answer: "Janji atau rekor", Clue: "Sering diucapkan orang"},
	{Category: "kata", Question: "Huruf apa yang selalu dingin dan menyegarkan?", Answer: "Huruf B (karena ada di tengah-tengah E-S)", Clue: "Pelesetan kata"},
	{Category: "logika", Question: "Bisa basah saat mengeringkan, benda apakah itu?", Answer: "Handuk", Clue: "Perlengkapan mandi"},
	{Category: "kata", Question: "Pintu apa yang didorong oleh 10 orang tetap tidak mau terbuka?", Answer: "Pintu yang tulisannya TARIK", Clue: "Perhatikan petunjuk arah"},
	{Category: "logika", Question: "Makin banyak yang kamu ambil, makin banyak yang kamu tinggalkan di belakang. Apakah itu?", Answer: "Langkah kaki", Clue: "Saat kamu berjalan"},
	{Category: "logika", Question: "Punya leher tapi tidak punya kepala, apakah itu?", Answer: "Botol atau baju", Clue: "Tempat menampung air"},
	{Category: "lucu", Question: "Kucing apa yang paling kuno dan bersejarah?", Answer: "Kucinggalan zaman", Clue: "Pelesetan kata"},
	{Category: "lucu", Question: "Hewan apa yang bersaudara?", Answer: "Katak beradik", Clue: "Amfibi"},
	{Category: "matematika", Question: "Berapa banyak bulan dalam setahun yang memiliki 28 hari?", Answer: "Semua 12 bulan (semua bulan memiliki setidaknya 28 hari)", Clue: "Bukan hanya Februari"},
	{Category: "logika", Question: "Jika kamu sedang lomba lari dan menyalip orang di posisi ke-2, sekarang kamu di posisi ke berapa?", Answer: "Posisi ke-2", Clue: "Bukan posisi ke-1"},
	{Category: "logika", Question: "Apa yang naik tapi tidak pernah turun kembali?", Answer: "Usia / Umur manusia", Clue: "Bertambah setiap tahun"},
	{Category: "benda", Question: "Punya banyak kunci tapi tidak bisa membuka satu pintu pun, apakah itu?", Answer: "Piano atau keyboard komputer", Clue: "Alat musik atau alat ketik"},
	{Category: "kata", Question: "Tikus apa yang cuma punya dua kaki?", Answer: "Mickey Mouse", Clue: "Karakter kartun"},
	{Category: "lucu", Question: "Bebek apa yang jalannya selalu ke kiri?", Answer: "Bebek yang dikunci stang", Clue: "Kendaraan"},
}

func FetchRandomBrainTeaser(query ...string) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	filter := ""
	if len(query) > 0 && strings.TrimSpace(query[0]) != "" {
		filter = strings.ToLower(strings.TrimSpace(query[0]))
	}

	var pool []BrainTeaserItem
	if filter != "" {
		for _, bt := range brainTeasersCatalog {
			if strings.Contains(strings.ToLower(bt.Category), filter) ||
				strings.Contains(strings.ToLower(bt.Question), filter) ||
				strings.Contains(strings.ToLower(bt.Answer), filter) {
				pool = append(pool, bt)
			}
		}
	}

	if len(pool) == 0 {
		pool = brainTeasersCatalog
	}

	idx := r.Intn(len(pool))
	bt := pool[idx]

	cat := strings.ToUpper(bt.Category)
	return fmt.Sprintf("🧠 *KUIS ASAH OTAK & LOGIKA (%s)*\n\n❓ *Pertanyaan:*\n%s\n\n💡 *Petunjuk:* _%s_\n\n||🔒 *Kunci Jawaban:* %s||\n\n_Ketik *!asahotak <kategori>* (logika, lucu, benda, matematika)_", cat, bt.Question, bt.Clue, bt.Answer)
}
