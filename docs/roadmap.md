# Roadmap CODENET

## v0.1 — Fondasi monitoring

- [x] Monorepo API dan WebFig.
- [x] Mock provider lintas platform.
- [x] Linux provider read-only dasar.
- [x] Dashboard, Interfaces, PPPoE, Bandwidth, System.
- [x] OpenAPI, unit test, build frontend.
- [ ] Validasi integrasi pada VM CODENET Router.
- [ ] Parser socket kontrol Accel-PPP.
- [ ] Sampling traffic time-series di backend.

## v0.1.1 — Security hardening

- [x] Login administrator melalui WebFig.
- [x] Argon2id + SQLite dengan database permission `0600`.
- [x] Session cookie HttpOnly, SameSite, expiry, logout, dan reset password CLI.
- [x] CSRF protection untuk method unsafe dan rate limiting login.
- [x] Linux provider timeout, bounded reads/output, fixed executable path, dan error handling.
- [x] Audit unit/backend/frontend.
- [ ] HTTPS/reverse proxy dan audit log security event pada deployment Debian.

## v0.2 — Linux integration dan persistence lanjutan

- Sampling traffic time-series di backend.
- Role-based access control awal.
- Hardening systemd dan packaging Debian.

## v0.3 — Manajemen terbatas

- CRUD pelanggan PPPoE melalui service tervalidasi.
- Disconnect session dengan policy dan audit.
- IP Pool dan PPPoE Profile.
- Transaction/rollback konfigurasi.
- Pengujian integrasi VM sebelum apply.

## Tahap berikutnya

- Address dan route management.
- Firewall policy dengan model deklaratif.
- Backup/restore konfigurasi.
- Installer ISO dan upgrade atomik.
- Observability, alerting, dan high availability.

Setiap fitur perubahan router harus melewati autentikasi, otorisasi, validasi, audit, preview perubahan, dan mekanisme rollback.
