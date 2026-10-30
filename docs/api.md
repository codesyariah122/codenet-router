# Management API v0.3

Base path: `/api/v1`. Kontrak mesin tersedia di `apps/api/openapi/openapi.yaml`.

## Format respons

Sukses:

```json
{"success":true,"message":"Informasi sistem berhasil diambil","data":{"hostname":"codenet-router"}}
```

Error:

```json
{"success":false,"message":"interface tidak ditemukan","error":{"code":"NOT_FOUND"}}
```

## Autentikasi

`GET /health` adalah liveness minimal dan tidak mengungkap informasi router. `POST /api/v1/auth/login` adalah satu-satunya endpoint domain publik. Login sukses menetapkan cookie `codenet_session` dengan atribut HttpOnly dan SameSite Strict, lalu mengembalikan CSRF token pada body JSON. Token tersebut hanya disimpan di memory aplikasi WebFig.

Semua endpoint lain membutuhkan cookie session aktif. Semua method selain GET/HEAD/OPTIONS di bawah `/api/v1/` juga membutuhkan header `X-CSRF-Token`. Logout menghapus session server-side dan cookie browser.

Endpoint admin lokal (bukan HTTP):

```bash
cd apps/api
go run ./cmd/codenet-admin init --username admin
go run ./cmd/codenet-admin reset-password --username admin
```

## Endpoint v0.3

| Method | Endpoint | Keterangan |
|---|---|---|
| GET | `/health` | Liveness API |
| POST | `/api/v1/auth/login` | Login administrator; publik dan rate-limited |
| GET | `/api/v1/auth/session` | Session administrator aktif |
| POST | `/api/v1/auth/logout` | Logout; membutuhkan CSRF |
| GET | `/api/v1/dashboard` | Agregat dashboard |
| GET | `/api/v1/system/info` | OS, hostname, kernel, uptime |
| GET | `/api/v1/system/resources` | CPU dan memori |
| GET | `/api/v1/system/services` | systemd, Accel-PPP, nftables |
| GET | `/api/v1/interfaces` | Semua interface |
| GET | `/api/v1/interfaces/{name}` | Detail interface |
| GET | `/api/v1/ip/addresses` | Alamat IPv4/IPv6 aktual dari interface kernel |
| GET | `/api/v1/ip/routes` | Routing table IPv4/IPv6 aktual dari kernel |
| GET | `/api/v1/pppoe/status` | Status access concentrator |
| GET | `/api/v1/pppoe/sessions` | Sesi pelanggan tanpa password; 503 jika control interface tidak tersedia |
| GET | `/api/v1/bandwidth/profiles` | Profil limiter read-only |
| GET | `/api/v1/bandwidth/statistics` | Statistik TC/shaper |

Semua endpoint router bersifat read-only. Operasi create/update/delete/disconnect belum tersedia. Kontrak lengkap berada di `apps/api/openapi/openapi.yaml`.

`/ip/addresses` mengembalikan `address`, `network`, `interface`, `ip_version`,
`scope`, dan `flags`. `/ip/routes` mengembalikan destination, gateway,
interface, metric, protocol, scope, routing table, versi IP, tipe route, dan
status. Kedua endpoint menggunakan provider terpilih dan tetap membutuhkan
session administrator.

`/pppoe/status` mengembalikan status service, status control interface,
access concentrator, interface server, kesiapan jumlah sesi, dan sumber IP
pool. Nilai pool/konsentrator tidak diisi sebagai konfigurasi hardcoded pada
Linux Provider. `/pppoe/sessions` mengembalikan counter kernel PPP, RX/TX,
upload/download rate, uptime, dan status interface. Error control interface
dikembalikan sebagai HTTP 503, bukan array kosong.
