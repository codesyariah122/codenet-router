# CODENET WebFig v0.4 — PPPoE User Management

Branch: `feature/webfig-pppoe-user-management-v0.4`

## Implementasi

- Parser dan validator format lima kolom `chap-secrets`.
- Validasi username, IP pool, server, rate `download/upload`, password, dan
  path.
- Atomic write dengan permission preservation, backup `.bak`, rollback dasar,
  lock file, dan audit log tanpa password.
- Disable memakai marker komentar `# codenet-disabled` sehingga riwayat akun
  tetap ada dan Accel-PPP tidak membaca akun nonaktif. Sesi online tidak
  diputus otomatis.
- Helper terpisah `codenet-pppoe-helper` dengan Unix socket dan HMAC IPC.
- API default fail-closed (`503`) ketika helper tidak tersedia.
- Endpoint user terproteksi session administrator dan CSRF untuk operasi tulis.
- UI PPPoE Users dengan search, filter, pagination-ready table, create,
  enable/disable, delete confirmation, loading, error, dan empty state.
- Session status dipetakan dari monitoring Accel-PPP v0.3.

## Keamanan dan batasan

Management API tidak membaca atau menulis `/etc/codenet/chap-secrets` secara
langsung pada mode produksi. Helper hanya mengizinkan path tersebut, tetapi
belum melakukan reload atau restart `codenet-pppoe`; perubahan berlaku sesuai
perilaku Accel-PPP yang harus diverifikasi manual pada Debian 13. Password
tidak pernah dikembalikan melalui HTTP atau audit log.

Fixture lokal dapat digunakan dengan `CODENET_PPPOE_USERS_FIXTURE`, sedangkan
produksi memakai `CODENET_PPPOE_HELPER_SOCKET` dan
`CODENET_PPPOE_HELPER_SECRET_FILE`.
Untuk integration test Debian, gunakan unit terpisah
`configs/systemd/codenet-pppoe-helper-fixture.service`; helper fixture hanya
menerima path di bawah `/var/lib/codenet-v04/helper-test/` dan menolak root.

## Pengujian

- Unit parser, validator, atomic write, backup, disable marker, rollback path,
  path traversal, malformed rate, password preservation, dan concurrent lock
  ditambahkan pada `internal/pppoeusers`.
- `npm test -- --run` dan `npm run build` lulus.
- Pengujian package Go user/helper/provider lulus.
- Full `go test ./...` dan race test penuh perlu dijalankan pada environment
  dengan dependency SQLite/crypto tersedia; dependency tersebut tidak tersedia
  di sandbox ini.

## Verifikasi aman Debian 13

1. Salin binary helper dan API setelah review, tanpa deployment otomatis.
2. Buat secret IPC owner-only mode `0600`; jalankan helper sebagai user
   terpisah yang memiliki akses terbatas ke file target.
3. Jalankan helper pada socket Unix dengan permission terbatas.
4. Jalankan API tanpa root dan set environment helper.
5. Uji pada snapshot/fixture salinan, verifikasi backup dan permission.
6. Setelah review, lakukan satu perubahan akun non-kritis dan verifikasi login
   baru Accel-PPP secara manual; jangan disconnect sesi aktif dan jangan
   mengubah nftables, routing, TC, atau service secara otomatis.

Milestone belum production-ready sebelum helper, reload behavior, dan satu
perubahan akun diverifikasi langsung pada Debian CODENET Router.
