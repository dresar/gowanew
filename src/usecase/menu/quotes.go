package menu

import (
	"fmt"
	"math/rand"
)

var predefinedQuotes = []struct {
	Text   string
	Author string
}{
	{Text: "Satu-satunya cara untuk melakukan pekerjaan hebat adalah dengan mencintai apa yang Anda lakukan.", Author: "Steve Jobs"},
	{Text: "Kesuksesan adalah kemampuan untuk beralih dari satu kegagalan ke kegagalan lain tanpa kehilangan antusiasme.", Author: "Winston Churchill"},
	{Text: "Jangan menunggu kesempatan luar biasa. Raih kesempatan biasa dan jadikan itu luar biasa.", Author: "Orison Swett Marden"},
	{Text: "Masa depan adalah milik mereka yang percaya pada keindahan impian mereka.", Author: "Eleanor Roosevelt"},
	{Text: "Lakukan yang terbaik hari ini, agar Anda tidak menyesal esok hari.", Author: "Anonim"},
}

func FetchRandomQuote() string {
	idx := rand.Intn(len(predefinedQuotes))
	q := predefinedQuotes[idx]
	return fmt.Sprintf("✨ *KATA MUTIARA HARI INI:*\n\n\"%s\"\n\n_— %s_", q.Text, q.Author)
}
