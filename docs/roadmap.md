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

## v0.2 — Security dan persistence

- Session authentication + HttpOnly Cookie.
- CSRF protection, rate limiting, dan audit log.
- SQLite untuk user admin, session, dan konfigurasi aplikasi.
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
