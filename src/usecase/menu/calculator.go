package menu

import (
	"fmt"
	"strconv"
	"strings"
)

func SimpleCalculate(expr string) (string, error) {
	clean := strings.TrimSpace(expr)
	if clean == "" {
		return "⚠️ Masukkan ekspresi matematika.\nContoh: *!calc 25 * 4 + 10*", nil
	}

	clean = strings.ReplaceAll(clean, "x", "*")
	clean = strings.ReplaceAll(clean, "X", "*")
	clean = strings.ReplaceAll(clean, ":", "/")

	parts := strings.Fields(clean)
	if len(parts) == 3 {
		n1, err1 := strconv.ParseFloat(parts[0], 64)
		op := parts[1]
		n2, err2 := strconv.ParseFloat(parts[2], 64)
		if err1 == nil && err2 == nil {
			var res float64
			switch op {
			case "+":
				res = n1 + n2
			case "-":
				res = n1 - n2
			case "*":
				res = n1 * n2
			case "/":
				if n2 == 0 {
					return "⚠️ Pembagian dengan nol tidak dapat dilakukan.", nil
				}
				res = n1 / n2
			default:
				return fmt.Sprintf("🔢 *Ekspresi:* %s", clean), nil
			}
			return fmt.Sprintf("🔢 *Hasil Hitung:* %g", res), nil
		}
	}

	return fmt.Sprintf("🔢 *Ekspresi:* %s", clean), nil
}
