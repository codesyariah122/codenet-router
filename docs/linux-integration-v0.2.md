# Laporan Implementasi Linux Integration WebFig v0.2

Tanggal: 30 Oktober 2026  
Branch: `feature/webfig-linux-integration-v0.2`  
Baseline: `security/webfig-auth-hardening-v0.1.1` (`836dcbd`)

## Ringkasan

WebFig v0.2 menambahkan integrasi Linux Provider read-only untuk alamat IP,
routing table, resource CPU/memori, interface, systemd, dan statistik traffic.
Autentikasi v0.1.1 dipertahankan: seluruh endpoint monitoring tetap memakai
session cookie administrator dan endpoint mutasi jaringan tidak dibuat.

## Perubahan backend

- Model `IPAddress` dan `Route` ditambahkan.
- `RouterProvider`, `RouterService`, Mock Provider, dan Linux Provider diperluas
  dengan `IPAddresses` serta `Routes`.
- Linux Provider membaca alamat melalui `net.Interfaces`/`Interface.Addrs` dan
  route melalui executable tetap iproute2 dengan format JSON.
- Dukungan IPv4 dan IPv6, default route, connected route, metric, protocol,
  scope, table, device, dan route type tersedia ketika kernel menyediakannya.
- CPU dihitung dari counter `/proc/stat`; memori dari `/proc/meminfo`.
- Traffic Linux mengembalikan sampling RX/TX antar-request. Sampling pertama
  bernilai nol karena belum ada baseline; interval ditentukan oleh jarak waktu
  antar pengambilan.
- Semua pembacaan memiliki context/timeout dan batas output. Tidak ada shell,
  `sudo`, `ip add`, `ip route replace`, atau perubahan konfigurasi.

## Endpoint baru

- `GET /api/v1/ip/addresses`
- `GET /api/v1/ip/routes`

Keduanya dilindungi authentication middleware dan tercantum dalam OpenAPI.

## Perubahan frontend

- Menu IP → Addresses dan IP → Routes tidak lagi `Coming Soon`.
- Halaman address menyediakan filter interface, IPv4/IPv6, pencarian, refresh,
  detail, loading, error, dan empty state.
- Halaman route menyediakan filter table/interface/IPv4/IPv6, pencarian,
  refresh, status, loading, error, dan empty state.
- Dashboard tetap memakai endpoint agregat provider aktif; tidak ada angka
  simulasi ketika mode Linux dipilih.
- Login, session cookie, CSRF, protected route, dan logout v0.1.1 tidak diubah.

## Rancangan CRUD masa depan

v0.2 sengaja belum menyediakan tombol Add/Edit/Delete. Implementasi berikutnya
harus menggunakan model deklaratif dan fixed-operation helper terpisah dengan:

1. Validasi address/prefix, gateway, table, metric, dan interface.
2. Preview diff dan konfirmasi administrator.
3. Safe Apply dengan snapshot konfigurasi.
4. Health check koneksi management setelah apply.
5. Timeout dan automatic rollback bila health check gagal.
6. Persistence atomik setelah sukses dan pemulihan setelah reboot.
7. Audit log untuk setiap perubahan dan hasil rollback.

Tidak ada bagian rancangan ini yang aktif pada v0.2.

## Pengujian

Pengujian yang ditambahkan mencakup parsing route JSON IPv4/IPv6, default route,
table/metric, invalid JSON, parser CPU, endpoint authentication, Mock Provider,
serta test komponen Addresses dan Routes. Jalankan:

```bash
cd apps/api
go test ./...
go test -race ./...

cd ../webfig
npm test
npm run build

cd ../..
git diff --check
```

Provider runtime Debian 13 belum diuji karena task ini tidak mengizinkan
koneksi atau deployment ke router.

Hasil pada lingkungan development:

- `go test -p 1 ./...` — lulus.
- `go test -race -p 1 ./...` — lulus.
- `npm test -- --run` — 6 file, 8 test lulus.
- `npm run build` — lulus.
- `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go test -c ./internal/providers` — lulus.
- `git diff --check` — lulus.

Build cross-platform CGO penuh dari macOS tidak dijadikan syarat karena
memerlukan toolchain Linux; binary deployment tetap harus dibangun native pada
Debian dengan `CGO_ENABLED=1`.

## Batasan dan risiko

- Runtime Linux Provider belum diuji langsung pada Debian 13/kernel 6.12.
- Ketersediaan `/usr/sbin/ip`, format output iproute2, permission `/proc`/`/sys`,
  systemd, dan TC harus diverifikasi pada VM target.
- Status route berasal dari keberadaan entry pada dump kernel dan belum
  melakukan probe konektivitas.
- Traffic rate baru memiliki satu baseline per proses; belum persisten dan
  belum menjadi time-series durable.
- Sesi Accel-PPP nyata masih bergantung pada kontrak control socket yang belum
  divalidasi.
- Driver SQLite deployment tetap membutuhkan `CGO_ENABLED=1`.

## Panduan uji Debian 13 (manual, tanpa perubahan konfigurasi)

1. Siapkan snapshot/VM terisolasi dan user service non-root.
2. Verifikasi `command -v ip`, `/usr/sbin/systemctl`, `/usr/sbin/tc`, dan akses
   baca `/proc`/`/sys`.
3. Inisialisasi admin melalui CLI lokal, lalu jalankan API dengan
   `CODENET_PROVIDER=linux` dan bind loopback.
4. Login WebFig dan panggil endpoint IP dengan session administrator.
5. Bandingkan hasil dengan `ip -j address show`, `ip -j -4 route show table all`,
   `ip -j -6 route show table all`, `systemctl is-active`, dan statistik `/sys`.
6. Pastikan tidak ada perubahan pada nftables, Accel-PPP, PPPoE, route, IP,
   atau TC selama pengujian.

## Status kesiapan

**Siap untuk review dan uji VM terisolasi; belum production-ready.** Kriteria
runtime Linux dan kompatibilitas Debian baru dapat dinyatakan selesai setelah
pengujian langsung pada CODENET Router Debian 13, yang sengaja belum dilakukan.
