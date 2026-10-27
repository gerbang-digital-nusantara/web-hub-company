package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

/* ===================== Paket langganan ===================== */

// Sumber paket & harga = katalog hub-control (GET /api/public/pricing, diisi
// admin di menu Paket & Harga), supaya harga di situs selalu sama dengan
// panel. Dibaca server lalu disajikan ke browser lewat GET /api/paket (kartu
// Harga di beranda + halaman Daftar) dan dipakai validasi paket + estimasi
// tagihan di pendaftaran â€” satu sumber untuk ketiganya.
//
// view/assets/data/paket.json = CADANGAN: dipakai bila hub-control belum
// pernah berhasil dibaca sejak server jalan, dan oleh pratinjau GitHub Pages
// (tanpa server Go).
type paketFile struct {
	// "hub-control" | "lokal" (paket.json) â€” untuk diagnosa.
	Sumber string  `json:"sumber"`
	Paket  []paket `json:"paket"`
}

type paket struct {
	ID    string `json:"id"` // code di hub-control
	Nama  string `json:"nama"`
	Untuk string `json:"untuk,omitempty"`
	// 0 = tidak dijual per periode itu.
	HargaBulanan int64    `json:"harga_bulanan"`
	HargaTahunan int64    `json:"harga_tahunan"`
	Rekomendasi  bool     `json:"rekomendasi,omitempty"`
	Fitur        []string `json:"fitur,omitempty"`
	// true = harga lewat penawaran (cta_type "contact"): tanpa periode & estimasi.
	Penawaran bool `json:"penawaran"`
}

// paketLokal = format paket.json. Harga tahunan dihitung dari
// harga_bulanan Ã— (12 âˆ’ diskon_tahunan_bulan).
type paketLokal struct {
	DiskonTahunanBulan int `json:"diskon_tahunan_bulan"`
	Paket              []struct {
		ID           string   `json:"id"`
		Nama         string   `json:"nama"`
		Untuk        string   `json:"untuk"`
		HargaBulanan int64    `json:"harga_bulanan"`
		Rekomendasi  bool     `json:"rekomendasi"`
		Fitur        []string `json:"fitur"`
	} `json:"paket"`
}

func loadPaket(path string) (*paketFile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var pl paketLokal
	if err := json.Unmarshal(b, &pl); err != nil {
		return nil, fmt.Errorf("format tidak valid: %w", err)
	}
	if len(pl.Paket) == 0 {
		return nil, fmt.Errorf("daftar paket kosong")
	}
	if pl.DiskonTahunanBulan < 0 || pl.DiskonTahunanBulan > 11 {
		return nil, fmt.Errorf("diskon_tahunan_bulan harus 0â€“11")
	}
	pf := &paketFile{Sumber: "lokal"}
	for _, p := range pl.Paket {
		pf.Paket = append(pf.Paket, paket{
			ID:           p.ID,
			Nama:         p.Nama,
			Untuk:        p.Untuk,
			HargaBulanan: p.HargaBulanan,
			HargaTahunan: p.HargaBulanan * int64(12-pl.DiskonTahunanBulan),
			Rekomendasi:  p.Rekomendasi,
			Fitur:        p.Fitur,
			Penawaran:    p.HargaBulanan <= 0,
		})
	}
	return pf, nil
}

// find mencocokkan id, atau nama paket (tanpa beda huruf besar/kecil) supaya
// tautan lama seperti daftar.html?paket=bisnis tetap jalan walau code di
// hub-control "business".
func (pf *paketFile) find(id string) *paket {
	for i := range pf.Paket {
		if pf.Paket[i].ID == id {
			return &pf.Paket[i]
		}
	}
	for i := range pf.Paket {
		if strings.EqualFold(pf.Paket[i].Nama, id) {
			return &pf.Paket[i]
		}
	}
	return nil
}

// tersedia = paket p bisa dibayar per periode itu.
func (p *paket) tersedia(periode string) bool {
	switch periode {
	case "bulanan":
		return p.HargaBulanan > 0
	case "tahunan":
		return p.HargaTahunan > 0
	}
	return false
}

// tagihan = estimasi tagihan per periode, belum termasuk PPN. 0 bila paket
// memakai penawaran. Rumusnya harus sama dengan hitungTagihan di paket.js.
func (pf *paketFile) tagihan(p *paket, periode string) int64 {
	if p == nil || p.Penawaran {
		return 0
	}
	if periode == "tahunan" {
		return p.HargaTahunan
	}
	return p.HargaBulanan
}

/* ===================== Katalog hub-control ===================== */

// katalog membaca GET {HUBCTL_BASE_URL}/api/public/pricing (tanpa API Key;
// hub-control membatasi 20 permintaan/menit/IP, jadi hasilnya di-cache).
// Gagal membaca â†’ pakai salinan terakhir yang berhasil; belum pernah
// berhasil â†’ paket.json.
type katalog struct {
	url           string
	lokalPath     string
	ttl           time.Duration
	http          *http.Client
	mu            sync.Mutex
	cache         *paketFile
	diambil       time.Time
	gagalTerakhir time.Time
}

func newKatalog(lokalPath string) *katalog {
	base := strings.TrimRight(getenv("HUBCTL_BASE_URL", "https://hub-control.xpanca.com"), "/")
	return &katalog{
		url:       base + "/api/public/pricing",
		lokalPath: lokalPath,
		ttl:       5 * time.Minute,
		http:      &http.Client{Timeout: 5 * time.Second},
	}
}

func (k *katalog) get(ctx context.Context) (*paketFile, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.cache != nil && time.Since(k.diambil) < k.ttl {
		return k.cache, nil
	}
	// Baru saja gagal: jangan membebani hub-control tiap permintaan.
	if time.Since(k.gagalTerakhir) >= 30*time.Second {
		pf, err := k.ambil(ctx)
		if err == nil {
			k.cache, k.diambil = pf, time.Now()
			return pf, nil
		}
		k.gagalTerakhir = time.Now()
		log.Printf("[paket] katalog hub-control gagal dibaca: %v", err)
	}
	if k.cache != nil {
		return k.cache, nil
	}
	return loadPaket(k.lokalPath)
}

type hubctlPricing struct {
	Code         string   `json:"code"`
	Name         string   `json:"name"`
	Tagline      string   `json:"tagline"`
	PriceMonthly int64    `json:"price_monthly"`
	PriceYearly  int64    `json:"price_yearly"`
	Features     []string `json:"features"`
	Recommended  bool     `json:"recommended"`
	CTAType      string   `json:"cta_type"`
}

func (k *katalog) ambil(ctx context.Context) (*paketFile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, k.url, nil)
	if err != nil {
		return nil, err
	}
	res, err := k.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", res.StatusCode)
	}
	var out struct {
		Status bool            `json:"status"`
		Data   []hubctlPricing `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("respons tidak valid: %w", err)
	}
	if !out.Status || len(out.Data) == 0 {
		return nil, fmt.Errorf("katalog kosong")
	}
	pf := &paketFile{Sumber: "hub-control"}
	for _, d := range out.Data {
		if d.Code == "" || d.Name == "" {
			continue
		}
		pf.Paket = append(pf.Paket, paket{
			ID:           d.Code,
			Nama:         d.Name,
			Untuk:        d.Tagline,
			HargaBulanan: d.PriceMonthly,
			HargaTahunan: d.PriceYearly,
			Rekomendasi:  d.Recommended,
			Fitur:        d.Features,
			Penawaran:    d.CTAType == "contact" || (d.PriceMonthly <= 0 && d.PriceYearly <= 0),
		})
	}
	if len(pf.Paket) == 0 {
		return nil, fmt.Errorf("katalog kosong")
	}
	return pf, nil
}

// paketHandler: GET /api/paket — katalog paket untuk kartu Harga & halaman
// Daftar (paket.js), bentuknya sama dengan yang dipakai validasi server.
func paketHandler(kat *katalog) fiber.Handler {
	return func(c *fiber.Ctx) error {
		pf, err := kat.get(c.UserContext())
		if err != nil {
			log.Printf("[paket] %v", err)
			return respond(c, fiber.StatusServiceUnavailable, "Paket belum bisa dimuat", nil, nil)
		}
		c.Set("Cache-Control", "no-cache")
		return respond(c, fiber.StatusOK, "OK", pf, nil)
	}
}
