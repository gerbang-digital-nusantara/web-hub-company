# web-hub-company

Situs perusahaan / landing page **HUB by Xpanca**: memperkenalkan HUB (penjualan, stok,
dan keuangan dalam satu alur), produk pendukung (HUB POS, HUB Commerce, Partner API),
paket harga, FAQ, dan form **Minta Demo**.

Polanya sama dengan `web-hub-v3`: server Go + Fiber kecil yang menyajikan folder `view/`
(HTML/CSS/JS statis, tanpa build front-end).

## Menjalankan lokal

```bash
cp .env.example .env     # opsional, semua punya default
go run .
# buka http://localhost:3076
```

Docker lokal: aktifkan `build: .` di `docker-compose.yml`, lalu `docker compose up -d --build`.

## Deploy ke VPS (pola sama dengan web-hub-v2)

Push ke branch **`dev`** menjalankan `.github/workflows/dev.yml`: build image
`ghcr.io/gerbang-digital-nusantara/web-hub-company:latest`, salin `docker-compose.yml`
ke `/home/app/web-hub-company/` di VPS, tulis `.env` dari secret **`ENV`**, lalu
`docker compose pull && up -d` (port 3076, data di `./data`).

Secret repo yang dibutuhkan (nama sama dengan web-hub-v2): `PACKAGE_USERNAME`,
`PACKAGE_TOKEN`, `VPS_IP`, `VPS_USER`, `VPS_PASSWORD`, `ENV`. Isi `ENV` =
isi `.env` (lihat `.env.example`) — termasuk `HUBCTL_API_KEY`/`HUBCTL_API_SECRET`
supaya form demo benar-benar diteruskan ke hub-control, dan `PROXY_HEADER=X-Forwarded-For`
bila di belakang Nginx.

## Pratinjau di GitHub Pages

Push ke `main` menjalankan `.github/workflows/pages.yml`, yang mengunggah folder
`view/` (statis saja) ke GitHub Pages:
`https://gerbang-digital-nusantara.github.io/web-hub-company/`.

- Sekali saja: **Settings → Pages → Source = GitHub Actions**.
- Server Go tidak ikut, jadi di `*.github.io` mode **pratinjau** menyala otomatis
  (`HUB_PREVIEW` di `config.js`): ada spanduk kuning, form Demo & Daftar bisa dicoba
  sampai layar "diterima" tetapi **tidak dikirim/disimpan**.
- Karena itu semua path di HTML/JS **relatif** (`assets/…`, `daftar.html`,
  `api/…`), bukan `/assets/…` — situs Pages berada di sub-path `/web-hub-company/`.
  `404.html` memakai `<base>` yang dihitung otomatis.

## Struktur

```
main.go                     app Fiber: static view/, /healthz, rute API, cache header
demo.go                     POST /api/demo   — form Coba Gratis, GET /api/subdomain
hubcontrol.go               klien Partner API hub-control (HMAC) + aturan format subdomain
daftar.go                   POST /api/daftar — form Daftar Berlangganan
paket.go                    katalog paket dari hub-control (GET /api/paket, validasi + estimasi), cadangan paket.json
view/index.html             beranda: Hero, Cara Kerja, Modul, Produk, Keunggulan, Harga, FAQ, Demo
view/daftar.html            halaman Daftar Berlangganan (juga /daftar?paket=bisnis)
view/404.html
view/assets/data/paket.json CADANGAN katalog (hub-control mati / pratinjau Pages)
view/assets/css/style.css   warna dari logo HUB (navy #1F3A68, biru #2B8BC0, oranye #E8541C)
view/assets/js/config.js    nomor WhatsApp & email — ISI DI SINI
view/assets/js/paket.js     muat /api/paket (pratinjau: paket.json), hitung tagihan, render kartu Harga
view/assets/js/main.js      menu ponsel, kontak, animasi, form demo
view/assets/js/daftar.js    halaman Daftar: pilih paket/periode, ringkasan, kirim
data/demo-requests.jsonl    permintaan demo yang masuk (tidak di-commit)
data/pendaftaran.jsonl      pendaftaran berlangganan yang masuk (tidak di-commit)
```

## Sebelum ditayangkan

1. **Kontak** — isi `whatsapp` dan `email` di `view/assets/js/config.js`. Selama kosong,
   tombol WhatsApp dan tautan email disembunyikan.
2. **Harga** — diatur admin di hub-control menu **Paket & Harga** (lihat *Katalog paket*
   di bawah), bukan di situs ini. `view/assets/data/paket.json` hanya cadangan; samakan
   dengan panel bila harga berubah.
3. Setiap mengubah CSS/JS, naikkan `?v=` di tag `<link>`/`<script>` pada `index.html`
   dan `daftar.html` (aset di-cache browser 1 hari).

## Katalog paket (hub-control)

Paket & harga dibaca dari `GET {HUBCTL_BASE_URL}/api/public/pricing` (tanpa API Key;
hanya paket berstatus Aktif, sudah terurut). Server menyimpannya 5 menit lalu
menyajikannya di `GET /api/paket` — dipakai kartu Harga, halaman Daftar, **dan** validasi
+ estimasi tagihan di `POST /api/daftar`, jadi ketiganya selalu sama dengan panel.

- Pemetaan: `code`→`id`, `name`→`nama`, `tagline`→`untuk`, `price_monthly`/`price_yearly`
  →`harga_bulanan`/`harga_tahunan` (0 = tidak dijual per periode itu; pilihan periodenya
  dimatikan), `recommended`→lencana Rekomendasi, `cta_type=contact`→paket penawaran.
- hub-control gagal → salinan terakhir yang berhasil; belum pernah berhasil sejak server
  jalan → `paket.json`. Respons `/api/paket` membawa `sumber` (`hub-control`/`lokal`).
- Tautan lama `daftar.html?paket=bisnis` tetap memilih Bisnis (code-nya `business`):
  paket dicari per `id`, lalu per nama.
- `/api/public/pricing` dibatasi 20/menit/IP dan CORS-nya hanya untuk `xpanca.id`, karena
  itu dibaca lewat server, bukan langsung dari browser.

## Form Daftar Berlangganan

`POST /api/daftar`. Wajib: `paket` (code paket di katalog), `periode` (`bulanan`/`tahunan`,
kecuali paket penawaran), `nama_pic`, `nama_usaha`, `phone`, `email`, `setuju`.
Opsional: usulan subdomain (`<slug>.hub.xpanca.com`, pola SaaS branch `tenant` web-hub-v3).
Isian data disamakan dengan form Coba Gratis (nama, nama usaha, WhatsApp, email, alamat
aplikasi) — kolom lama jenis usaha, gudang/pengguna, kota, alamat, NPWP, jabatan, catatan
dihapus sejak 2026-10-26.

Hasilnya nomor pendaftaran `HUB-YYMMDD-XXXX` + satu baris di `data/pendaftaran.jsonl`
(dengan estimasi tagihan saat mendaftar, status `baru`). Bila `HUBCTL_*` diisi, pendaftar
**ikut didaftarkan ke hub-control** (lihat di bawah) — semua calon klien masuk satu
antrean di menu Tenant. Pemetaan: `nama_usaha`→`business_name`, `nama_pic`→`full_name`,
`phone`, `email`, `subdomain`, plus paket produk: `intent=subscribe`, `package` = nama
paket (mis. `Bisnis`), `period` = `bulanan`/`tahunan` (kosong untuk paket penawaran).
Form Coba Gratis mengirim `intent=trial` + `package` pilihan pendaftar (kosong bila "Belum tahu").

## Form Coba Gratis (beranda, `#demo`)

Dulu bernama "Minta Demo"; endpoint, anchor `#demo`, dan `demo-requests.jsonl` tetap.
`POST /api/demo` (JSON: `name`, `company`, `phone`, `email`, `subdomain`, `package`; `message` masih diterima tapi kolomnya sudah tidak ada di form).
Dibatasi 5 kiriman/menit/IP, ada honeypot `website` untuk bot. Tiap permintaan ditulis
satu baris JSON ke `$DATA_DIR/demo-requests.jsonl` (`tail -f data/demo-requests.jsonl`).

## Integrasi hub-control (Partner API, partner PTR00001 "Web xpanca.id")

Aktif bila `HUBCTL_API_KEY` + `HUBCTL_API_SECRET` diisi di `.env` (`hubcontrol.go`),
untuk form Coba Gratis **dan** Daftar:

- Pendaftar didaftarkan sebagai tenant lewat `POST /api/public/register` (HMAC-SHA256)
  → `{subdomain}.hub.xpanca.com` (kosong = dibuat dari nama usaha). Akun aktif setelah
  admin hub-control **Approve**.
- IP pengunjung diteruskan sebagai header `X-Forwarded-For` supaya Audit Trail
  hub-control mencatat IP asli — di VPS wajib `PROXY_HEADER=X-Forwarded-For`, kalau
  tidak yang terkirim IP Nginx.
- Coba Gratis wajib `name`, `company`, `email`. Penolakan 400 dari hub-control
  (email/phone/subdomain sudah dipakai) dikembalikan ke form sebagai 422 di kolom
  terkait, tidak disimpan. Gangguan lain (401/5xx/jaringan) tetap dicatat ke jsonl
  (`hubctl_http`, `hubctl_message`) dan dijawab biasa — tim menindaklanjuti.
- Subdomain: situs hanya memeriksa **format** (3–30 karakter a-z/0-9/minus, tidak
  diawali/diakhiri minus, tanpa `--`; `subdomainError()` + `HubSub` di `main.js`).
  Ketersediaan & nama terlarang **acuannya hub-control**: `GET /api/subdomain?name=…`
  (30/menit/IP; alias lama `/api/demo/subdomain`) memproksi `check-subdomain`, dan
  kedua form memakainya sambil mengetik (input `data-subcheck`). Bukan reservasi.
- Belum ada langkah **OTP**. Aman selama `REQUIRE_OTP=false` di hub-control; bila OTP
  dinyalakan, form perlu isian kode OTP dulu.
- Secret **hanya** di `.env` server. Jangan pakai iframe `embed/register` hub-control di
  sini: iframe itu hanya mau tampil di domain terdaftar partner (`xpanca.id`).

Tanpa `HUBCTL_*` kedua form kembali ke perilaku lama (hanya dicatat ke jsonl; Coba Gratis
cukup `name` + salah satu dari `phone`/`email`).

Respons memakai amplop yang sama dengan hub-api-v2: `{code, status, message, errors}`.
