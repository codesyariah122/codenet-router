# Management API

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

## Endpoint v0.1

| Method | Endpoint | Keterangan |
|---|---|---|
| GET | `/health` | Liveness API |
| GET | `/api/v1/dashboard` | Agregat dashboard |
| GET | `/api/v1/system/info` | OS, hostname, kernel, uptime |
| GET | `/api/v1/system/resources` | CPU dan memori |
| GET | `/api/v1/system/services` | systemd, Accel-PPP, nftables |
| GET | `/api/v1/interfaces` | Semua interface |
| GET | `/api/v1/interfaces/{name}` | Detail interface |
| GET | `/api/v1/pppoe/status` | Status access concentrator |
| GET | `/api/v1/pppoe/sessions` | Sesi pelanggan tanpa password |
| GET | `/api/v1/bandwidth/profiles` | Profil limiter read-only |
| GET | `/api/v1/bandwidth/statistics` | Statistik TC/shaper |

Semua endpoint domain bersifat read-only. Operasi create/update/delete/disconnect tidak tersedia pada v0.1.
