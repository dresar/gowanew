package menu

import (
	"fmt"
	"math/rand"
)

var pantunList = []string{
	"Beli pulsa di toko Pak Rahmat,\nPulang ke rumah disambut senyuman.\nAwali hari dengan semangat,\nSemoga sukses dalam genggaman.",
	"Jalan-jalan ke Kota Blitar,\nJangan lupa membeli sukun.\nJika kamu ingin pintar,\nBelajarlah dengan tekun.",
	"Pohon beringin rindang daunnya,\nTempat berteduh sang gembala.\nBekerjalah dengan sepenuh jiwa,\nRezeki halal membawa pahala.",
	"Pergi ke pasar membeli nangka,\nNangka dibelah manis rasanya.\nTetaplah ramah kepada sesama,\nHidup bahagia damai selamanya.",
	"Burung dara terbang melayang,\nHinggap sebentar di dahan cemara.\nKepada sahabat selalu sayang,\nHati tenang tiada duka.",
}

func FetchRandomPantun() string {
	idx := rand.Intn(len(pantunList))
	return fmt.Sprintf("📜 *PANTUN NUSANTARA:*\n\n%s", pantunList[idx])
}
