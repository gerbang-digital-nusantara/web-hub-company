package main

import (
	"crypto/rand"
	"encoding/json"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

/* ===================== Pendaftaran Berlangganan ===================== */

// daftarRequest = isian halaman Daftar (view/daftar.html).
//
// Dicatat ke DATA_DIR/pendaftaran.jsonl, dan — bila HUBCTL_* diisi — ikut
// didaftarkan ke hub-control seperti form Coba Gratis, supaya semua calon
// klien masuk satu antrean di menu Tenant dan admin cukup Approve (paket &
// periode belum bisa dikirim lewat Partner API; tercatat di jsonl dan perlu
// dukungan dari sisi hub-control). `Website` = honeypot, sama seperti form demo.
type daftarRequest struct {
	Paket   string `json:"paket"`
	Periode string `json:"periode"` // "bulanan" | "tahunan"; diabaikan untuk paket penawaran

	NamaUsaha      string `json:"nama_usaha"`
	JenisUsaha     string `json:"jenis_usaha"`
	JumlahGudang   int    `json:"jumlah_gudang"`
	JumlahPengguna int    `json:"jumlah_pengguna"`
	Kota           string `json:"kota"`
	Alamat         string `json:"alamat"`
	NPWP           string `json:"npwp"`
	Subdomain      string `json:"subdomain"` // usulan alamat aplikasi: <subdomain>.hub.xpanca.com (pola SaaS branch tenant web-hub-v3)

	NamaPIC string `json:"nama_pic"`
	Jabatan string `json:"jabatan"`
	Phone   string `json:"phone"`
	Email   string `json:"email"`

	Catatan string `json:"catatan"`
	Setuju  bool   `json:"setuju"`
	Website string `json:"website,omitempty"`
}

type daftarRecord struct {
	Kode string `json:"kode"`
	daftarRequest
	NamaPaket string `json:"nama_paket"`
	// Estimasi tagihan per periode (belum PPN) menurut paket.json SAAT
	// mendaftar — supaya tetap terlacak walau harga diubah kemudian.
	Estimasi   int64     `json:"estimasi"`
	Status     string    `json:"status"`
	ReceivedAt time.Time `json:"received_at"`
	IP         string    `json:"ip"`
	UserAgent  string    `json:"user_agent"`
	hubctlResult
}

var reDigits = regexp.MustCompile(`[^0-9]`)

func (d *daftarRequest) normalize() {
	d.Paket = strings.ToLower(clip(d.Paket, 30))
	d.Periode = strings.ToLower(clip(d.Periode, 10))
	d.NamaUsaha = clip(d.NamaUsaha, 150)
	d.JenisUsaha = clip(d.JenisUsaha, 80)
	d.Kota = clip(d.Kota, 80)
	d.Alamat = clip(d.Alamat, 500)
	d.NPWP = reDigits.ReplaceAllString(clip(d.NPWP, 30), "")
	d.Subdomain = strings.ToLower(clip(d.Subdomain, 30))
	d.NamaPIC = clip(d.NamaPIC, 100)
	d.Jabatan = clip(d.Jabatan, 80)
	d.Phone = clip(d.Phone, 20)
	d.Email = strings.ToLower(clip(d.Email, 150))
	d.Catatan = clip(d.Catatan, 2000)
}

// Aturan ini dicerminkan di assets/js/daftar.js (validasi klien); server
// tetap pemutus akhir.
func (d *daftarRequest) validate(pf *paketFile) map[string]string {
	errs := map[string]string{}
	p := pf.find(d.Paket)
	if p == nil {
		errs["paket"] = "Pilih paket"
	} else if p.HargaBulanan > 0 && d.Periode != "bulanan" && d.Periode != "tahunan" {
		errs["periode"] = "Pilih periode pembayaran"
	}
	if d.NamaUsaha == "" {
		errs["nama_usaha"] = "Nama usaha wajib diisi"
	}
	if d.JumlahGudang < 0 || d.JumlahGudang > 10000 {
		errs["jumlah_gudang"] = "Jumlah gudang tidak valid"
	}
	if d.JumlahPengguna < 0 || d.JumlahPengguna > 10000 {
		errs["jumlah_pengguna"] = "Jumlah pengguna tidak valid"
	}
	if d.Kota == "" {
		errs["kota"] = "Kota wajib diisi"
	}
	if d.NPWP != "" && len(d.NPWP) != 15 && len(d.NPWP) != 16 {
		errs["npwp"] = "NPWP harus 15 atau 16 digit"
	}
	if msg := subdomainError(d.Subdomain); msg != "" {
		errs["subdomain"] = msg
	}
	if d.NamaPIC == "" {
		errs["nama_pic"] = "Nama penanggung jawab wajib diisi"
	}
	if d.Phone == "" {
		errs["phone"] = "No. WhatsApp wajib diisi"
	} else if !rePhone.MatchString(d.Phone) {
		errs["phone"] = "Nomor WhatsApp tidak valid"
	}
	// Email WAJIB di sini (opsional di form demo): dipakai untuk akun admin
	// pertama dan pengiriman tagihan.
	if d.Email == "" {
		errs["email"] = "Email wajib diisi"
	} else if !validEmail(d.Email) {
		errs["email"] = "Email tidak valid"
	}
	if !d.Setuju {
		errs["setuju"] = "Centang pernyataan ini untuk melanjutkan"
	}
	return errs
}

// kodeDaftar: HUB-YYMMDD-XXXX, XXXX acak tanpa huruf/angka yang mirip
// (0/O, 1/I/L) supaya mudah didiktekan lewat telepon/WhatsApp.
func kodeDaftar(t time.Time) string {
	const abjad = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		// rand.Read praktis tak pernah gagal; jatuh ke jam sebagai cadangan.
		ns := t.UnixNano()
		for i := range b {
			b[i] = byte(ns >> (8 * i))
		}
	}
	for i := range b {
		b[i] = abjad[int(b[i])%len(abjad)]
	}
	return "HUB-" + t.Format("060102") + "-" + string(b)
}

func daftarHandler(store *jsonlStore, paketPath string, hc *hubControl) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req daftarRequest
		if err := c.BodyParser(&req); err != nil {
			return respond(c, fiber.StatusBadRequest, "Format permintaan tidak valid", nil, nil)
		}
		now := time.Now()
		if strings.TrimSpace(req.Website) != "" {
			// Honeypot: jawab seolah berhasil, jangan simpan.
			return respond(c, fiber.StatusOK, "Pendaftaran diterima", fiber.Map{"kode": kodeDaftar(now)}, nil)
		}

		pf, err := loadPaket(paketPath)
		if err != nil {
			log.Printf("[daftar] paket.json: %v", err)
			return respond(c, fiber.StatusInternalServerError, "Pendaftaran sedang tidak tersedia, silakan hubungi kami lewat WhatsApp.", nil, nil)
		}

		req.normalize()
		if errs := req.validate(pf); len(errs) > 0 {
			return respond(c, fiber.StatusUnprocessableEntity, "Periksa kembali isian Anda", nil, errs)
		}

		p := pf.find(req.Paket)
		if p.HargaBulanan <= 0 {
			req.Periode = "" // paket penawaran: periode ditentukan saat negosiasi
		}
		rec := daftarRecord{
			Kode:          kodeDaftar(now),
			daftarRequest: req,
			NamaPaket:     p.Nama,
			Estimasi:      pf.tagihan(p, req.Periode),
			Status:        "baru",
			ReceivedAt:    now,
			IP:            c.IP(),
			UserAgent:     string(c.Request().Header.UserAgent()),
		}

		if hc != nil {
			alamat := req.Alamat
			if req.Kota != "" {
				alamat = strings.TrimLeft(alamat+", "+req.Kota, ", ")
			}
			rej := teruskanHubctl(c, hc, "daftar", hubctlRegister{
				Email:         req.Email,
				BusinessName:  req.NamaUsaha,
				Phone:         req.Phone,
				FullName:      req.NamaPIC,
				BusinessField: req.JenisUsaha,
				NPWP:          req.NPWP,
				Address:       alamat,
				Subdomain:     req.Subdomain,
			}, &rec.hubctlResult)
			if rej != "" {
				return respond(c, fiber.StatusUnprocessableEntity, rej, nil,
					hubctlFieldError(rej, hubctlFields{"subdomain", "email", "nama_usaha", "phone"}))
			}
		}

		if err := store.append(rec); err != nil {
			log.Printf("[daftar] gagal menyimpan: %v", err)
			if !rec.HubctlOK {
				return respond(c, fiber.StatusInternalServerError, "Gagal menyimpan pendaftaran, silakan hubungi kami lewat WhatsApp.", nil, nil)
			}
		}
		log.Printf("[daftar] %s: %s — paket %s %s hubctl=%d", rec.Kode, req.NamaUsaha, p.Nama, req.Periode, rec.HubctlHTTP)
		// Alamat aplikasi dari hub-control (data.domain, atau data.slug).
		var hub struct {
			Slug   string `json:"slug"`
			Domain string `json:"domain"`
		}
		if rec.HubctlOK {
			_ = json.Unmarshal(rec.HubctlData, &hub)
			if hub.Domain == "" && hub.Slug != "" {
				hub.Domain = hub.Slug + ".hub.xpanca.com"
			}
		}
		return respond(c, fiber.StatusOK, "Pendaftaran diterima", fiber.Map{
			"hubctl":     rec.HubctlOK,
			"domain":     hub.Domain,
			"kode":       rec.Kode,
			"paket":      p.Nama,
			"periode":    req.Periode,
			"estimasi":   rec.Estimasi,
			"penawaran":  p.HargaBulanan <= 0,
			"nama_usaha": req.NamaUsaha,
			"email":      req.Email,
		}, nil)
	}
}
