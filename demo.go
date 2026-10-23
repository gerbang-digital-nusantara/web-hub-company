package main

import (
	"encoding/json"
	"log"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

/* ===================== Form Minta Demo ===================== */

// demoRequest = isian form "Minta Demo" di halaman depan.
//
// `Website` adalah HONEYPOT: kolomnya disembunyikan dari manusia lewat CSS,
// jadi yang mengisinya hampir pasti bot. Permintaan seperti itu dijawab
// "berhasil" tetapi tidak disimpan, supaya bot tidak belajar bahwa ia ditolak.
//
// Bila HUBCTL_* diisi (lihat hubcontrol.go), pendaftar juga diteruskan ke
// Partner API hub-control sebagai tenant coba gratis {subdomain}.hub.xpanca.com
// (menunggu Approve admin). Karena itu nama usaha & email jadi WAJIB.
type demoRequest struct {
	Name      string `json:"name"`
	Company   string `json:"company"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	Subdomain string `json:"subdomain"`
	Package   string `json:"package"`
	Message   string `json:"message"`
	Website   string `json:"website,omitempty"`
}

type demoRecord struct {
	demoRequest
	ReceivedAt time.Time `json:"received_at"`
	IP         string    `json:"ip"`
	UserAgent  string    `json:"user_agent"`
	// Hasil penerusan ke hub-control (kosong bila HUBCTL_* belum diisi).
	// Baris tetap disimpan walau hub-control gagal, supaya tim sales bisa
	// menindaklanjuti / mendaftarkan manual.
	HubctlHTTP    int             `json:"hubctl_http,omitempty"`
	HubctlOK      bool            `json:"hubctl_ok,omitempty"`
	HubctlMessage string          `json:"hubctl_message,omitempty"`
	HubctlData    json.RawMessage `json:"hubctl_data,omitempty"`
}

var rePhone = regexp.MustCompile(`^[0-9+\-\s()]{8,20}$`)

func validEmail(s string) bool {
	a, err := mail.ParseAddress(s)
	return err == nil && a.Address == s
}

func (d *demoRequest) normalize() {
	d.Name = clip(d.Name, 100)
	d.Company = clip(d.Company, 150)
	d.Phone = clip(d.Phone, 20)
	d.Email = strings.ToLower(clip(d.Email, 150))
	d.Subdomain = strings.ToLower(clip(d.Subdomain, 30))
	d.Package = clip(d.Package, 50)
	d.Message = clip(d.Message, 2000)
}

// trial = pendaftar akan diteruskan ke hub-control (butuh nama usaha & email).
// Aturan ini dicerminkan di assets/js/main.js.
func (d *demoRequest) validate(trial bool) map[string]string {
	errs := map[string]string{}
	if d.Name == "" {
		errs["name"] = "Nama wajib diisi"
	}
	if trial && d.Company == "" {
		errs["company"] = "Nama usaha wajib diisi"
	}
	if d.Phone == "" && d.Email == "" {
		errs["phone"] = "Isi nomor WhatsApp atau email"
	}
	if d.Phone != "" && !rePhone.MatchString(d.Phone) {
		errs["phone"] = "Nomor WhatsApp tidak valid"
	}
	if trial && d.Email == "" {
		errs["email"] = "Email wajib diisi"
	} else if d.Email != "" && !validEmail(d.Email) {
		errs["email"] = "Email tidak valid"
	}
	if msg := subdomainError(d.Subdomain); msg != "" {
		errs["subdomain"] = msg
	}
	return errs
}

// subdomainError: aturan format yang sama dengan hub-control (3–30 karakter,
// a-z/0-9/minus, tidak diawali/diakhiri minus) + nama yang kita pakai sendiri.
// Kosong = boleh (hub-control membuatnya otomatis dari nama usaha).
func subdomainError(s string) string {
	switch {
	case s == "":
		return ""
	case !reSubdomain.MatchString(s):
		return "3–30 karakter: huruf kecil, angka, atau tanda hubung"
	case subdomainDipakai[s]:
		return "Alamat ini tidak tersedia, coba yang lain"
	}
	return ""
}

// hubctlFieldError memetakan penolakan 400 hub-control ke kolom form,
// mis. "email ini sudah pernah digunakan untuk mendaftar" → email.
func hubctlFieldError(msg string) map[string]string {
	m := strings.ToLower(msg)
	switch {
	case strings.Contains(m, "subdomain"):
		return map[string]string{"subdomain": msg}
	case strings.Contains(m, "email"):
		return map[string]string{"email": msg}
	case strings.Contains(m, "usaha") || strings.Contains(m, "business"):
		return map[string]string{"company": msg}
	case strings.Contains(m, "phone") || strings.Contains(m, "telepon"):
		// mis. "phone ini sudah pernah digunakan untuk mendaftar"
		return map[string]string{"phone": msg}
	}
	return nil
}

const msgDemoDiterima = "Terima kasih, tim kami akan segera menghubungi Anda."

func demoHandler(store *jsonlStore, hc *hubControl) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req demoRequest
		if err := c.BodyParser(&req); err != nil {
			return respond(c, fiber.StatusBadRequest, "Format permintaan tidak valid", nil, nil)
		}
		if strings.TrimSpace(req.Website) != "" {
			return respond(c, fiber.StatusOK, msgDemoDiterima, nil, nil)
		}
		req.normalize()
		if errs := req.validate(hc != nil); len(errs) > 0 {
			return respond(c, fiber.StatusUnprocessableEntity, "Periksa kembali isian Anda", nil, errs)
		}
		rec := demoRecord{
			demoRequest: req,
			ReceivedAt:  time.Now(),
			IP:          c.IP(),
			UserAgent:   string(c.Request().Header.UserAgent()),
		}

		msg := msgDemoDiterima
		if hc != nil {
			code, res, err := hc.register(c.UserContext(), hubctlRegister{
				Email:        req.Email,
				BusinessName: req.Company,
				Phone:        req.Phone,
				FullName:     req.Name,
				Subdomain:    req.Subdomain,
			})
			rec.HubctlHTTP = code
			switch {
			case err != nil:
				rec.HubctlMessage = err.Error()
			case res.Status:
				rec.HubctlOK = true
				rec.HubctlMessage = res.Message
				rec.HubctlData = res.Data
				msg = "Pendaftaran coba gratis diterima. Akun Anda aktif setelah disetujui tim kami — kabarnya kami kirim ke email Anda."
			case code == fiber.StatusBadRequest:
				// Ditolak karena isian (email/subdomain sudah dipakai, dst.):
				// kembalikan ke pendaftar untuk diperbaiki, tidak disimpan.
				log.Printf("[demo] hub-control menolak %s: %s", req.Email, res.Message)
				return respond(c, fiber.StatusUnprocessableEntity, res.Message, nil, hubctlFieldError(res.Message))
			default:
				rec.HubctlMessage = res.Message
			}
			if !rec.HubctlOK {
				// 401/5xx/jaringan: bukan salah pendaftar. Tetap dicatat dan
				// dijawab sebagai permintaan demo biasa; tim yang menindaklanjuti.
				log.Printf("[demo] hub-control gagal (HTTP %d): %s", code, rec.HubctlMessage)
			}
		}

		if err := store.append(rec); err != nil {
			log.Printf("[demo] gagal menyimpan: %v", err)
			// Sudah tercatat di hub-control → jangan suruh mengulang.
			if !rec.HubctlOK {
				return respond(c, fiber.StatusInternalServerError, "Gagal menyimpan permintaan, silakan hubungi kami lewat WhatsApp.", nil, nil)
			}
		}
		log.Printf("[demo] permintaan baru: %s (%s) hubctl=%d", req.Name, req.Company, rec.HubctlHTTP)
		var data interface{}
		if rec.HubctlOK {
			data = rec.HubctlData
		}
		return respond(c, fiber.StatusOK, msg, data, nil)
	}
}

// subdomainHandler: GET /api/demo/subdomain?name=… — cek ketersediaan
// {name}.hub.xpanca.com lewat hub-control untuk petunjuk langsung di form.
// Hasilnya bukan reservasi; nama baru terkunci saat register berhasil.
func subdomainHandler(hc *hubControl) fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := strings.ToLower(clip(c.Query("name"), 30))
		if name == "" {
			return respond(c, fiber.StatusBadRequest, "Isi nama subdomain", nil, nil)
		}
		if msg := subdomainError(name); msg != "" {
			return respond(c, fiber.StatusOK, msg, fiber.Map{"subdomain": name, "available": false, "reason": msg}, nil)
		}
		if hc == nil {
			return respond(c, fiber.StatusServiceUnavailable, "Pengecekan subdomain belum diaktifkan", nil, nil)
		}
		code, res, err := hc.checkSubdomain(c.UserContext(), name)
		if err != nil || !res.Status {
			log.Printf("[demo] cek subdomain gagal (HTTP %d): %v", code, err)
			return respond(c, fiber.StatusBadGateway, "Pengecekan subdomain sedang tidak tersedia", nil, nil)
		}
		return respond(c, fiber.StatusOK, res.Message, res.Data, nil)
	}
}
