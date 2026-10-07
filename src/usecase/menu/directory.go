package menu

import (
	"fmt"
	"strings"
)

type DirectoryConfig struct {
	CheckNumber bool
	UserInfo    bool
	Avatar      bool
	Business    bool
}

func FormatDirectoryHelp() string {
	var sb strings.Builder
	sb.WriteString("📇 *MENU DIREKTORI & KONTAK*\n\n")
	sb.WriteString("Perintah yang tersedia:\n")
	sb.WriteString("• *!cek <nomor>* : Cek registrasi WhatsApp nomor\n")
	sb.WriteString("• *!info <nomor>* : Cek info detail akun / status\n")
	sb.WriteString("• *!avatar <nomor>* : Ambil foto profil WhatsApp\n")
	sb.WriteString("• *!bisnis <nomor>* : Cek profil WhatsApp bisnis\n\n")
	sb.WriteString("_Contoh: !cek 628123456789_")
	return sb.String()
}

func FormatDirectoryLookup(query string) string {
	q := strings.TrimSpace(query)
	if q == "" {
		return FormatDirectoryHelp()
	}
	return fmt.Sprintf("🔍 *PENCARIAN DIREKTORI:*\nNomor: %s\nStatus: Terverifikasi di WhatsApp", q)
}
