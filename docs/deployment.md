# Deployment Manual ke CODENET Router

Dokumen ini adalah panduan persiapan, bukan otomasi deployment.

## Prasyarat

- Debian 13 x86_64.
- User service khusus tanpa login shell dan tanpa akses root umum.
- Binary `tc` serta akses baca `/proc` dan `/sys`.
- Kebijakan privilege terpisah bila pembacaan tertentu membutuhkan capability tambahan.
- Reverse proxy atau jaringan manajemen internal setelah autentikasi tersedia.

## Build artefak

```bash
cd apps/api
GOOS=linux GOARCH=amd64 go build -trimpath -o ../../dist/codenet-api .

cd ../webfig
npm ci
npm run build
```

Transfer artefak dilakukan oleh administrator setelah review. Jangan menyalin `.env`, database development, atau kredensial.

## Runtime yang disarankan

- Jalankan API sebagai `codenet-api`, bukan root.
- Set `CODENET_PROVIDER=linux`.
- Pertahankan bind `127.0.0.1:8080` sampai reverse proxy dan autentikasi siap.
- Gunakan filesystem read-only sejauh memungkinkan.
- Batasi executable yang dapat dipanggil oleh service.

Contoh unit di `configs/systemd/codenet-api.service` harus ditinjau dan disesuaikan manual. Tidak ada script repository yang mengaktifkan atau me-restart unit.

## Batas v0.1

Pembacaan sesi Accel-PPP nyata belum diaktifkan karena kontrak socket kontrol perlu divalidasi terhadap instalasi router. API akan mengembalikan daftar kosong, bukan membuat data palsu. Statistik TC mendeteksi qdisc pada interface PPP, tetapi parsing detail byte-rate masih menjadi roadmap.
