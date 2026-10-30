# Development Lokal

## Backend

```bash
cd apps/api
go test ./...
go run .
```

Environment yang didukung:

| Variabel | Default | Fungsi |
|---|---|---|
| `CODENET_PROVIDER` | `mock` | `mock` atau `linux` |
| `CODENET_API_ADDRESS` | `127.0.0.1:8080` | Alamat bind API |
| `CODENET_ALLOWED_ORIGIN` | `http://127.0.0.1:5173` | Origin CORS tunggal |

Mode mock tidak membutuhkan Linux, root, systemd, tc, atau Accel-PPP.

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
