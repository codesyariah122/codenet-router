# Arsitektur CODENET v0.2

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

Linux provider tidak menggunakan shell (`sh -c`) dan tidak memanggil `sudo`. Pemanggilan binary absolut `/usr/bin/systemctl`, `/usr/sbin/tc`, dan path iproute2 yang telah ditentukan memakai daftar argumen tetap dengan timeout. Nama interface untuk `tc` hanya berasal dari inventaris interface kernel, bukan input HTTP. Output command dibatasi dan pembacaan file kernel memiliki batas ukuran/context check.

`IPAddresses` menggunakan `net.Interfaces` dan `Interface.Addrs`, sehingga IPv4
dan IPv6 dibaca langsung dari kernel beserta prefix, network, interface, scope,
dan flag link. `Routes` memanggil `ip -j -4/-6 route show table all` dengan
binary tetap, lalu mem-parsing destination, gateway, device, metric, protocol,
scope, table, dan route type. Tidak ada operasi `ip add`, `ip route replace`,
atau perintah konfigurasi lain.

## Frontend

WebFig menggunakan route terpisah untuk setiap domain. TanStack Query mengelola cache, loading, error, retry, dan refresh periodik. Service API menjadi satu-satunya lokasi akses HTTP. Komponen presentasional tidak mengetahui URL backend. Modul IP Addresses dan IP Routes menyediakan filter interface/versi, pencarian, refresh, detail, dan empty state.

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

## Rancangan CRUD IP/route berikutnya

v0.2 hanya read-only. Add/edit/remove address dan route harus memakai model
deklaratif yang divalidasi sebelum apply, menghasilkan preview/diff, meminta
CSRF dan otorisasi tambahan, mencatat audit event, serta memakai helper
terpisah dengan operasi fixed-command. Safe Apply harus menyimpan snapshot,
menguji konektivitas management, memiliki timeout dan automatic rollback bila
health check gagal, lalu melakukan persistence secara atomik setelah sukses.
