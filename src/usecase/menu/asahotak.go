package menu

import (
	"fmt"
	"math/rand"
)

var brainTeasers = []struct {
	Question string
	Answer   string
}{
	{Question: "Apa yang selalu datang tapi tidak pernah tiba?", Answer: "Besok (hari esok)"},
	{Question: "Memiliki banyak gigi tapi tidak bisa menggigit, benda apakah itu?", Answer: "Sisir rambut"},
	{Question: "Benda apa yang jika diisi semakin ringan?", Answer: "Balon udara"},
	{Question: "Semakin dipotong semakin panjang, apakah itu?", Answer: "Celana panjang yang dipotong bagian bawahnya (atau tali)"},
	{Question: "Benda apa yang selalu berjalan tapi tidak punya kaki?", Answer: "Jam dinding / Jam tangan"},
}

func FetchRandomBrainTeaser() string {
	idx := rand.Intn(len(brainTeasers))
	bt := brainTeasers[idx]
	return fmt.Sprintf("🧠 *KUIS ASAH OTAK:*\n\n*%s*\n\n_Kunci Jawaban: %s_", bt.Question, bt.Answer)
}
