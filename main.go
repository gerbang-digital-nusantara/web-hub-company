package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
)

/* ===================== Helpers ===================== */

// Amplop respons sama dengan hub-api-v2 / web-hub-v3: {code,status,message,data}.
type apiResponse struct {
	Code    int         `json:"code"`
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Errors  interface{} `json:"errors,omitempty"`
}

func getenv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultValue
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func respond(c *fiber.Ctx, code int, msg string, data, errs interface{}) error {
	return c.Status(code).JSON(apiResponse{
		Code: code, Status: code < 400, Message: msg, Data: data, Errors: errs,
	})
}

// clip = trim spasi + potong ke maks `max` karakter (rune, bukan byte).
func clip(s string, max int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) > max {
		s = string([]rune(s)[:max])
	}
	return s
}

/* ===================== Penyimpanan JSON Lines ===================== */

// jsonlStore menulis tiap kiriman form sebagai SATU baris JSON (JSON Lines),
// mis. DATA_DIR/demo-requests.jsonl dan DATA_DIR/pendaftaran.jsonl. Sengaja
// tanpa database: situs ini tidak punya DB, dan berkas append-only cukup
// untuk dibaca tim sales (`tail -f`, atau dibuka di editor). Mutex menjaga
// baris tidak bertumpuk saat dua kiriman datang bersamaan.
type jsonlStore struct {
	mu   sync.Mutex
	path string
}

func (s *jsonlStore) append(rec interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	line, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	_, err = f.Write(append(line, '\n'))
	return err
}

// Maks `max` kiriman per menit per IP — cukup untuk manusia, menahan spam.
func perMinute(max int) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        max,
		Expiration: time.Minute,
		LimitReached: func(c *fiber.Ctx) error {
			return respond(c, fiber.StatusTooManyRequests, "Terlalu banyak permintaan, coba lagi sebentar.", nil, nil)
		},
	})
}

/* ===================== Static ===================== */

// HTML: no-cache (browser wajib bertanya ulang, dijawab 304 bila sama), supaya
// perubahan konten langsung terlihat. paket.json juga no-cache karena harga
// harus selalu terbaru. Aset lain (css/js/gambar) di-cache 1 hari —
// halaman memuatnya dengan ?v= sehingga versi baru tetap tertarik.
func cacheHeaders(c *fiber.Ctx) error {
	p := c.Path()
	switch {
	case p == "/" || strings.HasSuffix(p, ".html") || strings.HasSuffix(p, ".json"):
		c.Set("Cache-Control", "no-cache")
	case strings.HasPrefix(p, "/assets/"):
		c.Set("Cache-Control", "public, max-age=86400")
	}
	return c.Next()
}

type stores struct {
	demo   *jsonlStore
	daftar *jsonlStore
}

func buildApp(webRoot string, st stores, hc *hubControl) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:   "web-hub-company",
		BodyLimit: 64 * 1024,
		// Di belakang Nginx: IP asli dari X-Forwarded-For (dipakai limiter & log).
		ProxyHeader: getenv("PROXY_HEADER", ""),
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			if strings.HasPrefix(c.Path(), "/api/") {
				return respond(c, code, err.Error(), nil, nil)
			}
			if code == fiber.StatusNotFound {
				if p := filepath.Join(webRoot, "404.html"); fileExists(p) {
					c.Set("Cache-Control", "no-cache")
					return c.Status(code).SendFile(p)
				}
			}
			return c.Status(code).SendString(err.Error())
		},
	})

	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "${time} ${status} ${method} ${path} ${latency}\n",
	}))
	app.Use(compress.New())

	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	paketPath := filepath.Join(webRoot, "assets", "data", "paket.json")
	app.Post("/api/demo", perMinute(5), demoHandler(st.demo, hc))
	app.Get("/api/demo/subdomain", perMinute(30), subdomainHandler(hc))
	app.Post("/api/daftar", perMinute(5), daftarHandler(st.daftar, paketPath))

	// URL rapi tanpa .html: /daftar → daftar.html.
	app.Get("/daftar", func(c *fiber.Ctx) error {
		return c.Redirect("/daftar.html"+queryOf(c), fiber.StatusMovedPermanently)
	})

	app.Use(cacheHeaders)
	app.Static("/", webRoot, fiber.Static{
		Compress:  false, // sudah lewat middleware compress
		ByteRange: true,
		Index:     "index.html",
	})

	app.Use(func(c *fiber.Ctx) error { return fiber.ErrNotFound })
	return app
}

func queryOf(c *fiber.Ctx) string {
	if q := string(c.Request().URI().QueryString()); q != "" {
		return "?" + q
	}
	return ""
}

func main() {
	_ = godotenv.Load()

	webRoot := getenv("WEB_ROOT", "./view")
	if !fileExists(filepath.Join(webRoot, "index.html")) {
		log.Fatalf("index.html tidak ditemukan di %s (atur WEB_ROOT)", webRoot)
	}
	// Gagal cepat bila paket.json rusak — lebih baik tidak jalan daripada
	// form daftar menolak semua pendaftar.
	if _, err := loadPaket(filepath.Join(webRoot, "assets", "data", "paket.json")); err != nil {
		log.Fatalf("paket.json: %v", err)
	}

	dataDir := getenv("DATA_DIR", "./data")
	st := stores{
		demo:   &jsonlStore{path: filepath.Join(dataDir, "demo-requests.jsonl")},
		daftar: &jsonlStore{path: filepath.Join(dataDir, "pendaftaran.jsonl")},
	}

	hc := newHubControl()
	if hc == nil {
		log.Printf("[web-hub-company] HUBCTL_API_KEY/SECRET kosong: form demo hanya dicatat ke jsonl")
	} else {
		log.Printf("[web-hub-company] form demo diteruskan ke hub-control %s (key %s)", hc.baseURL, hc.key)
	}

	port := getenv("PORT", "3076")
	app := buildApp(webRoot, st, hc)
	log.Printf("[web-hub-company] serving %s on :%s", webRoot, port)
	if err := app.Listen(fmt.Sprintf(":%s", port)); err != nil {
		log.Fatal(err)
	}
}
