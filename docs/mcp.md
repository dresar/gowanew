# Dokumentasi Lengkap Model Context Protocol (MCP) GoWA

Dokumentasi ini dirancang agar **Model AI (Claude, Cursor, Windsurf, OpenCode, Antigravity, dsb.)** maupun developer dapat memahami 100% cara berkomunikasi, mengontrol, dan mengeksekusi operasi CRUD (*Create, Read, Update, Delete*) pada engine WhatsApp Multi-Device.

---

## 1. Arsitektur & Spesifikasi Protokol

GoWA menyediakan server MCP resmi berbasis spesifikasi **Model Context Protocol (MCP v2025)** dengan dua transport:

1. **Streamable HTTP (JSON-RPC 2.0)**:
   - **Local Endpoint**: `http://localhost:3000/mcp`
   - **VPS Endpoint**: `http://103.253.213.185:3000/mcp`
   - **Metode**: `POST`
   - **Headers Wajib**:
     - `Content-Type: application/json`
     - `Accept: application/json, text/event-stream`
     - `X-Device-Id: <device_id>` *(Opsional: jika tidak diisi, otomatis menggunakan default device yang sedang aktif)*

2. **STDIO Runner (Node.js Proxy)**:
   - **Script Path**: `tools/mcp-runner/runner.mjs`
   - **Eksekusi**: `node tools/mcp-runner/runner.mjs`
   - **Environment**:
     - `GOWA_MCP_URL=http://localhost:3000/mcp`
     - `GOWA_DEVICE_ID=<device_id>` *(opsional)*

---

## 2. Struktur Pesan JSON-RPC 2.0

Semua interaksi MCP menggunakan format standar JSON-RPC 2.0.

### A. Format Request Standar
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "<nama_tool>",
    "arguments": {
      "<parameter>": "<nilai>"
    }
  }
}
```

### B. Format Response Standar
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Deskripsi atau ringkasan hasil"
      }
    ],
    "structuredContent": {
      "data": "Objek JSON hasil eksekusi"
    }
  }
}
```

---

## 3. Katalog Lengkap 8 Tools MCP & Panduan CRUD

Berikut adalah 8 tool yang terdaftar pada `tools/list`:

### 🛠️ 1. `whatsapp_send` — Mengirim Pesan & Media
Tool serbaguna untuk mengirim semua jenis pesan ke nomor pribadi maupun grup.

#### Parameter Utama:
- `phone` (string, wajib): Nomor tujuan dengan kode negara (contoh: `6281234567890@s.whatsapp.net`) atau ID grup (contoh: `120363xxx@g.us`).
- `type` (string, wajib): `text`, `image`, `video`, `audio`, `document`, `sticker`, `location`, `contact`, `poll`, `link`, `forward`.
- `message` (string): Isi teks pesan (wajib jika `type="text"`).
- `image_url` (string): URL gambar publik atau lokal (wajib jika `type="image"`).
- `video_url` (string): URL file video MP4 (wajib jika `type="video"`).
- `audio_url` (string): URL file audio/MP3 (wajib jika `type="audio"`).
- `ptt` (boolean): `true` untuk mengirim audio sebagai pesan suara (*voice note*).
- `file_url` (string): URL dokumen PDF/DOCX/ZIP (wajib jika `type="document"`).
- `sticker_url` (string): URL gambar untuk dikonversi otomatis menjadi stiker WebP.
- `latitude` & `longitude` (string): Koordinat GPS (wajib jika `type="location"`).
- `contact_name` & `contact_phone` (string): Kontak vCard (wajib jika `type="contact"`).
- `question` & `options` (array of string): Polling interaktif (wajib jika `type="poll"`).
- `reply_message_id` (string): ID pesan yang ingin dibalas (*quote reply*).
- `mentions` (array of string): Daftar JID yang di-mention atau `["@everyone"]` untuk mention seluruh anggota grup.

#### Contoh Call (Kirim Teks):
```json
{
  "jsonrpc": "2.0",
  "id": 101,
  "method": "tools/call",
  "params": {
    "name": "whatsapp_send",
    "arguments": {
      "phone": "6281234567890@s.whatsapp.net",
      "type": "text",
      "message": "Halo! Ini pesan otomatis dari AI melalui MCP."
    }
  }
}
```

---

### 🛠️ 2. `whatsapp_chat` — Membaca Percakapan & Kontak (READ)
Digunakan untuk query percakapan, membaca pesan masuk, dan mengambil kontak.

#### Parameter Utama:
- `action` (string, wajib): `list_chats`, `list_contacts`, `get_messages`.
- `chat_jid` (string): JID chat target (wajib jika `action="get_messages"`).
- `limit` (integer, opsional, default 25): Jumlah data yang diambil.
- `offset` (integer, opsional, default 0): Paginasi data.
- `search` (string, opsional): Pencarian nama atau nomor.

#### Contoh Call (Membaca Daftar Chat Terkini):
```json
{
  "jsonrpc": "2.0",
  "id": 102,
  "method": "tools/call",
  "params": {
    "name": "whatsapp_chat",
    "arguments": {
      "action": "list_chats",
      "limit": 10
    }
  }
}
```

#### Contoh Call (Mengambil Riwayat Pesan dari Kontak):
```json
{
  "jsonrpc": "2.0",
  "id": 103,
  "method": "tools/call",
  "params": {
    "name": "whatsapp_chat",
    "arguments": {
      "action": "get_messages",
      "chat_jid": "6281234567890@s.whatsapp.net",
      "limit": 20
    }
  }
}
```

---

### 🛠️ 3. `whatsapp_message` — Operasi Pesan (Reaksi, Edit, Hapus / UPDATE & DELETE)
Digunakan untuk memodifikasi atau menghapus pesan yang sudah ada.

#### Parameter Utama:
- `action` (string, wajib): `react`, `star`, `edit`, `revoke`, `delete`.
- `phone` (string, wajib): JID pengirim/chat.
- `message_id` (string, wajib): ID pesan target.
- `emoji` (string): Karakter emoji (wajib jika `action="react"`, contoh: `"👍"` atau `""` untuk menghapus reaksi).
- `message` (string): Teks baru (wajib jika `action="edit"`).

#### Contoh Call (Memberi Reaksi Emoji):
```json
{
  "jsonrpc": "2.0",
  "id": 104,
  "method": "tools/call",
  "params": {
    "name": "whatsapp_message",
    "arguments": {
      "action": "react",
      "phone": "6281234567890@s.whatsapp.net",
      "message_id": "3EB0A1B2C3D4",
      "emoji": "🔥"
    }
  }
}
```

#### Contoh Call (Revoke / Tarik Pesan untuk Semua Orang - DELETE):
```json
{
  "jsonrpc": "2.0",
  "id": 105,
  "method": "tools/call",
  "params": {
    "name": "whatsapp_message",
    "arguments": {
      "action": "revoke",
      "phone": "6281234567890@s.whatsapp.net",
      "message_id": "3EB0A1B2C3D4"
    }
  }
}
```

---

### 🛠️ 4. `whatsapp_group` — Manajemen Grup & Komunitas
Digunakan untuk membuat grup, mengelola anggota, dan hak akses admin.

#### Parameter Utama:
- `action` (string, wajib): `create`, `info`, `participants`, `join_with_link`, `leave`, `invite_link`.
- `title` (string): Nama grup baru (wajib jika `action="create"`).
- `group_jid` (string): ID grup WhatsApp (contoh: `120363xxx@g.us`).
- `operation` (string): `add`, `remove`, `promote`, `demote` (wajib jika `action="participants"`).
- `participants` (array of string): Daftar nomor yang dimanipulasi.

#### Contoh Call (Membuat Grup Baru - CREATE):
```json
{
  "jsonrpc": "2.0",
  "id": 106,
  "method": "tools/call",
  "params": {
    "name": "whatsapp_group",
    "arguments": {
      "action": "create",
      "title": "Komunitas AI Engineer",
      "participants": ["6281234567890@s.whatsapp.net"]
    }
  }
}
```

#### Contoh Call (Mengeluarkan Anggota dari Grup - DELETE):
```json
{
  "jsonrpc": "2.0",
  "id": 107,
  "method": "tools/call",
  "params": {
    "name": "whatsapp_group",
    "arguments": {
      "action": "participants",
      "group_jid": "120363123456@g.us",
      "operation": "remove",
      "participants": ["6281234567890@s.whatsapp.net"]
    }
  }
}
```

---

### 🛠️ 5. `whatsapp_bot` — CRUD Aturan Auto-Reply & Monitoring Log
Tool utama untuk mengontrol otomatisasi kecerdasan bot secara penuh.

#### Siklus Lengkap CRUD:

#### [CREATE] Tambah Aturan Auto-Reply Baru:
```json
{
  "jsonrpc": "2.0",
  "id": 108,
  "method": "tools/call",
  "params": {
    "name": "whatsapp_bot",
    "arguments": {
      "action": "create_rule",
      "trigger_value": "!bantuan",
      "response_content": "Halo! Silakan ketik !menu untuk melihat daftar layanan."
    }
  }
}
```

#### [READ] Baca Seluruh Aturan yang Terdaftar:
```json
{
  "jsonrpc": "2.0",
  "id": 109,
  "method": "tools/call",
  "params": {
    "name": "whatsapp_bot",
    "arguments": {
      "action": "list_rules"
    }
  }
}
```

#### [UPDATE] Toggle Aktif / Nonaktif Aturan:
```json
{
  "jsonrpc": "2.0",
  "id": 110,
  "method": "tools/call",
  "params": {
    "name": "whatsapp_bot",
    "arguments": {
      "action": "toggle_rule",
      "rule_id": 824
    }
  }
}
```

#### [DELETE] Hapus Aturan Secara Permanen:
```json
{
  "jsonrpc": "2.0",
  "id": 111,
  "method": "tools/call",
  "params": {
    "name": "whatsapp_bot",
    "arguments": {
      "action": "delete_rule",
      "rule_id": 824
    }
  }
}
```

#### [MONITOR] Baca Log Aktivitas Bot:
```json
{
  "jsonrpc": "2.0",
  "id": 112,
  "method": "tools/call",
  "params": {
    "name": "whatsapp_bot",
    "arguments": {
      "action": "query_logs",
      "limit": 20
    }
  }
}
```

---

### 🛠️ 6. `whatsapp_schedule` — Penjadwalan Pesan Terprogram
Mengatur pengiriman pesan otomatis di waktu tertentu atau berulang.

#### Parameter Utama:
- `action` (string, wajib): `list`, `get`, `pause`, `resume`, `cancel`.
- `schedule_id` (string): ID pesan terjadwal.
- `status` (string, opsional): Filter status (`active`, `paused`, `completed`, `cancelled`).

#### Contoh Call (Melihat Daftar Antrean Pesan Terjadwal):
```json
{
  "jsonrpc": "2.0",
  "id": 113,
  "method": "tools/call",
  "params": {
    "name": "whatsapp_schedule",
    "arguments": {
      "action": "list",
      "status": "active"
    }
  }
}
```

#### Contoh Call (Membatalkan Pesan Terjadwal - DELETE):
```json
{
  "jsonrpc": "2.0",
  "id": 114,
  "method": "tools/call",
  "params": {
    "name": "whatsapp_schedule",
    "arguments": {
      "action": "cancel",
      "schedule_id": "SCH_987654321"
    }
  }
}
```

---

### 🛠️ 7. `whatsapp_webhook` — Kelola & Uji Webhook Real-Time
Digunakan untuk membaca, mengupdate, dan mengetes transmisi payload event webhook.

#### Parameter Utama:
- `action` (string, wajib): `get`, `set`, `test`.
- `webhook_url` (string): Endpoint HTTP/HTTPS tujuan.
- `webhook_secret` (string): Kunci rahasia HMAC-SHA256.
- `webhook_events` (string): Daftar event dipisahkan koma (contoh: `message,message.ack,group.participants`).
- `insecure_skip_verify` (boolean): Lewati validasi SSL/TLS untuk local dev.
- `event` (string, opsional): Tipe event simulasi (default: `message`).

#### Contoh Call (Mengubah Pengaturan Webhook - UPDATE):
```json
{
  "jsonrpc": "2.0",
  "id": 115,
  "method": "tools/call",
  "params": {
    "name": "whatsapp_webhook",
    "arguments": {
      "action": "set",
      "webhook_url": "https://api.perusahaan.com/webhook/wa",
      "webhook_secret": "9f8e7d6c5b4a3210",
      "webhook_events": "message,message.ack,group.participants",
      "insecure_skip_verify": false
    }
  }
}
```

#### Contoh Call (Mengirim Tes Simulasi Webhook):
```json
{
  "jsonrpc": "2.0",
  "id": 116,
  "method": "tools/call",
  "params": {
    "name": "whatsapp_webhook",
    "arguments": {
      "action": "test",
      "event": "message"
    }
  }
}
```

---

### 🛠️ 8. `whatsapp_app` — Status Perangkat & Sesi WhatsApp
Digunakan untuk melihat kondisi sesi, meminta QR login atau pairing code, dan sinkronisasi.

#### Parameter Utama:
- `action` (string, wajib): `status`, `login_qr`, `login_code`, `logout`, `reconnect`, `sync_contacts`.
- `phone` (string): Nomor telepon untuk pairing code (wajib jika `action="login_code"`).

#### Contoh Call (Mengecek Status Koneksi & Perangkat):
```json
{
  "jsonrpc": "2.0",
  "id": 117,
  "method": "tools/call",
  "params": {
    "name": "whatsapp_app",
    "arguments": {
      "action": "status"
    }
  }
}
```

#### Contoh Call (Meminta Pairing Code 8 Digit untuk Login Nomor Baru):
```json
{
  "jsonrpc": "2.0",
  "id": 118,
  "method": "tools/call",
  "params": {
    "name": "whatsapp_app",
    "arguments": {
      "action": "login_code",
      "phone": "6281234567890"
    }
  }
}
```

---

## 4. Panduan Prompt Sistem untuk AI Agent (System Prompt Injection)

Salin teks berikut ke instruksi agen AI Anda (misal: Claude Projects, Cursor Custom Rule, atau Cherry Studio Agent Prompt) agar AI mengetahui batasan dan cara menggunakan alat-alat ini secara cerdas:

```text
Anda adalah AI Coding & Operations Assistant yang terhubung langsung ke WhatsApp Multi-Device Gateway melalui MCP Server (Model Context Protocol).

Anda memiliki akses ke 8 tools WhatsApp berikut:
1. whatsapp_send: Kirim pesan teks, gambar, video, dokumen, stiker, polling, atau lokasi. Selalu gunakan format nomor internasional tanpa tanda tambah (contoh: "6281234567890@s.whatsapp.net").
2. whatsapp_chat: Ambil riwayat chat (get_messages), daftar obrolan aktif (list_chats), atau kontak.
3. whatsapp_message: Reaksi emoji (react), edit teks pesan (edit), atau tarik pesan untuk semua orang (revoke).
4. whatsapp_group: Buat grup (create), kelola anggota (participants: add/remove/promote/demote), atau ambil link undangan (invite_link).
5. whatsapp_bot: Kelola aturan balasan otomatis (create_rule, list_rules, toggle_rule, delete_rule, query_logs).
6. whatsapp_schedule: Jadwalkan pesan otomatis di masa mendatang (list, get, cancel).
7. whatsapp_webhook: Periksa atau ubah URL webhook dan kirim paket uji coba simulasi (get, set, test).
8. whatsapp_app: Periksa status koneksi perangkat (status), minta kode pairing (login_code), atau logout.

Aturan Penting:
- Jangan pernah mengirim pesan berulang tanpa instruksi eksplisit pengguna.
- Untuk nomor telepon, selalu tambahkan domain "@s.whatsapp.net" jika belum ada.
- Untuk grup WhatsApp, ID berakhiran "@g.us".
- Lakukan konfirmasi sebelum mengeksekusi operasi destruktif seperti revoke pesan atau remove peserta grup.
```

---

## 5. Ringkasan Endpoint Cepat

| Endpoint | Protokol | Contoh Request |
| :--- | :--- | :--- |
| `POST /mcp` | Streamable HTTP | `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}` |
| `POST /mcp` | Streamable HTTP | `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"whatsapp_send","arguments":{...}}}` |
| `node tools/mcp-runner/runner.mjs` | Stdio CLI Proxy | Pipe JSON-RPC line via stdin &rarr; Output JSON-RPC via stdout |
