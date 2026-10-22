package main

import (
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
type demoRequest struct {
	Name    string `json:"name"`
	Company string `json:"company"`
	Phone   string `json:"phone"`
	Email   string `json:"email"`
	Package string `json:"package"`
	Message string `json:"message"`
	Website string `json:"website,omitempty"`
}

type demoRecord struct {
	demoRequest
	ReceivedAt time.Time `json:"received_at"`
	IP         string    `json:"ip"`
	UserAgent  string    `json:"user_agent"`
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
	d.Email = clip(d.Email, 150)
	d.Package = clip(d.Package, 50)
	d.Message = clip(d.Message, 2000)
}

func (d *demoRequest) validate() map[string]string {
	errs := map[string]string{}
	if d.Name == "" {
		errs["name"] = "Nama wajib diisi"
	}
	if d.Phone == "" && d.Email == "" {
		errs["phone"] = "Isi nomor WhatsApp atau email"
	}
	if d.Phone != "" && !rePhone.MatchString(d.Phone) {
		errs["phone"] = "Nomor WhatsApp tidak valid"
	}
	if d.Email != "" && !validEmail(d.Email) {
		errs["email"] = "Email tidak valid"
	}
	return errs
}

func demoHandler(store *jsonlStore) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req demoRequest
		if err := c.BodyParser(&req); err != nil {
			return respond(c, fiber.StatusBadRequest, "Format permintaan tidak valid", nil, nil)
		}
		if strings.TrimSpace(req.Website) != "" {
			return respond(c, fiber.StatusOK, "Terima kasih, tim kami akan segera menghubungi Anda.", nil, nil)
		}
		req.normalize()
		if errs := req.validate(); len(errs) > 0 {
			return respond(c, fiber.StatusUnprocessableEntity, "Periksa kembali isian Anda", nil, errs)
		}
		rec := demoRecord{
			demoRequest: req,
			ReceivedAt:  time.Now(),
			IP:          c.IP(),
			UserAgent:   string(c.Request().Header.UserAgent()),
		}
		if err := store.append(rec); err != nil {
			log.Printf("[demo] gagal menyimpan: %v", err)
			return respond(c, fiber.StatusInternalServerError, "Gagal menyimpan permintaan, silakan hubungi kami lewat WhatsApp.", nil, nil)
		}
		log.Printf("[demo] permintaan baru: %s (%s)", req.Name, req.Company)
		return respond(c, fiber.StatusOK, "Terima kasih, tim kami akan segera menghubungi Anda.", nil, nil)
	}
}
