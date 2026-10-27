package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

/* ===================== hub-control Partner API ===================== */

// hubControl = klien Partner API hub-control (registrasi tenant coba gratis).
// Form Minta Demo meneruskan pendaftar ke sini supaya tenant
// {subdomain}.hub.xpanca.com langsung tercatat dan tinggal di-Approve admin.
//
// Autentikasi HMAC-SHA256 (dokumentasi: {BASE}/docs.html):
//
//	stringToSign = X-Timestamp + "\n" + METHOD + "\n" + PATH + "\n" + body
//	X-Signature  = HEX(HMAC_SHA256(API_Secret, stringToSign))
//
// PATH tanpa query string. Secret HANYA di server (.env), jangan ke browser.
type hubControl struct {
	baseURL string
	key     string
	secret  string
	http    *http.Client
}

// newHubControl mengembalikan nil bila HUBCTL_* belum diisi — form demo lalu
// kembali ke perilaku lama (hanya dicatat ke demo-requests.jsonl).
func newHubControl() *hubControl {
	h := &hubControl{
		baseURL: strings.TrimRight(getenv("HUBCTL_BASE_URL", "https://hub-control.xpanca.com"), "/"),
		key:     getenv("HUBCTL_API_KEY", ""),
		secret:  getenv("HUBCTL_API_SECRET", ""),
		http:    &http.Client{Timeout: 15 * time.Second},
	}
	if h.key == "" || h.secret == "" {
		return nil
	}
	return h
}

// hubctlResponse = amplop hub-control: {status, message, data}.
type hubctlResponse struct {
	Status  bool            `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type hubctlRegister struct {
	Email         string `json:"email"`
	BusinessName  string `json:"business_name"`
	Phone         string `json:"phone,omitempty"`
	FullName      string `json:"full_name,omitempty"`
	BusinessField string `json:"business_field,omitempty"`
	NPWP          string `json:"npwp,omitempty"`
	Address       string `json:"address,omitempty"`
	Subdomain     string `json:"subdomain,omitempty"`
	// Paket produk: intent "trial" (Coba Gratis) | "subscribe" (Daftar),
	// package = nama paket di paket.json (mis. "Bisnis"), period =
	// "bulanan" | "tahunan" (hub-control juga menerima monthly/yearly).
	Intent  string `json:"intent,omitempty"`
	Package string `json:"package,omitempty"`
	Period  string `json:"period,omitempty"`
}

func (h *hubControl) sign(ts, method, path string, body []byte) string {
	m := hmac.New(sha256.New, []byte(h.secret))
	m.Write([]byte(ts + "\n" + method + "\n" + path + "\n"))
	m.Write(body)
	return hex.EncodeToString(m.Sum(nil))
}

// do mengirim request bertanda tangan. HTTP status dikembalikan apa adanya
// (400 = ditolak hub-control, mis. email/subdomain sudah dipakai); error
// hanya untuk gangguan jaringan / respons yang bukan JSON.
//
// clientIP = IP pengunjung asli (c.IP(); isi PROXY_HEADER bila di belakang
// Nginx), diteruskan sebagai X-Forwarded-For supaya Audit Trail hub-control
// tidak mencatat semua pendaftaran dari satu IP server situs ini.
func (h *hubControl) do(ctx context.Context, clientIP, method, path string, query url.Values, body []byte) (int, *hubctlResponse, error) {
	u := h.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, u, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	req.Header.Set("X-Api-Key", h.key)
	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Signature", h.sign(ts, method, path, body))
	if clientIP != "" {
		req.Header.Set("X-Forwarded-For", clientIP)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := h.http.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return res.StatusCode, nil, err
	}
	var out hubctlResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return res.StatusCode, nil, fmt.Errorf("respons hub-control bukan JSON (HTTP %d)", res.StatusCode)
	}
	return res.StatusCode, &out, nil
}

func (h *hubControl) register(ctx context.Context, clientIP string, r hubctlRegister) (int, *hubctlResponse, error) {
	body, err := json.Marshal(r)
	if err != nil {
		return 0, nil, err
	}
	return h.do(ctx, clientIP, http.MethodPost, "/api/public/register", nil, body)
}

func (h *hubControl) checkSubdomain(ctx context.Context, clientIP, name string) (int, *hubctlResponse, error) {
	return h.do(ctx, clientIP, http.MethodGet, "/api/public/check-subdomain", url.Values{"name": {name}}, nil)
}

// hubctlFieldError memetakan penolakan 400 hub-control ke kolom form, mis.
// "email ini sudah pernah digunakan untuk mendaftar" → kolom email.
// fields = nama kolom form untuk subdomain/email/usaha/phone (beda tiap form).
type hubctlFields struct{ Subdomain, Email, Usaha, Phone string }

func hubctlFieldError(msg string, f hubctlFields) map[string]string {
	m := strings.ToLower(msg)
	switch {
	case strings.Contains(m, "subdomain"):
		return map[string]string{f.Subdomain: msg}
	case strings.Contains(m, "email"):
		return map[string]string{f.Email: msg}
	case strings.Contains(m, "usaha") || strings.Contains(m, "business"):
		return map[string]string{f.Usaha: msg}
	case strings.Contains(m, "phone") || strings.Contains(m, "telepon"):
		return map[string]string{f.Phone: msg}
	}
	return nil
}

/* ===================== Subdomain ===================== */

// reSubdomain = aturan FORMAT hub-control: 3–30 karakter a-z/0-9/minus, tidak
// diawali/diakhiri minus, tanpa "--". Ketersediaan & nama terlarang TIDAK
// ditebak di sini — acuannya hasil check-subdomain hub-control.
var reSubdomain = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{1,28}[a-z0-9])$`)

const msgSubdomainFormat = "3–30 karakter: huruf kecil, angka, atau tanda hubung (tanpa \"--\")"

func subdomainError(s string) string {
	if s != "" && (!reSubdomain.MatchString(s) || strings.Contains(s, "--")) {
		return msgSubdomainFormat
	}
	return ""
}
