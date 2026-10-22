package main

import (
	"encoding/json"
	"fmt"
	"os"
)

/* ===================== Paket langganan ===================== */

// paket.json (view/assets/data/paket.json) = SATU-SATUNYA sumber paket &
// harga. Dibaca browser (kartu Harga di beranda + halaman Daftar) dan server
// (validasi paket + estimasi tagihan yang ikut disimpan di pendaftaran).
// Dibaca ulang setiap ada pendaftaran, jadi mengubah harga tidak perlu
// restart server.
type paketFile struct {
	// Bayar tahunan = harga_bulanan × (12 − diskon_tahunan_bulan).
	DiskonTahunanBulan int     `json:"diskon_tahunan_bulan"`
	Paket              []paket `json:"paket"`
}

type paket struct {
	ID   string `json:"id"`
	Nama string `json:"nama"`
	// 0 / kosong = harga lewat penawaran (Enterprise): tanpa periode & estimasi.
	HargaBulanan int64 `json:"harga_bulanan"`
}

func loadPaket(path string) (*paketFile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var pf paketFile
	if err := json.Unmarshal(b, &pf); err != nil {
		return nil, fmt.Errorf("format tidak valid: %w", err)
	}
	if len(pf.Paket) == 0 {
		return nil, fmt.Errorf("daftar paket kosong")
	}
	if pf.DiskonTahunanBulan < 0 || pf.DiskonTahunanBulan > 11 {
		return nil, fmt.Errorf("diskon_tahunan_bulan harus 0–11")
	}
	return &pf, nil
}

func (pf *paketFile) find(id string) *paket {
	for i := range pf.Paket {
		if pf.Paket[i].ID == id {
			return &pf.Paket[i]
		}
	}
	return nil
}

// tagihan = estimasi tagihan per periode, belum termasuk PPN. 0 bila paket
// memakai penawaran. Rumusnya harus sama dengan hitungTagihan di paket.js.
func (pf *paketFile) tagihan(p *paket, periode string) int64 {
	if p == nil || p.HargaBulanan <= 0 {
		return 0
	}
	if periode == "tahunan" {
		return p.HargaBulanan * int64(12-pf.DiskonTahunanBulan)
	}
	return p.HargaBulanan
}
