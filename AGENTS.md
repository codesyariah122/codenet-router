# Panduan Kontributor dan Agen CODENET

## Ruang lingkup

Repository ini adalah fondasi CODENET Router OS. Semua perubahan harus mempertahankan prinsip **read-only by default**, pemisahan antara HTTP dan integrasi Linux, serta kemampuan development tanpa akses router.

## Aturan wajib

1. Gunakan bahasa Inggris untuk nama fungsi, tipe, variabel, endpoint, dan konvensi kode.
2. Gunakan bahasa Indonesia untuk dokumentasi, komentar yang menjelaskan keputusan, UI, dan laporan perubahan.
3. Jangan membaca, mencatat, atau mengirim isi `/etc/codenet/chap-secrets`.
4. Jangan menerima command shell atau argumen command bebas dari HTTP request.
5. Jangan mengubah nftables, interface, routing, TC, Accel-PPP, atau unit systemd pada v0.1.
6. Jangan menyimpan kredensial nyata, token, atau password di repository.
7. Handler hanya berbicara dengan service/provider; akses `/proc`, `/sys`, dan command Linux berada di provider.
8. Semua endpoint baru harus memiliki pengujian untuk respons sukses dan error relevan.
9. WebFig tidak boleh menampilkan fitur belum tersedia seolah-olah berfungsi; gunakan label `Coming Soon`.
10. Jangan melakukan deployment otomatis ke router.

## Pemeriksaan sebelum selesai

```bash
gofmt -w apps/api
cd apps/api && go test ./...
cd apps/webfig && npm test && npm run build
```

Perubahan pada kontrak API juga wajib memperbarui `apps/api/openapi/openapi.yaml` dan `docs/api.md`.
