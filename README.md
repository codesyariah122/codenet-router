# CODENET Router OS

CODENET adalah fondasi sistem operasi router Linux independen yang dikembangkan oleh **CodeSyariah**. Repository ini berisi **CODENET Management API** dan **CODENET WebFig v0.1** dengan fokus awal pada monitoring read-only.

> **Intelligent Network Operating System**

Status v0.1: dashboard, interface, PPPoE, bandwidth, dan status sistem tersedia melalui mock provider. Linux provider membaca data sistem tanpa mengubah konfigurasi router. Fitur autentikasi produksi dan operasi konfigurasi belum diaktifkan.

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
go run .
```

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

## Batas keamanan v0.1

- Jangan mengekspos API langsung ke internet.
- Jangan menjalankan API sebagai root tanpa pembatasan.
- Deployment ke router dilakukan manual setelah review.
- Autentikasi session + HttpOnly Cookie adalah pekerjaan tahap berikutnya; sebelum tersedia, bind hanya pada loopback atau jaringan manajemen yang dilindungi.
- Tidak ada password PPPoE dalam model atau respons API.

Lisensi belum ditetapkan. Seluruh hak produk dan identitas CODENET dimiliki CodeSyariah.
