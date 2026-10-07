package menu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type FactItem struct {
	Text     string
	Category string
	Source   string
}

var factsDatabase = []FactItem{
	{Category: "sains", Text: "DNA manusia jika dibentangkan dari satu sel tubuh bisa mencapai panjang sekitar 2 meter, dan jika seluruh sel digabungkan bisa mencapai jarak bolak-balik Bumi ke Matahari ratusan kali.", Source: "Genetika"},
	{Category: "sains", Text: "Air panas dapat membeku lebih cepat daripada air dingin dalam kondisi tertentu, sebuah fenomena fisika yang dikenal sebagai Mpemba Effect.", Source: "Termodinamika"},
	{Category: "sains", Text: "Sebagian besar atom adalah ruang kosong. Jika seluruh ruang kosong di dalam atom tubuh manusia dihilangkan, seluruh populasi manusia bisa muat ke dalam sebutir gula batu.", Source: "Fisika Kuantum"},
	{Category: "sains", Text: "Petir memiliki suhu sekitar 30.000 derajat Celsius, yaitu sekitar 5 kali lebih panas daripada permukaan Matahari.", Source: "Meteorologi"},
	{Category: "sains", Text: "Cahaya dari Matahari membutuhkan waktu sekitar 8 menit dan 20 detik untuk mencapai permukaan Bumi.", Source: "Astrofisika"},
	{Category: "sains", Text: "Intan dan grafit pensil terbuat dari elemen kimia yang sama persis, yaitu karbon murni, namun memiliki struktur kristal yang berbeda.", Source: "Kimia Material"},
	{Category: "sains", Text: "Satu sendok teh materi bintang neutron memiliki berat sekitar 6 miliar ton jika ditimbang di Bumi.", Source: "Astrofisika"},
	{Category: "sains", Text: "Kaca sebenarnya bukan zat padat sempurna atau cair biasa, melainkan zat padat amorf yang molekulnya tidak tersusun secara teratur.", Source: "Fisika Material"},

	{Category: "hewan", Text: "Gurita memiliki tiga buah jantung, sembilan otak (satu otak utama dan delapan ganglion di setiap tentakel), serta darah berwarna biru karena mengandung hemosianin.", Source: "Biologi Laut"},
	{Category: "hewan", Text: "Kucing rumahan memiliki kesamaan DNA sebesar 95,6% dengan harimau liar Sumatera.", Source: "Zoologi"},
	{Category: "hewan", Text: "Jantung paus biru memiliki berat sekitar 180 kilogram dan ukurannya sebanding dengan sebuah mobil kecil.", Source: "Biologi Laut"},
	{Category: "hewan", Text: "Burung kolibri adalah satu-satunya burung di dunia yang mampu terbang mundur dan melayang di udara dengan stabil.", Source: "Ornitologi"},
	{Category: "hewan", Text: "Kuda laut jantan adalah yang mengandung dan melahirkan anak, bukan sang betina.", Source: "Biologi Laut"},
	{Category: "hewan", Text: "Lumba-lumba tidur dengan hanya menonaktifkan separuh belahan otaknya dan membiarkan satu matanya tetap terbuka untuk mewaspadai predator.", Source: "Mamalogi"},
	{Category: "hewan", Text: "Kecoa dapat bertahan hidup selama beberapa minggu tanpa kepala sebelum akhirnya mati karena kehausan, bukan karena kehilangan kepala.", Source: "Entomologi"},
	{Category: "hewan", Text: "Sidik jari koala sangat mirip dengan sidik jari manusia, hingga terkadang sulit dibedakan oleh ahli forensik di tempat kejadian perkara.", Source: "Forensik"},
	{Category: "hewan", Text: "Flamingo sebenarnya lahir dengan bulu berwarna abu-abu. Warna merah muda mereka berasal dari pigmen karotenoid pada makanan udang dan alga yang mereka konsumsi.", Source: "Zoologi"},
	{Category: "hewan", Text: "Semut tidak memiliki paru-paru. Mereka bernapas melalui lubang-lubang kecil di seluruh tubuhnya yang disebut spirakel.", Source: "Entomologi"},

	{Category: "antariksa", Text: "Satu hari di planet Venus lebih lama daripada satu tahun di planet tersebut. Venus membutuhkan 243 hari Bumi untuk berotasi, tetapi hanya 225 hari Bumi untuk mengitari Matahari.", Source: "Astronomi"},
	{Category: "antariksa", Text: "Di planet Saturnus dan Jupiter, tekanan atmosfer yang begitu dahsyat diperkirakan bisa membuat hujan berlian.", Source: "Astronomi"},
	{Category: "antariksa", Text: "Matahari menyumbang sekitar 99,86% dari total massa seluruh tata surya kita.", Source: "Astrofisika"},
	{Category: "antariksa", Text: "Bulan perlahan-lahan menjauhi Bumi sekitar 3,8 sentimeter setiap tahunnya karena gaya pasang surut gravitasi.", Source: "Astronomi"},
	{Category: "antariksa", Text: "Jika dua logam sejenis bersentuhan di ruang hampa udara antariksa tanpa lapisan oksidasi, mereka akan menyatu secara permanen melalui proses cold welding.", Source: "Teknik Antariksa"},
	{Category: "antariksa", Text: "Ruang angkasa sepenuhnya hening tanpa suara karena gelombang suara membutuhkan medium udara atau materi untuk merambat.", Source: "Akustik"},
	{Category: "antariksa", Text: "Gunung Olympus Mons di planet Mars adalah gunung berapi tertinggi di tata surya, tingginya hampir 3 kali lipat dari Gunung Everest.", Source: "Astronomi"},

	{Category: "tubuh", Text: "Otak manusia menghasilkan daya listrik sekitar 12 hingga 25 watt saat terjaga, cukup untuk menyalakan sebuah lampu LED kecil.", Source: "Neurosains"},
	{Category: "tubuh", Text: "Panjang seluruh pembuluh darah di dalam tubuh orang dewasa jika disambungkan bisa mencapai sekitar 100.000 kilometer, cukup untuk mengelilingi Bumi 2,5 kali.", Source: "Anatomi"},
	{Category: "tubuh", Text: "Lambung manusia menghasilkan lapisan lendir baru setiap beberapa hari agar asam lambung berkonsentrasi tinggi tidak mencerna lambung itu sendiri.", Source: "Fisiologi"},
	{Category: "tubuh", Text: "Kornea mata adalah satu-satunya bagian tubuh manusia yang tidak memiliki suplai pembuluh darah, dan mendapatkan oksigen langsung dari udara bebas.", Source: "Oftalmologi"},
	{Category: "tubuh", Text: "Tulang paha manusia (femur) lebih kuat daripada beton padat dan mampu menopang beban hingga berkali-kali lipat berat tubuh.", Source: "Ortopedi"},
	{Category: "tubuh", Text: "Manusia kehilangan sekitar 30.000 hingga 40.000 sel kulit mati setiap menitnya, yang berarti hampir 4 kilogram debu sel kulit per tahun.", Source: "Dermatologi"},

	{Category: "sejarah", Text: "Perang tersingkat dalam sejarah dunia terjadi antara Britania Raya dan Kesultanan Zanzibar pada 27 Agustus 1896, yang berlangsung hanya selama 38 hingga 45 menit.", Source: "Sejarah Dunia"},
	{Category: "sejarah", Text: "Piramida Agung Giza di Mesir sudah berusia lebih dari 2.500 tahun ketika peradaban Kekaisaran Romawi Kuno didirikan.", Source: "Arkeologi"},
	{Category: "sejarah", Text: "Universitas tertua di dunia yang masih beroperasi hingga kini didirikan oleh seorang wanita muslimah bernama Fatimah al-Fihri pada tahun 859 M di Fes, Maroko (Universitas al-Qarawiyyin).", Source: "Sejarah Pendidikan"},
	{Category: "sejarah", Text: "Korek api kayu modern ditemukan setelah penemuan pemantik api gas (pemantik Dobereiner ditemukan tahun 1823, korek api gesek tahun 1826).", Source: "Sejarah Inovasi"},
	{Category: "sejarah", Text: "Candi Borobudur dibangun pada abad ke-8 dan ke-9 Masehi oleh Dinasti Syailendra, berabad-abad sebelum Katedral Notre-Dame di Paris mulai dibangun.", Source: "Sejarah Nusantara"},

	{Category: "bumi", Text: "Sekitar 71% permukaan Bumi tertutup oleh air, namun lebih dari 80% lautan di dunia masih belum pernah dijelajahi atau dipetakan oleh manusia.", Source: "Oseanografi"},
	{Category: "bumi", Text: "Hutan Hujan Amazon menghasilkan sekitar 20% oksigen daratan dunia dan merupakan rumah bagi 10% dari seluruh spesies flora dan fauna di muka Bumi.", Source: "Ekologi"},
	{Category: "bumi", Text: "Palung Mariana di Samudera Pasifik adalah titik terdalam di muka bumi dengan kedalaman hampir 11.000 meter. Jika Gunung Everest ditaruh di sana, puncaknya masih berada 2 kilometer di bawah air.", Source: "Geologi"},
	{Category: "bumi", Text: "Antartika adalah benua terkering sekaligus terdingin di dunia, dan secara teknis digolongkan sebagai gurun terbesar di muka Bumi.", Source: "Geografi"},
	{Category: "bumi", Text: "Indonesia memiliki lebih dari 17.000 pulau, menjadikannya negara kepulauan terbesar di dunia dan salah satu pemilik garis pantai terpanjang di planet Bumi.", Source: "Geografi Indonesia"},

	{Category: "teknologi", Text: "Komputer pertama yang memandu astronot Apollo 11 mendarat di Bulan pada tahun 1969 hanya memiliki RAM sebesar 4 kilobyte, jauh lebih kecil dari prosesor jam tangan digital masa kini.", Source: "Sejarah Komputer"},
	{Category: "teknologi", Text: "Bug komputer pertama yang tercatat dalam sejarah adalah ngengat sungguhan yang terjebak di dalam relay tabung mesin komputer Harvard Mark II pada tahun 1947.", Source: "Informatika"},
	{Category: "teknologi", Text: "Kabel serat optik bawah laut mengirimkan lebih dari 95% seluruh lalu lintas internet internasional di dunia, bukan satelit luar angkasa.", Source: "Telekomunikasi"},
	{Category: "teknologi", Text: "Bahasa pemrograman Python dinamai dari kelompok komedi asal Inggris 'Monty Python', bukan dari jenis ular piton.", Source: "Ilmu Komputer"},
}

func TranslateToIndonesian(ctx context.Context, text string) (string, error) {
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("empty text")
	}

	endpoint := fmt.Sprintf("https://translate.googleapis.com/translate_a/single?client=gtx&sl=auto&tl=id&dt=t&q=%s", url.QueryEscape(text))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return text, err
	}
	req.Header.Set("User-Agent", "GoWA-Bot/1.0")

	resp, err := GetStandardClient().Do(req)
	if err != nil {
		return text, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return text, fmt.Errorf("translate status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return text, err
	}

	var parsed []interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return text, err
	}

	if len(parsed) > 0 {
		if segments, ok := parsed[0].([]interface{}); ok {
			var sb strings.Builder
			for _, seg := range segments {
				if sArr, ok := seg.([]interface{}); ok && len(sArr) > 0 {
					if translatedText, ok := sArr[0].(string); ok {
						sb.WriteString(translatedText)
					}
				}
			}
			result := sb.String()
			if strings.TrimSpace(result) != "" {
				return result, nil
			}
		}
	}

	return text, nil
}

func FetchOnlineFact(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://uselessfacts.jsph.pl/api/v2/facts/random", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "GoWA-Bot/1.0")

	resp, err := GetStandardClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status: %d", resp.StatusCode)
	}

	var parsed struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", err
	}

	if parsed.Text == "" {
		return "", fmt.Errorf("empty fact")
	}

	translated, _ := TranslateToIndonesian(ctx, parsed.Text)
	return translated, nil
}

func FetchRandomFact(ctx context.Context, query ...string) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	filter := ""
	if len(query) > 0 && strings.TrimSpace(query[0]) != "" {
		filter = strings.ToLower(strings.TrimSpace(query[0]))
	}

	var pool []FactItem

	if filter != "" {
		for _, f := range factsDatabase {
			if strings.Contains(strings.ToLower(f.Category), filter) ||
				strings.Contains(strings.ToLower(f.Text), filter) ||
				strings.Contains(strings.ToLower(f.Source), filter) {
				pool = append(pool, f)
			}
		}
	}

	if len(pool) == 0 && filter != "" {
		onlineFact, err := FetchOnlineFact(ctx)
		if err == nil && onlineFact != "" {
			return fmt.Sprintf("💡 *TAHUKAH KAMU? (FAKTA UNIK)*\n🏷️ *Topik:* %s\n\n\"%s\"\n\n_Sumber: Ensiklopedia Digital (Terjemahan Otomatis)_", strings.Title(filter), onlineFact)
		}
	}

	if len(pool) == 0 {
		pool = factsDatabase
	}

	idx := r.Intn(len(pool))
	item := pool[idx]

	catName := strings.ToUpper(item.Category)
	if catName == "" {
		catName = "UMUM"
	}

	return fmt.Sprintf("💡 *TAHUKAH KAMU? (FAKTA UNIK)*\n🏷️ *Kategori:* %s\n\n\"%s\"\n\n_Sumber: %s_\n_Ketik *!fakta <topik>* untuk mencari topik tertentu (sains, hewan, antariksa, sejarah, tubuh, bumi)_", catName, item.Text, item.Source)
}
