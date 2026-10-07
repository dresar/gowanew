package menu

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

var predefinedQuotes = []struct {
	Text   string
	Author string
	Tag    string
}{
	{Text: "Satu-satunya cara untuk melakukan pekerjaan hebat adalah dengan mencintai apa yang Anda lakukan.", Author: "Steve Jobs", Tag: "kerja"},
	{Text: "Kesuksesan adalah kemampuan untuk beralih dari satu kegagalan ke kegagalan lain tanpa kehilangan antusiasme.", Author: "Winston Churchill", Tag: "sukses"},
	{Text: "Jangan menunggu kesempatan luar biasa. Raih kesempatan biasa dan jadikan itu luar biasa.", Author: "Orison Swett Marden", Tag: "kesempatan"},
	{Text: "Masa depan adalah milik mereka yang percaya pada keindahan impian mereka.", Author: "Eleanor Roosevelt", Tag: "mimpi"},
	{Text: "Lakukan yang terbaik hari ini, agar Anda tidak menyesal esok hari.", Author: "Anonim", Tag: "hidup"},
	{Text: "Bermimpilah setinggi langit. Jika engkau jatuh, engkau akan jatuh di antara bintang-bintang.", Author: "Ir. Soekarno", Tag: "mimpi"},
	{Text: "Kurang cerdas dapat diperbaiki dengan belajar, kurang cakap dapat dihilangkan dengan pengalaman. Namun tidak jujur itu sulit diperbaiki.", Author: "Mohammad Hatta", Tag: "kejujuran"},
	{Text: "Jadilah mata air yang jernih, yang memberikan kehidupan kepada sekitarmu.", Author: "B.J. Habibie", Tag: "hidup"},
	{Text: "Hanya ada dua pilihan: menjadi bermakna atau hilang tanpa jejak.", Author: "Pramoedya Ananta Toer", Tag: "makna"},
	{Text: "Jangan menjelaskan tentang dirimu kepada siapapun, karena yang menyukaimu tidak butuh itu, dan yang membencimu tidak percaya itu.", Author: "Ali bin Abi Thalib", Tag: "kebijaksanaan"},
	{Text: "Ikhlaslah dalam berbuat, karena Allah hanya menerima apa yang murni untuk-Nya.", Author: "Imam Syafi'i", Tag: "ikhlas"},
	{Text: "Kesabaran itu ada dua macam: sabar atas apa yang engkau benci dan sabar atas apa yang engkau sukai.", Author: "Ali bin Abi Thalib", Tag: "sabar"},
	{Text: "Bukan karena suatu hal sulit kita tidak berani, melainkan karena kita tidak berani maka hal itu menjadi sulit.", Author: "Seneca", Tag: "keberanian"},
	{Text: "Kamu memiliki kekuatan atas pikiranmu, bukan kejadian di luar. Sadarilah ini, dan kamu akan menemukan kekuatan.", Author: "Marcus Aurelius", Tag: "kekuatan"},
	{Text: "Orang yang tidak pernah membuat kesalahan tidak pernah mencoba sesuatu yang baru.", Author: "Albert Einstein", Tag: "belajar"},
	{Text: "Perjalanan seribu mil selalu dimulai dengan satu langkah kecil.", Author: "Lao Tzu", Tag: "langkah"},
	{Text: "Pendidikan adalah senjata paling ampuh untuk mengubah dunia.", Author: "Nelson Mandela", Tag: "pendidikan"},
	{Text: "Jangan menunggu waktu yang tepat, mulailah sekarang juga di tempat kamu berada.", Author: "Napoleon Hill", Tag: "aksi"},
	{Text: "Disiplin adalah jembatan penghubung antara impian dan pencapaian nyata.", Author: "Jim Rohn", Tag: "disiplin"},
	{Text: "Hari kemarin adalah kenangan, hari esok adalah harapan, hari ini adalah kenyataan yang harus diperjuangkan.", Author: "Buya Hamka", Tag: "hidup"},
	{Text: "Keberanian bukanlah ketiadaan rasa takut, melainkan kemenangan atas rasa takut itu sendiri.", Author: "Nelson Mandela", Tag: "keberanian"},
	{Text: "Kunci hidup tenang adalah tidak membandingkan prosesmu dengan hasil orang lain.", Author: "Anonim", Tag: "ketenangan"},
	{Text: "Apa yang kamu tanam hari ini dengan kerja keras, akan kamu tuai esok hari dengan rasa syukur.", Author: "Anonim", Tag: "usaha"},
	{Text: "Kebahagiaan terbesar dalam hidup adalah melakukan apa yang orang lain katakan tidak bisa kamu lakukan.", Author: "Walter Bagehot", Tag: "sukses"},
	{Text: "Jangan biarkan suara opini orang lain menenggelamkan suara hatimu sendiri.", Author: "Steve Jobs", Tag: "prinsip"},
	{Text: "Hidup yang tidak dipertaruhkan tidak akan pernah dimenangkan.", Author: "Sutan Sjahrir", Tag: "perjuangan"},
	{Text: "Ilmu tanpa amal bagaikan pohon tanpa buah.", Author: "Peribahasa", Tag: "ilmu"},
	{Text: "Jika kamu ingin berjalan cepat, berjalanlah sendirian. Jika kamu ingin berjalan jauh, berjalanlah bersama-sama.", Author: "Pepatah Kuno", Tag: "kolaborasi"},
	{Text: "Bekerjalah seolah-olah kamu hidup selamanya, dan beribadahlah seolah-olah kamu mati esok hari.", Author: "Ali bin Abi Thalib", Tag: "keseimbangan"},
	{Text: "Waktu yang kamu nikmati untuk dihabiskan bukanlah waktu yang terbuang sia-sia.", Author: "John Lennon", Tag: "waktu"},
	{Text: "Fokus pada apa yang bisa kamu kendalikan, lepaskan apa yang di luar kendalimu.", Author: "Epictetus", Tag: "fokus"},
	{Text: "Kemauan untuk menang adalah penting, tetapi kemauan untuk mempersiapkan diri jauh lebih penting.", Author: "Joe Paterno", Tag: "persiapan"},
	{Text: "Jangan takut berjalan lambat, takutlah jika kamu hanya berdiri diam.", Author: "Pepatah", Tag: "progres"},
	{Text: "Kualitas hidupmu ditentukan oleh kualitas pertanyaan yang kamu ajukan pada dirimu sendiri.", Author: "Tony Robbins", Tag: "refleksi"},
	{Text: "Bersyukurlah atas hal-hal kecil, karena suatu hari nanti kamu akan menyadari bahwa itu adalah hal-hal besar.", Author: "Robert Brault", Tag: "syukur"},
	{Text: "Rasa sakit karena disiplin jauh lebih ringan daripada rasa sakit karena penyesalan.", Author: "Jim Rohn", Tag: "disiplin"},
	{Text: "Setiap kesulitan selalu membawa benih kemudahan yang setara atau bahkan lebih besar.", Author: "Napoleon Hill", Tag: "optimisme"},
	{Text: "Jadilah pribadi yang bernilai, bukan sekadar pribadi yang sukses.", Author: "Albert Einstein", Tag: "nilai"},
	{Text: "Ketenangan hati lahir saat kamu melepaskan keinginan untuk mengontrol semua hal.", Author: "Anonim", Tag: "damai"},
	{Text: "Kesempatan seringkali datang menyamar sebagai kerja keras yang dihindari banyak orang.", Author: "Thomas Edison", Tag: "kerja"},
	{Text: "Tidak ada lift menuju sukses, kamu harus menaiki tangga satu per satu.", Author: "Zig Ziglar", Tag: "proses"},
	{Text: "Memaafkan bukan berarti mengubah masa lalu, melainkan memperluas ruang untuk masa depan yang damai.", Author: "Paul Boese", Tag: "maaf"},
	{Text: "Karakter sejati seseorang terlihat saat ia menghadapi ujian dan kesulitan, bukan saat ia berada di puncak kenyamanan.", Author: "Martin Luther King Jr.", Tag: "karakter"},
	{Text: "Apa yang kamu cari sebenarnya sedang mencari kamu juga.", Author: "Jalaluddin Rumi", Tag: "takdir"},
	{Text: "Ketika kamu bersyukur, rasa takut akan hilang dan kelimpahan akan hadir.", Author: "Tony Robbins", Tag: "syukur"},
	{Text: "Satu ons aksi nyata bernilai lebih dari satu ton teori belaka.", Author: "Friedrich Engels", Tag: "aksi"},
	{Text: "Jangan biarkan kegagalan kemarin merampok keindahan hari ini.", Author: "Anonim", Tag: "bangkit"},
	{Text: "Kebaikan kecil yang kamu sebarkan hari ini bisa menjadi keajaiban besar bagi orang lain esok hari.", Author: "Anonim", Tag: "kebaikan"},
	{Text: "Kecerdasan tanpa ketulusan hanya akan melahirkan kepalsuan.", Author: "Buya Hamka", Tag: "tulus"},
	{Text: "Bukan banyaknya ilmu yang membuat seseorang mulia, melainkan manfaat dari ilmu tersebut bagi sesama.", Author: "Hasan Al-Bashri", Tag: "ilmu"},
}

func FetchRandomQuote(query ...string) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	var pool []struct {
		Text   string
		Author string
		Tag    string
	}

	filter := ""
	if len(query) > 0 && strings.TrimSpace(query[0]) != "" {
		filter = strings.ToLower(strings.TrimSpace(query[0]))
	}

	if filter != "" {
		for _, q := range predefinedQuotes {
			if strings.Contains(strings.ToLower(q.Text), filter) ||
				strings.Contains(strings.ToLower(q.Author), filter) ||
				strings.Contains(strings.ToLower(q.Tag), filter) {
				pool = append(pool, q)
			}
		}
	}

	if len(pool) == 0 {
		pool = predefinedQuotes
	}

	idx := r.Intn(len(pool))
	q := pool[idx]
	return fmt.Sprintf("✨ *KATA MUTIARA HARI INI:*\n\n\"%s\"\n\n_— %s_", q.Text, q.Author)
}
