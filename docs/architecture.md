# Arsitektur CODENET v0.1.1

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
- `middleware`: structured request logging, recovery, security header, pembatasan CORS, request timeout, autentikasi cookie, CSRF, dan batas request.

Linux provider tidak menggunakan shell (`sh -c`) dan tidak memanggil `sudo`. Pemanggilan binary absolut `/usr/bin/systemctl` dan `/usr/sbin/tc` memakai daftar argumen tetap dengan timeout. Nama interface untuk `tc` hanya berasal dari inventaris interface kernel, bukan input HTTP. Output command dibatasi dan pembacaan file kernel memiliki batas ukuran/context check.

## Frontend

WebFig menggunakan route terpisah untuk setiap domain. TanStack Query mengelola cache, loading, error, retry, dan refresh periodik. Service API menjadi satu-satunya lokasi akses HTTP. Komponen presentasional tidak mengetahui URL backend.

## Autentikasi administrator

Alur autentikasi v0.1.1:

1. CLI lokal `codenet-admin init` membuat satu akun administrator pertama; tidak ada pendaftaran HTTP.
2. Password disimpan sebagai Argon2id PHC string di SQLite dengan permission database `0600`.
3. Login menghasilkan token session acak; hanya SHA-256 token yang disimpan di SQLite.
4. Browser menerima cookie `HttpOnly`, `SameSite=Strict`, dan `Secure` berdasarkan konfigurasi HTTPS.
5. Session memiliki expiry; reset password membatalkan semua session.
6. CSRF token disimpan per-session dan diwajibkan pada seluruh method unsafe di subtree API terlindungi.
7. Rate limiter in-memory membatasi percobaan berdasarkan user dan IP.
8. CLI reset password membaca password dari terminal tanpa echo atau dari dua baris stdin, tanpa mencetak secret.

`/health` hanya mengembalikan status minimal. Semua route `/api/v1/*` lain, termasuk session dan monitoring, membutuhkan autentikasi.

## Batas trust dan privilege

API tidak mengubah nftables, routing, interface, TC, Accel-PPP, atau systemd. Unit contoh menjalankan service sebagai `codenet-api`, tanpa capability Linux, dengan `ProtectSystem=strict`, `ProtectHome=true`, `PrivateDevices=true`, dan state directory khusus. Helper privilege versi mendatang harus terpisah, fixed-operation, dan tidak dipanggil langsung dari parameter HTTP.
