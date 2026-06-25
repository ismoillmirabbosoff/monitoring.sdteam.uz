package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holmonitoring backend uchun barcha sozlamalar (muhit o'zgaruvchilaridan).
type Config struct {
	// HTTP
	HTTPAddr    string   // masalan ":8080"
	CORSOrigins []string // ruxsat etilgan frontend domenlari
	WebDir      string   // build qilingan UI papkasi (statik xizmat)
	WSPort      string   // OnlinePBX websocket porti (frontend uchun)

	// OnlinePBX
	OnpbxDomain  string // masalan "pbx12127.onpbx.ru"
	OnpbxAPIKey  string // kabinetdan olingan api_key
	OnpbxAPIID   string // kabinetdan olingan api_id
	OnpbxBase    string // "https://api2.onlinepbx.ru"
	OnpbxKeysURL string // ixtiyoriy: token/auth_key ni shu upstream'dan olish
	//             (masalan https://phone.sdteam.uz/api/monitoring/keys).
	//             Berilsa, backend o'zi auth.json qilmaydi.
	OnpbxToken string // ixtiyoriy: to'g'ridan-to'g'ri key_and_id (proxy'siz, mustaqil)
	OnpbxWSKey string // websocket auth_key (frontend uchun)

	// Postgres
	DatabaseURL string // postgres://user:pass@host:5432/db?sslmode=disable

	// Sync worker
	SyncInterval  time.Duration // history'ni qancha vaqtda bir tortish
	SyncLookback  time.Duration // har sinxda nechta orqaga qarab tortish (overlap)
	HistoryLimit  int           // bitta so'rovdagi maksimal yozuvlar
}

func Load() (*Config, error) {
	cfg := &Config{
		HTTPAddr:     getEnv("HTTP_ADDR", ":8080"),
		CORSOrigins:  splitCSV(getEnv("CORS_ORIGINS", "*")),
		WebDir:       getEnv("WEB_DIR", "./web/dist"),
		WSPort:       getEnv("ONPBX_WS_PORT", "3342"),
		OnpbxDomain:  os.Getenv("ONPBX_DOMAIN"),
		OnpbxAPIKey:  os.Getenv("ONPBX_API_KEY"),
		OnpbxAPIID:   os.Getenv("ONPBX_API_ID"),
		OnpbxBase:    getEnv("ONPBX_BASE", "https://api2.onlinepbx.ru"),
		OnpbxKeysURL: os.Getenv("ONPBX_KEYS_URL"),
		OnpbxToken:   os.Getenv("ONPBX_STATIC_TOKEN"),
		OnpbxWSKey:   os.Getenv("ONPBX_WS_KEY"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		SyncInterval: getDuration("SYNC_INTERVAL", 60*time.Second),
		SyncLookback: getDuration("SYNC_LOOKBACK", 6*time.Hour),
		HistoryLimit: getInt("HISTORY_LIMIT", 1000),
	}

	if cfg.OnpbxDomain == "" {
		return nil, fmt.Errorf("ONPBX_DOMAIN majburiy (masalan pbx12127.onpbx.ru)")
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL majburiy")
	}
	return cfg, nil
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getDuration(k string, def time.Duration) time.Duration {
	if v := os.Getenv(k); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func splitCSV(s string) []string {
	out := []string{}
	cur := ""
	for _, r := range s {
		if r == ',' {
			if cur != "" {
				out = append(out, trim(cur))
			}
			cur = ""
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, trim(cur))
	}
	return out
}

func trim(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
