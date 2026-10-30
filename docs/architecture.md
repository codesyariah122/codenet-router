# Arsitektur CODENET v0.1

## Tujuan

Arsitektur memisahkan penyajian HTTP dari detail Linux agar aplikasi dapat diuji di macOS/CI dan dipindahkan ke router tanpa menulis ulang handler.

```text
┌─────────────────────┐   HTTP /api/v1   ┌──────────────────────┐
│ CODENET WebFig      │ ───────────────▶ │ Management API       │
│ React + Query       │                  │ Handler + Middleware │
└─────────────────────┘                  └──────────┬───────────┘
                                                  │
                                           RouterService
                                                  │
                                      RouterProvider interface
                                         ┌────────┴────────┐
                                         │                 │
                                  MockProvider       LinuxProvider
                                  data deterministik /proc, /sys,
                                                    net, systemctl, tc
```

## Lapisan backend

- `models`: kontrak data internal dan JSON. Model sesi PPPoE sengaja tidak memiliki field password.
- `providers`: abstraksi pembacaan router. `MockProvider` menyediakan data lokal; `LinuxProvider` menyediakan pembacaan terbatas.
- `services`: orkestrasi use case, misalnya agregasi dashboard.
- `handlers`: transport HTTP, validasi input, dan envelope respons.
- `middleware`: structured request logging, recovery, security header, pembatasan CORS, dan batas request.

Linux provider tidak menggunakan shell (`sh -c`). Pemanggilan `systemctl` dan `tc` memakai daftar argumen tetap dengan timeout. Nama interface untuk `tc` hanya berasal dari inventaris interface kernel, bukan input HTTP.

## Frontend

WebFig menggunakan route terpisah untuk setiap domain. TanStack Query mengelola cache, loading, error, retry, dan refresh periodik. Service API menjadi satu-satunya lokasi akses HTTP. Komponen presentasional tidak mengetahui URL backend.

## Autentikasi berikutnya

Struktur v0.1 belum mengaktifkan login. Target desain:

1. Kredensial administrator tersimpan sebagai hash kuat, bukan hardcoded.
2. Login menghasilkan session server-side dengan expiry.
3. Browser menerima cookie `HttpOnly`, `Secure`, dan `SameSite=Strict`.
4. Endpoint router berada di belakang middleware autentikasi dan CSRF protection.
5. Audit event dicatat tanpa data rahasia.

Sebelum tahap tersebut selesai, API harus tetap bind ke loopback secara default.
