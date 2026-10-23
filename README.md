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
# buka http://localhost:3110
```

Docker:

```bash
docker compose up -d --build
```

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
demo.go                     POST /api/demo   — form Minta Demo (+ cek subdomain)
hubcontrol.go               klien Partner API hub-control (HMAC) untuk form demo
daftar.go                   POST /api/daftar — form Daftar Berlangganan
paket.go                    baca paket.json (validasi paket + estimasi tagihan)
view/index.html             beranda: Hero, Cara Kerja, Modul, Produk, Keunggulan, Harga, FAQ, Demo
view/daftar.html            halaman Daftar Berlangganan (juga /daftar?paket=bisnis)
view/404.html
view/assets/data/paket.json SATU-SATUNYA sumber paket & harga — ISI DI SINI
view/assets/css/style.css   warna dari logo HUB (navy #1F3A68, biru #2B8BC0, oranye #E8541C)
view/assets/js/config.js    nomor WhatsApp & email — ISI DI SINI
view/assets/js/paket.js     muat paket.json, hitung tagihan, render kartu Harga
view/assets/js/main.js      menu ponsel, kontak, animasi, form demo
view/assets/js/daftar.js    halaman Daftar: pilih paket/periode, ringkasan, kirim
data/demo-requests.jsonl    permintaan demo yang masuk (tidak di-commit)
data/pendaftaran.jsonl      pendaftaran berlangganan yang masuk (tidak di-commit)
```

## Sebelum ditayangkan

1. **Kontak** — isi `whatsapp` dan `email` di `view/assets/js/config.js`. Selama kosong,
   tombol WhatsApp dan tautan email disembunyikan.
2. **Harga** — angka di `view/assets/data/paket.json` masih **contoh**
   (Starter Rp 299rb, Bisnis Rp 799rb, batas gudang/pengguna, diskon tahunan 2 bulan).
   Berkas ini dibaca kartu Harga di beranda, halaman Daftar, *dan* server (estimasi
   tagihan yang disimpan), jadi cukup diubah di satu tempat — tanpa restart.
3. Setiap mengubah CSS/JS, naikkan `?v=` di tag `<link>`/`<script>` pada `index.html`
   dan `daftar.html` (aset di-cache browser 1 hari).

## Form Daftar Berlangganan

`POST /api/daftar`. Wajib: `paket` (id di paket.json), `periode` (`bulanan`/`tahunan`,
kecuali paket penawaran), `nama_usaha`, `kota`, `nama_pic`, `phone`, `email`, `setuju`.
Opsional: jenis usaha, jumlah gudang/pengguna, alamat, NPWP (15/16 digit), usulan
subdomain (`<slug>.hub.xpanca.com`, pola SaaS branch `tenant` web-hub-v3), catatan.

Hasilnya nomor pendaftaran `HUB-YYMMDD-XXXX` + satu baris di `data/pendaftaran.jsonl`
(dengan estimasi tagihan saat mendaftar, status `baru`). **Belum membuat tenant**:
DB, realm SSO, dan subdomain masih disiapkan tim secara manual.

## Form Minta Demo

`POST /api/demo` (JSON: `name`, `company`, `phone`, `email`, `subdomain`, `package`, `message`).
Dibatasi 5 kiriman/menit/IP, ada honeypot `website` untuk bot. Tiap permintaan ditulis
satu baris JSON ke `$DATA_DIR/demo-requests.jsonl` (`tail -f data/demo-requests.jsonl`).

**Diteruskan ke hub-control (Partner API, partner PTR00001 "Web xpanca.id")** bila
`HUBCTL_API_KEY` + `HUBCTL_API_SECRET` diisi di `.env` (`hubcontrol.go`):

- Pendaftar didaftarkan sebagai tenant coba gratis lewat `POST /api/public/register`
  (HMAC-SHA256; `company`→`business_name`, `name`→`full_name`, `phone`, `email`,
  `subdomain` → `{subdomain}.hub.xpanca.com`, kosong = dibuat dari nama usaha).
  Akun aktif setelah admin hub-control **Approve**.
- Wajib: `name`, `company`, `email`. Penolakan 400 dari hub-control (email/subdomain
  sudah dipakai) dikembalikan ke form sebagai 422 di kolom terkait, tidak disimpan.
  Gangguan lain (401/5xx/jaringan) tetap dicatat ke jsonl (`hubctl_http`,
  `hubctl_message`) dan dijawab seperti permintaan demo biasa — tim menindaklanjuti.
- `GET /api/demo/subdomain?name=…` (30/menit/IP) = cek ketersediaan alamat sambil
  mengetik; proksi ke `GET /api/public/check-subdomain` (path ditandatangani tanpa
  query, body kosong). Bukan reservasi.
- Secret **hanya** di `.env` server. Jangan pakai iframe `embed/register` hub-control di
  sini: iframe itu hanya mau tampil di domain terdaftar partner (`xpanca.id`).

Tanpa `HUBCTL_*` form kembali ke perilaku lama: wajib `name` + salah satu dari
`phone`/`email`, hanya dicatat ke jsonl.

Respons memakai amplop yang sama dengan hub-api-v2: `{code, status, message, errors}`.
