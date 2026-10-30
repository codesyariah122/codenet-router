# Laporan Audit Security Hardening CODENET WebFig v0.1.1

Tanggal audit: 30 Oktober 2026  
Branch: `security/webfig-auth-hardening-v0.1.1`  
Scope: repository lokal CODENET Router OS; tidak ada koneksi atau perubahan ke Debian Router nyata.

## Ringkasan eksekutif

Hardening v0.1.1 menutup temuan utama pada fondasi v0.1: seluruh endpoint monitoring sebelumnya publik. Sekarang API fail-closed bila administrator belum diinisialisasi, memakai Argon2id, session server-side SQLite, cookie HttpOnly/SameSite, expiry, logout, CSRF, dan rate limiting. Linux provider tetap read-only dan tidak menerima command dari parameter HTTP.

Milestone dapat dianggap **siap untuk review integrasi**, bukan siap deployment produksi tanpa verifikasi HTTPS, privilege Debian, dan Linux provider terhadap VM CODENET.

## Temuan awal dan risiko

| ID | Temuan | Risiko awal | Perbaikan |
|---|---|---|---|
| SEC-001 | Endpoint monitoring tidak membutuhkan autentikasi | Kritis | Semua `/api/v1/*` kecuali login dilindungi middleware session; `/health` minimal dan publik |
| SEC-002 | Belum ada mekanisme admin/password recovery | Tinggi | CLI lokal `codenet-admin init` dan `reset-password`; tidak ada signup HTTP |
| SEC-003 | Belum ada session/CSRF | Tinggi | Token session acak, hash token di SQLite, HttpOnly, SameSite Strict, expiry, CSRF unsafe methods |
| SEC-004 | Percobaan login tidak dibatasi | Tinggi | Rate limit per username dan IP, generic error, Retry-After |
| SEC-005 | Linux command/output/file handling belum dibatasi seragam | Sedang-Tinggi | Binary path absolut, context timeout, fixed args, bounded output/read, validasi interface, error propagation |
| SEC-006 | Service unit belum eksplisit membatasi privilege/state | Sedang | User khusus, capability kosong, NoNewPrivileges, ProtectSystem, PrivateDevices, StateDirectory 0700 |

## Perubahan implementasi

### Backend

- `internal/auth`: Argon2id PHC hashing, validasi password, SQLite migration, admin init/reset, session token hash, expiry, logout, dan limiter.
- `internal/middleware`: auth cookie, context principal, CSRF untuk method unsafe, timeout request, CORS credentials/allowlist, no-store API response, security headers.
- `internal/handlers`: login/session/logout, JSON strict decoding, Content-Type validation, generic errors, auth route separation.
- `cmd/codenet-admin`: CLI lokal tanpa password di argumen atau log; reset membatalkan session lama.
- `internal/providers/linux`: tidak membaca `chap-secrets`, tidak mengubah jaringan, tidak menjalankan shell/sudo, memakai `/usr/bin/systemctl` dan `/usr/sbin/tc`, timeout, bounded reads/output, dan error handling.

### Frontend

- `AuthProvider` melakukan session check melalui cookie browser.
- `ProtectedRoute` mencegah akses dashboard tanpa session.
- `LoginPage` dan logout tersedia.
- CSRF token hanya disimpan di memory React; tidak ada localStorage/sessionStorage untuk token.
- 401 mengosongkan state/query cache dan mengembalikan user ke login.

### Repository dan deployment

`.gitignore` mencakup dependency, dist/coverage, SQLite files, environment files, TypeScript build info, binary, dan data directory. Unit systemd contoh tidak mengaktifkan service atau mengubah router.

## Pengujian yang dijalankan

Backend:

- `gofmt -w .`
- `GOMODCACHE=/private/tmp/codenet-gomodcache GOCACHE=/private/tmp/codenet-go-cache go test -p 1 ./...` — lulus.
- `GOMODCACHE=/private/tmp/codenet-gomodcache GOCACHE=/private/tmp/codenet-go-cache go test -race -p 1 ./...` — lulus.
- Test mencakup Argon2id, admin single-init, reset password, session expiry, login sukses/gagal, cookie flags, endpoint unauthorized, logout, CSRF, rate limit, input validation, provider error sanitization, mock provider, parsing Linux data, bounded read, bounded command output.
- Kompilasi provider Linux untuk `GOOS=linux GOARCH=amd64` berhasil diverifikasi.
  Binary deployment wajib dibangun native pada Debian dengan `CGO_ENABLED=1`
  karena driver SQLite menggunakan CGO; build `CGO_ENABLED=0` tidak dianggap
  valid untuk runtime database meskipun dapat lolos tahap kompilasi. Direct
  provider runtime belum diuji pada router nyata.

Frontend:

- `npm test` — 4 test file, 6 test lulus.
- `npm run build` — lulus.
- Test mencakup LoginPage success/error, Dashboard API rendering, TrafficChart, dan format utility.

Verifikasi browser lokal juga lulus: login menampilkan dashboard dengan session
cookie, logout mengembalikan pengguna ke `/login`, dan token tidak disimpan di
localStorage/sessionStorage. CORS allowlist diuji dengan origin yang salah dan
request ditolak oleh browser; origin lokal yang dikonfigurasi kemudian berhasil.

## Kriteria keberhasilan v0.1.1

1. Login administrator melalui WebFig — terpenuhi.
2. Password Argon2id — terpenuhi.
3. Endpoint monitoring tanpa login ditolak — terpenuhi.
4. Session expiry dan logout — terpenuhi.
5. CSRF pada method unsafe — terpenuhi.
6. Rate limiting login — terpenuhi.
7. Mock Provider tetap berfungsi — terpenuhi.
8. Linux Provider read-only — terpenuhi melalui inspeksi kode, unit test, dan compile provider; runtime Debian belum diuji.
9. Test backend/frontend — terpenuhi untuk test yang tersedia di lingkungan development.
10. Tidak ada perubahan router nyata — terpenuhi; tidak ada deployment atau koneksi ke router.

## Risiko yang belum terselesaikan

1. Linux provider runtime belum diuji langsung pada Debian 13 CODENET; hanya helper/provider compile dan unit test lintas platform yang tersedia.
2. Kontrak control socket/CLI Accel-PPP belum divalidasi, sehingga sesi PPPoE Linux nyata masih kosong secara eksplisit.
3. API belum menyediakan HTTPS sendiri. Reverse proxy HTTPS dan `CODENET_COOKIE_SECURE=true` wajib digunakan sebelum akses jaringan manajemen yang tidak sepenuhnya tepercaya.
4. Rate limiter masih in-memory; restart service mereset counter dan beberapa instance memerlukan limiter terpusat.
5. Belum ada audit log khusus security event yang persisten. Structured request log tidak mencatat password, token, atau recovery secret.
6. SQLite tidak dienkripsi at-rest; perlindungan bergantung pada permission `0600`, service user, filesystem, dan host hardening.
7. Token CSRF disimpan di database sebagai nilai non-authentication secret. Session token tetap hanya disimpan sebagai hash; database permission wajib dipertahankan.

## Kesiapan integrasi Debian 13

Status: **siap untuk tahap review dan uji VM terisolasi, belum siap dinyatakan production-ready**.

Sebelum integrasi nyata, administrator harus:

1. Membuat akun service non-root dan state directory 0700.
2. Menjalankan `codenet-admin init` secara lokal dengan password baru yang tidak pernah masuk repository/log.
3. Mengaktifkan HTTPS/reverse proxy dan `CODENET_COOKIE_SECURE=true`.
4. Memverifikasi path `/usr/bin/systemctl` dan `/usr/sbin/tc` pada Debian target.
5. Menguji read-only access, timeout, dan kegagalan provider tanpa mengubah nftables, Accel-PPP, PPPoE, routing, atau TC.
6. Menguji rollback unit/service secara manual; repository tidak melakukan deployment otomatis.

Tidak ada pengujian atau perubahan pada Debian Router nyata dalam audit ini.
