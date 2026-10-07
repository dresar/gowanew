package menu

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

type PantunItem struct {
	Category string
	Text     string
}

var pantunCatalog = []PantunItem{
	{Category: "nasehat", Text: "Beli pulsa di toko Pak Rahmat,\nPulang ke rumah disambut senyuman.\nAwali hari dengan semangat,\nSemoga sukses dalam genggaman."},
	{Category: "nasehat", Text: "Jalan-jalan ke Kota Blitar,\nJangan lupa membeli sukun.\nJika kamu ingin pintar,\nBelajarlah dengan tekun."},
	{Category: "nasehat", Text: "Pohon beringin rindang daunnya,\nTempat berteduh sang gembala.\nBekerjalah dengan sepenuh jiwa,\nRezeki halal membawa pahala."},
	{Category: "nasehat", Text: "Pergi ke pasar membeli nangka,\nNangka dibelah manis rasanya.\nTetaplah ramah kepada sesama,\nHidup bahagia damai selamanya."},
	{Category: "nasehat", Text: "Burung dara terbang melayang,\nHinggap sebentar di dahan cemara.\nKepada sahabat selalu sayang,\nHati tenang tiada duka."},
	{Category: "nasehat", Text: "Kayu jati dibuat papan,\nDibawa pedagang ke Surabaya.\nJanganlah malas di masa depan,\nAgar hidupmu penuh mulia."},
	{Category: "nasehat", Text: "Anak rusa di tepi kali,\nMinum air segar rasanya.\nJaga lisan jangan mencaci,\nAgar selamat di mana saja."},
	{Category: "nasehat", Text: "Ikan gabus berenang tenang,\nDi bawah daun teratai mekar.\nOrang sabar hatinya lapang,\nMenghadapi cobaan pantang gusar."},

	{Category: "jenaka", Text: "Pohon kelapa tumbuh sebatang,\nTertiup angin condong ke rawa.\nKucing tetangga bergaya garang,\nLihat cicak langsung tertawa."},
	{Category: "jenaka", Text: "Makan bakso ditambah cuka,\nKeringat mengalir sampai ke dahi.\nNiat hati ingin bergaya,\nCelana robek tak disadari."},
	{Category: "jenaka", Text: "Buah mangga buah pepaya,\nDibawa kancil ke tengah hutan.\nKukira dia sudah kaya,\nRupanya utang di mana-mana."},
	{Category: "jenaka", Text: "Naik sepeda rodanya lepas,\nJatuh tersungkur di pohon randu.\nMuka glowing dompet mengempas,\nAkhir bulan makan terigu."},
	{Category: "jenaka", Text: "Pergi ke kali mencuci wajan,\nAirnya keruh kena jelaga.\nMimpi indah jadi jutawan,\nPas bangun masih di kasur tua."},
	{Category: "jenaka", Text: "Beli onde di pasar baru,\nPulang ke rumah jalan meliuk.\nNiat diet mulai hari Rabu,\nLihat martabak langsung diringkuk."},

	{Category: "cinta", Text: "Bunga mawar harum baunya,\nDipetik gadis di pagi hari.\nBukan paras yang memesona,\nKetulusan hatimu yang kucari."},
	{Category: "cinta", Text: "Bintang malam bersinar terang,\nMenemani bulan di cakrawala.\nWalau jarak membentang panjang,\nRasa setia tetap terjaga."},
	{Category: "cinta", Text: "Sungai jernih airnya mengalir,\nMenuju muara di tepi lautan.\nSejak pertama senyummu hadir,\nHatiku terpikat sepanjang zaman."},
	{Category: "cinta", Text: "Hujan gerimis di kota dingin,\nSecangkir teh manis pengusir duka.\nBukan harta yang kuinginkan,\nHanya dirimu pendamping setia."},
	{Category: "cinta", Text: "Mekar melati di taman bunga,\nDisiram embun di fajar pagi.\nBila nanti kita bersama,\nBahagia kan abadi selamanya."},

	{Category: "teka-teki", Text: "Ada sisir tidak berambut,\nAda mulut tidak berkata.\nBenda apakah yang lembut,\nBila ditiup melayang ke angkasa?"},
	{Category: "teka-teki", Text: "Tinggi ramping tiada bertangan,\nBila dibakar badannya cair.\nTerang benderang menerangi ruangan,\nBenda apakah yang terakhir?"},
	{Category: "teka-teki", Text: "Makan rumput berkaki empat,\nAda janggut tiada kumis.\nBila dagingnya dibuat gulai lezat,\nSiapakah hewan yang sering meringis?"},

	{Category: "agama", Text: "Menjelang maghrib burung kembali,\nMenuju sarang di atas bukit.\nJangan lupa sholat lima waktu sehari,\nAgar terhindar dari siksa yang sulit."},
	{Category: "agama", Text: "Indah nian bunga kamboja,\nTumbuh mekar di tanah basah.\nBerbuat baik janganlah riya,\nAgar pahalamu tiada musnah."},
	{Category: "agama", Text: "Pergi mengaji membawa kitab,\nKitab suci pedoman insan.\nJadikan Al-Qur'an sebagai adab,\nSelamat hidup dunia dan akhir zaman."},
}

func FetchRandomPantun(query ...string) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	filter := ""
	if len(query) > 0 && strings.TrimSpace(query[0]) != "" {
		filter = strings.ToLower(strings.TrimSpace(query[0]))
	}

	var pool []PantunItem
	if filter != "" {
		for _, p := range pantunCatalog {
			if strings.Contains(strings.ToLower(p.Category), filter) || strings.Contains(strings.ToLower(p.Text), filter) {
				pool = append(pool, p)
			}
		}
	}

	if len(pool) == 0 {
		pool = pantunCatalog
	}

	idx := r.Intn(len(pool))
	item := pool[idx]

	cat := strings.ToUpper(item.Category)
	return fmt.Sprintf("📜 *PANTUN NUSANTARA (%s):*\n\n%s\n\n_Ketik *!pantun <kategori>* untuk mencari jenis pantun (jenaka, nasehat, cinta, agama, teka-teki)_", cat, item.Text)
}
