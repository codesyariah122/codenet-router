# Laporan Implementasi PPPoE Integration WebFig v0.3

Tanggal: 30 Oktober 2026  
Branch: `feature/webfig-pppoe-integration-v0.3`  
Baseline: `feature/webfig-linux-integration-v0.2` (`40344aa`)

## Ringkasan

v0.3 menambahkan fondasi monitoring Accel-PPP yang read-only dan fail-closed.
Adapter TCP langsung kini tersedia untuk CLI Accel-PPP yang telah diverifikasi
di Debian CODENET (`127.0.0.1:2001`, password file, perintah tetap `show
sessions`). Tanpa control interface yang dikonfigurasi, status menandai
monitoring tidak tersedia dan endpoint sesi mengembalikan HTTP 503.

## Arsitektur

- `AccelPPPMonitor` adalah kontrak provider read-only.
- `unavailableAccelPPPMonitor` dipakai secara default.
- `TCPAccelPPPMonitor` memakai koneksi TCP langsung, membaca password dari file
  owner-only, timeout 3 detik, output maksimum 2 MiB, lalu mengirim password,
  `show sessions`, dan `exit` agar koneksi ditutup oleh daemon. Password tidak
  pernah masuk argv, log, atau HTTP response.
- Parser tabel menerima delimiter `|` maupun whitespace, header/separator,
  sesi kosong, serta menolak respons rusak.
- `JSONSocketAccelPPPMonitor` mengirim JSON-lines `status`/`sessions`, dengan
  timeout 2 detik, batas respons 2 MiB, validasi data, dan tanpa shell.
- Linux Provider tetap membaca service melalui systemd fixed command.
- Sesi diperkaya dengan counter `/sys/class/net/<ppp>/statistics`.
- Perspektif counter: RX router = upload pelanggan; TX router = download
  pelanggan.

## Endpoint dan UI

- `GET /api/v1/pppoe/status` mengembalikan status service, access concentrator,
  interface server, status control interface, kesiapan jumlah sesi, dan sumber
  pool.
- `GET /api/v1/pppoe/sessions` mengembalikan username, IP, interface, uptime,
  RX/TX bytes, upload/download rate, dan status. HTTP 503 digunakan jika sumber
  monitoring gagal.
- Halaman PPPoE menyediakan search, filter interface, refresh, auto-refresh,
  detail, loading/error/empty state, serta membedakan control failure dari nol
  pelanggan.
- Halaman bandwidth tidak lagi menampilkan 10/5 Mbps sebagai fakta Linux.
  Profil kosong dan limiter `Tidak tersedia` ditampilkan jika belum dapat
  diverifikasi dari TC/Accel-PPP.
- Dashboard menampilkan jumlah sesi hanya jika `session_count_ready=true`.

## Persiapan pengelolaan pelanggan

Create/update/disable/delete user, assignment bandwidth profile, dan disconnect
session belum diimplementasikan. Desain berikutnya wajib memakai adapter write
terpisah, validasi, CSRF/otorisasi, audit log, preview, Safe Apply, dan rollback.
`chap-secrets` tidak boleh diubah otomatis.

## Pengujian

Backend:

- `go test ./internal/providers` — lulus.
- `go test -race ./internal/providers` — lulus.
- Test parser tabel TCP, password permission, autentikasi gagal, timeout, sesi
  kosong/rusak, counter reset, dan rate calculation — lulus.
- `go test ./...` dan `go test -race ./...` belum dapat diselesaikan pada
  workspace ini karena dependency `sqlite3`/`x/term` tidak tersedia di cache
  dan akses `proxy.golang.org` dibatasi; perintah wajib diulang pada host
  pengembangan dengan network/dependency cache.

Frontend:

- `npm test -- --run` — 8 file, 10 test lulus.
- `npm run build` — lulus.
- `git diff --check` — lulus.

Sesi Accel-PPP nyata belum dapat diuji dari workspace ini karena tidak ada
koneksi ke Debian CODENET. Uji mock TCP dan parser telah dijalankan.

## Panduan verifikasi Debian 13

1. Gunakan VM/snapshot dan user service non-root.
2. Verifikasi `systemctl`, `tc`, `/sys/class/net`, dan instalasi Accel-PPP.
3. Buat file password lokal, misalnya `/etc/codenet/accel-cli.password`,
   owner service API, mode `0600`; jangan masukkan password ke unit command.
4. Set environment berikut pada unit API:
   `CODENET_ACCEL_PPP_CLI_HOST=127.0.0.1`,
   `CODENET_ACCEL_PPP_CLI_PORT=2001`, dan
   `CODENET_ACCEL_PPP_CLI_PASSWORD_FILE=/etc/codenet/accel-cli.password`.
5. Login WebFig dan validasi status serta sesi terhadap `accel-cmd show
   sessions` dan counter sysfs.
   counter sysfs.
6. Pastikan tidak ada perubahan konfigurasi, restart service, modifikasi
   `chap-secrets`, atau perubahan TC selama pengujian.

## Risiko dan status

- Kontrak control adapter belum diverifikasi terhadap router Debian nyata.
- IP pool dan access concentrator Linux tidak akan ditampilkan bila sumbernya
  tidak tersedia.
- TC qdisc dapat terdeteksi tanpa detail limiter arah download/upload.
- Rate sesi pertama bernilai nol sampai baseline kedua tersedia.
- Interface PPP yang hilang ditandai `interface_missing`.
- API belum production-ready untuk operasi manajemen pelanggan.

Status: **adapter TCP siap untuk review kode dan uji VM terisolasi; integrasi
Accel-PPP belum dinyatakan selesai sebelum diuji dengan sesi PPPoE nyata.**
