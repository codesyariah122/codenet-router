# CODENET Router OS

CODENET adalah fondasi sistem operasi router Linux independen yang dikembangkan oleh **CodeSyariah**. Repository ini berisi **CODENET Management API** dan **CODENET WebFig v0.1.1** dengan fokus monitoring read-only serta hardening autentikasi administrator.

> **Intelligent Network Operating System**

Status v0.1.1: dashboard, interface, PPPoE, bandwidth, dan status sistem tersedia melalui mock provider. Seluruh endpoint monitoring membutuhkan session administrator; hanya `/health` dan login yang publik. Linux provider membaca data sistem tanpa mengubah konfigurasi router. Operasi perubahan konfigurasi router belum diaktifkan.

## Arsitektur singkat

```text
Browser → WebFig (React) → Management API (Go) → RouterProvider
                                              ├── MockProvider (lokal)
                                              └── LinuxProvider (router)
```

- `apps/api`: REST API Go berbasis standard library.
- `apps/webfig`: React, TypeScript, Vite, Tailwind CSS, TanStack Query.
- `apps/api/openapi`: kontrak OpenAPI.
- `configs`: contoh konfigurasi non-rahasia untuk CODENET Router.
- `docs`: arsitektur, API, development, deployment, dan roadmap.

## Menjalankan secara lokal

Persyaratan: Go 1.23+, Node.js 20+, dan npm 10+.

Terminal pertama:

```bash
cd apps/api
go run ./cmd/codenet-admin init --username admin
go run .
```

Perintah init meminta password dua kali tanpa menampilkannya. Pilih password administrator sendiri dengan minimal 14 karakter, kombinasi huruf dan non-huruf. Untuk pemulihan lokal:

```bash
go run ./cmd/codenet-admin reset-password --username admin
```

CLI reset tidak tersedia melalui HTTP dan membatalkan seluruh session lama.

Terminal kedua:

```bash
cd apps/webfig
npm install
npm run dev
```

Buka `http://127.0.0.1:5173`. API secara default bind ke `127.0.0.1:8080` dan memakai mock provider. Vite meneruskan request `/api` ke API lokal.

## Pengujian dan build

```bash
./scripts/check.sh
```

Atau jalankan terpisah:

```bash
cd apps/api && go test ./...
cd apps/webfig && npm test && npm run build
```

## Mode Linux

Pada CODENET Router, pilih provider Linux secara eksplisit:

```bash
CODENET_PROVIDER=linux CODENET_API_ADDRESS=127.0.0.1:8080 go run .
```

Provider Linux hanya membaca `/proc`, `/sys`, informasi interface, status systemd, serta statistik TC dengan command yang sudah ditentukan. Provider tidak membaca `chap-secrets`, tidak mengubah nftables/jaringan, dan tidak me-restart layanan.

Lihat [panduan development](docs/development.md), [arsitektur](docs/architecture.md), dan [deployment manual](docs/deployment.md) untuk rincian lebih lanjut.

## Batas keamanan v0.1.1

- Jangan mengekspos API langsung ke internet.
- Jangan menjalankan API sebagai root tanpa pembatasan.
- Deployment ke router dilakukan manual setelah review.
- Session disimpan server-side di SQLite; cookie hanya berisi token acak yang tidak disimpan plaintext di database.
- Password administrator di-hash dengan Argon2id. Tidak ada endpoint pendaftaran publik atau password default.
- Cookie menggunakan HttpOnly dan SameSite=Strict; set `CODENET_COOKIE_SECURE=true` ketika API diakses melalui HTTPS.
- Request unsafe yang terautentikasi membutuhkan `X-CSRF-Token`; rate limit login berjalan per-user dan per-IP.
- Bind API tetap loopback secara default dan API tidak boleh dijalankan sebagai root.
- Tidak ada password PPPoE dalam model atau respons API.

Laporan audit tersedia di [docs/security-audit-v0.1.1.md](docs/security-audit-v0.1.1.md).

Lisensi belum ditetapkan. Seluruh hak produk dan identitas CODENET dimiliki CodeSyariah.
