// Package api HTTP endpointlarni e'lon qiladi (frontend kontraktiga mos).
package api

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/salesdoc/monitoring-api/internal/email"
	"github.com/salesdoc/monitoring-api/internal/livestate"
	"github.com/salesdoc/monitoring-api/internal/onlinepbx"
	"github.com/salesdoc/monitoring-api/internal/store"
)

type Server struct {
	store     *store.Store
	pbx       *onlinepbx.Client
	email     *email.Sender
	bridge    *livestate.Bridge
	origins   []string
	domain    string
	wsPort    string
	webDir    string
	adminPass string

	globalLimiter   *ipLimiter // DDoS (barcha so'rovlar)
	authLimiter     *ipLimiter // brute force (login/verify)
	feedbackLimiter *ipLimiter // public feedback spam
}

// SetBridge jonli holat bridge'ini o'rnatadi (ixtiyoriy).
func (s *Server) SetBridge(b *livestate.Bridge) { s.bridge = b }

func NewServer(st *store.Store, pbx *onlinepbx.Client, em *email.Sender, origins []string, domain, wsPort, webDir, adminPass string) *Server {
	return &Server{
		store: st, pbx: pbx, email: em, origins: origins, domain: domain,
		wsPort: wsPort, webDir: webDir, adminPass: adminPass,
		globalLimiter:   newIPLimiter(1500, time.Minute),   // IP'ga 1500 so'rov/daqiqa
		authLimiter:     newIPLimiter(20, 5*time.Minute),   // IP'ga 20 login urinishi/5 daqiqa
		feedbackLimiter: newIPLimiter(30, time.Minute),     // IP'ga 30 baho/daqiqa
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /api/config", s.handleConfig)
	mux.HandleFunc("GET /api/monitoring/keys", s.handleKeys)
	mux.HandleFunc("GET /api/monitoring/data", s.handleData)
	mux.HandleFunc("GET /api/monitoring/bigData", s.handleBigData)
	mux.HandleFunc("GET /api/monitoring/operatorTime", s.handleOperatorTime)
	mux.HandleFunc("GET /api/monitoring/fifo", s.handleFifo)
	mux.HandleFunc("GET /api/monitoring/users", s.handleUsers)
	mux.HandleFunc("GET /api/monitoring/hidden", s.handleHidden)
	mux.HandleFunc("GET /api/monitoring/stats", s.handleStats)
	mux.HandleFunc("GET /api/monitoring/recording", s.handleRecording)
	mux.HandleFunc("GET /api/monitoring/liveState", s.handleLiveState)
	// --- Auth (email + kod) — brute force himoyasi (authLimiter) ---
	mux.HandleFunc("POST /api/auth/login", s.limit(s.authLimiter, s.handleLogin))
	mux.HandleFunc("POST /api/auth/verify", s.limit(s.authLimiter, s.handleVerify))
	mux.HandleFunc("GET /api/auth/me", s.handleMe)
	mux.HandleFunc("POST /api/auth/logout", s.handleLogout)
	// --- Admin: foydalanuvchilar va sessiyalar ---
	mux.HandleFunc("GET /api/admin/users", s.requireAdmin(s.handleListUsers))
	mux.HandleFunc("POST /api/admin/users", s.requireAdmin(s.handleCreateUser))
	mux.HandleFunc("PATCH /api/admin/users/{id}", s.requireAdmin(s.handleUpdateUser))
	mux.HandleFunc("DELETE /api/admin/users/{id}", s.requireAdmin(s.handleDeleteUser))
	mux.HandleFunc("GET /api/admin/sessions", s.requireAdmin(s.handleListSessions))
	mux.HandleFunc("DELETE /api/admin/sessions/{token}", s.requireAdmin(s.handleRevokeSession))
	// --- Anketa ---
	mux.HandleFunc("GET /api/survey/questions", s.requireAuth(s.handleActiveQuestions))
	mux.HandleFunc("GET /api/survey/config", s.requireAuth(s.handleActiveSurveyConfig))
	mux.HandleFunc("GET /api/admin/survey/config", s.requireAdmin(s.handleGetSurveyConfig))
	mux.HandleFunc("PUT /api/admin/survey/config", s.requireAdmin(s.handleSaveSurveyConfig))
	mux.HandleFunc("POST /api/survey/responses", s.requireAuth(s.handleSaveResponse))
	mux.HandleFunc("GET /api/survey/responses", s.requireAuth(s.handleListResponses))
	mux.HandleFunc("GET /api/admin/survey/questions", s.requireAdmin(s.handleListQuestions))
	mux.HandleFunc("POST /api/admin/survey/questions", s.requireAdmin(s.handleCreateQuestion))
	mux.HandleFunc("PATCH /api/admin/survey/questions/{id}", s.requireAdmin(s.handleUpdateQuestion))
	mux.HandleFunc("DELETE /api/admin/survey/questions/{id}", s.requireAdmin(s.handleDeleteQuestion))
	// --- Admin (parol bilan himoyalangan) ---
	mux.HandleFunc("POST /api/admin/login", s.limit(s.authLimiter, s.handleAdminLogin))
	mux.HandleFunc("GET /api/admin/employees", s.requireAdmin(s.handleListEmployees))
	mux.HandleFunc("POST /api/admin/employees", s.requireAdmin(s.handleAddEmployee))
	mux.HandleFunc("GET /api/admin/employees/{id}", s.requireAdmin(s.handleEmployeeDetail))
	mux.HandleFunc("PATCH /api/admin/employees/{id}", s.requireAdmin(s.handleUpdateEmployee))
	mux.HandleFunc("PATCH /api/admin/employees/by-ext/{ext}", s.requireAdmin(s.handleSetHiddenByExt))
	mux.HandleFunc("GET /api/admin/servers", s.requireAdmin(s.handleListServers))
	mux.HandleFunc("POST /api/admin/servers", s.requireAdmin(s.handleAddServer))
	mux.HandleFunc("PATCH /api/admin/servers/{id}", s.requireAdmin(s.handleUpdateServer))
	mux.HandleFunc("DELETE /api/admin/servers/{id}", s.requireAdmin(s.handleDeleteServer))
	// --- Ish vaqti + Audit log ---
	mux.HandleFunc("GET /api/admin/work-hours", s.requireAdmin(s.handleListWorkHours))
	mux.HandleFunc("POST /api/admin/work-hours", s.requireAdmin(s.handleSaveWorkHours))
	mux.HandleFunc("GET /api/admin/audit-log", s.requireAdmin(s.handleListAudit))
	// --- Mijoz baholari (otzyvlar) ---
	mux.HandleFunc("POST /api/feedback", s.limit(s.feedbackLimiter, s.handleSubmitFeedback)) // public + spam himoya
	mux.HandleFunc("GET /api/admin/feedback", s.requireAdmin(s.handleListFeedback))
	// --- Operator ballari (avtomatizatsiya) ---
	mux.HandleFunc("GET /api/admin/scores", s.requireAdmin(s.handleListScores))
	mux.HandleFunc("POST /api/admin/scores", s.requireAdmin(s.handleAddScore))
	mux.HandleFunc("DELETE /api/admin/scores/{id}", s.requireAdmin(s.handleDeleteScore))
	// statik UI (SPA fallback bilan) — barcha boshqa GET so'rovlar
	mux.HandleFunc("GET /", s.handleStatic)
	return s.secure(s.cors(mux))
}

// handleConfig frontend uchun ochiq sozlamalar (auth talab qilmaydi).
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"domain": s.domain,
		"wsPort": s.wsPort,
	})
}

// handleStatic build qilingan Vue UI'ni xizmat qiladi; topilmasa index.html (SPA).
func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if s.webDir == "" {
		writeErr(w, http.StatusNotFound, "UI build qilinmagan (WEB_DIR yo'q)")
		return
	}
	clean := filepath.Clean(r.URL.Path)
	full := filepath.Join(s.webDir, clean)
	// papkadan tashqariga chiqishni oldini olish
	if !strings.HasPrefix(full, filepath.Clean(s.webDir)) {
		http.NotFound(w, r)
		return
	}
	if fi, err := os.Stat(full); err == nil && !fi.IsDir() {
		http.ServeFile(w, r, full)
		return
	}
	// SPA fallback
	http.ServeFile(w, r, filepath.Join(s.webDir, "index.html"))
}

// ---- CORS ----

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if s.allowOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", originOrStar(origin, s.origins))
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Admin-Password")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) allowOrigin(origin string) bool {
	for _, o := range s.origins {
		if o == "*" || o == origin {
			return true
		}
	}
	return false
}

func originOrStar(origin string, allowed []string) string {
	for _, o := range allowed {
		if o == "*" {
			if origin != "" {
				return origin // credentials uchun aniq origin qaytaramiz
			}
			return "*"
		}
	}
	return origin
}

// ---- Handlers ----

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// /api/monitoring/keys → {key_and_id, auth_key}
func (s *Server) handleKeys(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	token, err := s.pbx.Token(ctx)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "auth: "+err.Error())
		return
	}
	wsKey, _ := s.pbx.WSKey(ctx)
	writeJSON(w, http.StatusOK, map[string]string{
		"key_and_id": token,
		"auth_key":   wsKey,
	})
}

// /api/monitoring/data?gateway=&from=&to=  (unix soniya)
func (s *Server) handleData(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, _ := strconv.ParseInt(q.Get("from"), 10, 64)
	to, _ := strconv.ParseInt(q.Get("to"), 10, 64)
	if from == 0 || to == 0 {
		writeErr(w, http.StatusBadRequest, "from va to (unix soniya) majburiy")
		return
	}
	rows, err := s.store.CallsByRange(r.Context(), q.Get("gateway"), from, to)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

// /api/monitoring/bigData?date=YYYY-MM-DD
func (s *Server) handleBigData(w http.ResponseWriter, r *http.Request) {
	dateStr := r.URL.Query().Get("date")
	day := time.Now()
	if dateStr != "" {
		// mahalliy vaqt zonasida (TZ=Asia/Tashkent) parse qilamiz
		d, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "date format: YYYY-MM-DD")
			return
		}
		day = d
	}
	rows, err := s.store.CallsByDay(r.Context(), day)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

// /api/monitoring/operatorTime?from=&to=  yoki  ?date=YYYY-MM-DD
func (s *Server) handleOperatorTime(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var from, to int64
	if d := q.Get("date"); d != "" {
		day, err := time.ParseInLocation("2006-01-02", d, time.Local)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "date format: YYYY-MM-DD")
			return
		}
		from = day.Unix()
		to = from + 86400 - 1
	} else {
		from, _ = strconv.ParseInt(q.Get("from"), 10, 64)
		to, _ = strconv.ParseInt(q.Get("to"), 10, 64)
	}
	if from == 0 || to == 0 {
		writeErr(w, http.StatusBadRequest, "from/to yoki date majburiy")
		return
	}
	stats, err := s.store.OperatorTime(r.Context(), from, to)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// /api/monitoring/users → operator ro'yxati (extension → ism)
func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	users, err := s.pbx.UserGet(ctx)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, users)
}

// /api/monitoring/hidden → yashiringan operator extension'lari (public)
func (s *Server) handleHidden(w http.ResponseWriter, r *http.Request) {
	exts, err := s.store.HiddenExts(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, exts)
}

// /api/monitoring/fifo → jonli navbatlar/operatorlar (OnlinePBX'dan to'g'ridan-to'g'ri)
func (s *Server) handleFifo(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	fifos, err := s.pbx.FifoGet(ctx)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "1", "data": fifos})
}

// GET /api/monitoring/recording?uuid=... — qo'ng'iroq yozuvini (mp3) OnlinePBX'dan
// olib, brauzerga uzatadi (Range qo'llab-quvvatlanadi — audio pleyerда oldinga surish uchun).
func (s *Server) handleRecording(w http.ResponseWriter, r *http.Request) {
	uuid := r.URL.Query().Get("uuid")
	if uuid == "" {
		writeErr(w, http.StatusBadRequest, "uuid majburiy")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	url, err := s.pbx.RecordingURL(ctx, uuid)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "yozuv URL: "+err.Error())
		return
	}
	if url == "" {
		writeErr(w, http.StatusNotFound, "yozuv topilmadi")
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if rng := r.Header.Get("Range"); rng != "" {
		req.Header.Set("Range", rng) // seek/qisman yuklab olishni uzatamiz
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "yozuv yuklab olinmadi: "+err.Error())
		return
	}
	defer resp.Body.Close()

	for _, h := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges"} {
		if v := resp.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "audio/mpeg")
	}
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Cache-Control", "private, max-age=600")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// GET /api/monitoring/liveState — operatorlarning jonli holati (WS bridge'dan).
// {operators: {ext: "online|offline|talking|ringing|dnd"}, connected, version}
func (s *Server) handleLiveState(w http.ResponseWriter, r *http.Request) {
	ops := map[string]string{}
	connected := false
	var version int64
	if s.bridge != nil {
		ops = s.bridge.Snapshot()
		connected = s.bridge.Connected()
		version = s.bridge.Version()
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"operators": ops,
		"connected": connected,
		"version":   version,
	})
}

// ---- helpers ----

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("json yozishda xato: %v", err)
	}
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
