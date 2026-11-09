# Deployment Manual ke CODENET Router

Dokumen ini adalah panduan persiapan, bukan otomasi deployment.

## Prasyarat

- Debian 13 x86_64.
- User service khusus tanpa login shell dan tanpa akses root umum.
- Akun administrator diinisialisasi manual melalui CLI lokal sebelum service API dijalankan.
- Binary `tc` serta akses baca `/proc` dan `/sys`.
- Binary `ip` iproute2 dan Accel-PPP 1.14.x dengan CLI TCP terverifikasi pada `127.0.0.1:2001`.
- Kebijakan privilege terpisah bila pembacaan tertentu membutuhkan capability tambahan.
- Reverse proxy atau jaringan manajemen internal setelah autentikasi tersedia.

## Build artefak

```bash
cd apps/api
CGO_ENABLED=1 go build -trimpath -o ../../dist/codenet-api .
CGO_ENABLED=1 go build -trimpath -o ../../dist/codenet-admin ./cmd/codenet-admin

cd ../webfig
npm ci
npm run build
```

Driver SQLite menggunakan CGO. Jangan menggunakan binary `CGO_ENABLED=0`
untuk deployment: binary dapat lolos kompilasi tetapi tidak memiliki driver
SQLite yang berfungsi saat runtime.

Transfer artefak dilakukan oleh administrator setelah review. Jangan menyalin `.env`, database development, atau kredensial.

## Runtime yang disarankan

- Jalankan API sebagai `codenet-api`, bukan root.
- Set `CODENET_PROVIDER=linux`.
- Buat password file CLI Accel-PPP dengan owner service API dan mode `0600`.
  Set `CODENET_ACCEL_PPP_CLI_HOST=127.0.0.1`, `CODENET_ACCEL_PPP_CLI_PORT=2001`,
  dan `CODENET_ACCEL_PPP_CLI_PASSWORD_FILE=/etc/codenet/accel-cli.password`.
  Password tidak boleh berada pada argumen proses atau unit command.
- Adapter JSON Unix socket lama tetap tersedia sebagai fallback kompatibilitas.
- Set `CODENET_DB_PATH=/var/lib/codenet/codenet.db` dan jalankan `codenet-admin init` sebagai user service/administrator lokal sebelum start API.
- Set `CODENET_COOKIE_SECURE=true` hanya saat WebFig diakses melalui HTTPS.
- Pertahankan bind `127.0.0.1:8080` sampai reverse proxy dan autentikasi siap.
- Gunakan filesystem read-only sejauh memungkinkan.
- Batasi executable yang dapat dipanggil oleh service.

Contoh unit di `configs/systemd/codenet-api.service` harus ditinjau dan disesuaikan manual. Tidak ada script repository yang mengaktifkan atau me-restart unit.

Unit contoh memakai `StateDirectory=codenet`, `NoNewPrivileges`, capability set kosong, `ProtectSystem=strict`, dan bind loopback. API tidak membutuhkan root. Jangan menambahkan `sudo` ke handler atau memberi capability `CAP_NET_ADMIN`; informasi `/proc`, `/sys`, interface, systemd, dan TC dibaca dengan privilege service yang tersedia.

## Batas v0.3

IP Addresses dan IP Routes membaca kernel secara read-only melalui provider
Linux. Tidak ada operasi konfigurasi IP/route. Pembacaan sesi Accel-PPP nyata
belum diaktifkan karena kontrak socket kontrol perlu divalidasi terhadap
instalasi router. API akan mengembalikan daftar kosong, bukan membuat data
palsu. Statistik TC mendeteksi qdisc pada interface PPP, tetapi parsing detail
byte-rate masih menjadi roadmap. Integrasi ke Debian 13 harus dilakukan setelah
audit VM dan pengujian privilege terisolasi; tidak termasuk dalam task ini.

Sesi PPPoE hanya dapat dibaca jika control adapter tersedia. Adapter TCP hanya
mengirim password, perintah tetap `show sessions`, lalu `exit` untuk menutup
sesi CLI, dengan batas respons 2 MiB dan timeout. Tanpa adapter,
`/api/v1/pppoe/status` tetap melaporkan service tetapi menandai control
interface tidak tersedia, sedangkan `/api/v1/pppoe/sessions` mengembalikan
HTTP 503. Tidak ada restart service, perubahan Accel-PPP, atau pembacaan
`chap-secrets`.

## Helper PPPoE User v0.4

Build helper secara terpisah:

```bash
cd apps/api
CGO_ENABLED=0 go build -trimpath -o ../../dist/codenet-pppoe-helper ./cmd/codenet-pppoe-helper
```

Pada Debian, buat secret IPC mode `0600`, jalankan helper sebagai service
terpisah dengan akses terbatas, dan set `CODENET_PPPOE_HELPER_SOCKET` serta
`CODENET_PPPOE_HELPER_SECRET_FILE` pada API. Helper hanya mengizinkan
`/etc/codenet/chap-secrets`, memakai lock dan atomic write, membuat backup
`.bak`, serta tidak melakukan reload atau disconnect otomatis. Uji dahulu
terhadap fixture/snapshot; jangan menulis file router asli dari workstation.
Untuk pengujian Debian tanpa file produksi, gunakan unit terpisah
`configs/systemd/codenet-pppoe-helper-fixture.service`. Mode `--fixture`
menolak root dan hanya menerima path di bawah
`/var/lib/codenet-v04/helper-test/`.
