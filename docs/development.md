# Development Lokal

## Backend

```bash
cd apps/api
go run ./cmd/codenet-admin init --username admin
go test ./...
go test -race ./...
go run .
```

Environment yang didukung:

| Variabel | Default | Fungsi |
|---|---|---|
| `CODENET_PROVIDER` | `mock` | `mock` atau `linux` |
| `CODENET_API_ADDRESS` | `127.0.0.1:8080` | Alamat bind API |
| `CODENET_ALLOWED_ORIGIN` | `http://127.0.0.1:5173` | Origin CORS tunggal |
| `CODENET_DB_PATH` | `data/codenet.db` | SQLite admin/session |
| `CODENET_COOKIE_SECURE` | `false` | `true` hanya jika browser memakai HTTPS |
| `CODENET_SESSION_TTL` | `8h` | Masa berlaku session |
| `CODENET_LOGIN_MAX_ATTEMPTS` | `5` | Batas gagal login per user/IP |
| `CODENET_LOGIN_WINDOW` | `15m` | Jendela rate limiting |
| `CODENET_ACCEL_PPP_CONTROL_SOCKET` | kosong | Path Unix socket control adapter JSON Accel-PPP yang sudah diverifikasi |
| `CODENET_ACCEL_PPP_CLI_HOST` | `127.0.0.1` | Host CLI TCP Accel-PPP |
| `CODENET_ACCEL_PPP_CLI_PORT` | `2001` | Port CLI TCP Accel-PPP |
| `CODENET_ACCEL_PPP_CLI_PASSWORD_FILE` | kosong | File password owner-only mode 0600 |

Mode mock tidak membutuhkan Linux, root, systemd, tc, atau Accel-PPP.

Untuk memeriksa provider Linux tanpa mengubah router, jalankan test pada host
Linux/VM terisolasi. Endpoint IP memakai `GET /api/v1/ip/addresses` dan
`GET /api/v1/ip/routes`; keduanya tetap membutuhkan session administrator.

Adapter TCP aktif jika password file diisi dan hanya mengirim perintah tetap
`show sessions`, lalu `exit` sebagai terminator sesi. Jangan menyimpan password
di environment atau command-line.
Adapter JSON fallback memakai kontrak berikut. Jangan mengisi variabel
socket dengan nama/format yang belum diverifikasi. Kontrak adapter mengirim
baris JSON `{"action":"status"}` atau `{"action":"sessions"}` dan menerima
respons JSON ber-envelope `success`/`data`; helper yang menerapkan kontrak ini
harus read-only dan dijalankan tanpa root.

Jangan memakai password contoh pada dokumentasi. Password init/reset dibaca tanpa echo; gunakan `--password-stdin` hanya jika stdin dikelola dengan aman dan berisi dua baris password yang sama. Database development otomatis dibuat dengan permission `0600` dan sudah di-ignore Git.

## Frontend

```bash
cd apps/webfig
npm install
npm run dev
```

Vite menyediakan proxy lokal. Untuk API lain, salin `.env.example` ke `.env.local` dan ubah `VITE_API_URL`.

## Konvensi perubahan

- Tambahkan domain provider lebih dulu, lalu service, handler, OpenAPI, dan UI.
- Jangan mengakses backend langsung dari page; gunakan `src/services/api.ts`.
- Loading, error, dan empty state wajib terlihat jelas.
- Gunakan data mock yang diberi konteks development, bukan klaim kondisi router nyata.
- Jalankan `./scripts/check.sh` sebelum commit.
