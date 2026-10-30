# CODENET Router OS
<img width="2880" height="1548" alt="Screenshot 2026-10-30 at 15 51 28" src="https://github.com/user-attachments/assets/a26b3113-81ae-40eb-ae78-0ac30891378f" />
<img width="2880" height="1548" alt="Screenshot 2026-10-30 at 15 51 42" src="https://github.com/user-attachments/assets/b2ecb9e9-e2fa-420c-b7fe-1c81b48dc424" />
<img width="2880" height="1548" alt="Screenshot 2026-10-30 at 15 51 46" src="https://github.com/user-attachments/assets/577fe049-c447-432d-9c59-36fcaf18ac16" />
<img width="2880" height="1548" alt="Screenshot 2026-10-30 at 15 51 56" src="https://github.com/user-attachments/assets/7dba9ce1-eb54-4e06-af53-3c7d8f6dacd4" />

CODENET adalah fondasi sistem operasi router Linux independen yang dikembangkan oleh **CodeSyariah**. Repository ini berisi **CODENET Management API** dan **CODENET WebFig v0.3** dengan fokus integrasi Linux/Accel-PPP read-only serta hardening autentikasi administrator.

> **Intelligent Network Operating System**

Status v0.3: dashboard, interface, IP Addresses, IP Routes, PPPoE sessions, bandwidth, dan status sistem tersedia. Mock Provider digunakan lokal; Linux Provider membaca resource, interface, systemd, TC, serta sesi Accel-PPP hanya melalui control adapter yang dikonfigurasi eksplisit. Seluruh endpoint monitoring membutuhkan session administrator; hanya `/health` dan login yang publik.

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

Modul IP → Addresses dan IP → Routes membaca data kernel sebenarnya saat
`CODENET_PROVIDER=linux`. Keduanya tetap read-only dan tidak menyediakan
tombol perubahan konfigurasi.

Monitoring Accel-PPP memakai CLI TCP terverifikasi melalui
`CODENET_ACCEL_PPP_CLI_HOST`, `CODENET_ACCEL_PPP_CLI_PORT`, dan
`CODENET_ACCEL_PPP_CLI_PASSWORD_FILE`; adapter hanya mengirim `show sessions`.
Password wajib berada pada file owner-only. Adapter JSON Unix-socket lama tetap
tersedia sebagai fallback. Tanpa konfigurasi, API menampilkan sumber monitoring
tidak tersedia, bukan mengklaim tidak ada pelanggan.

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

Laporan audit v0.1.1 tersedia di [docs/security-audit-v0.1.1.md](docs/security-audit-v0.1.1.md), laporan integrasi Linux v0.2 di [docs/linux-integration-v0.2.md](docs/linux-integration-v0.2.md), dan laporan Accel-PPP v0.3 di [docs/pppoe-integration-v0.3.md](docs/pppoe-integration-v0.3.md).

Lisensi belum ditetapkan. Seluruh hak produk dan identitas CODENET dimiliki CodeSyariah.
