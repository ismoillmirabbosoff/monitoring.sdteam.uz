package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/salesdoc/monitoring-api/internal/store"
)

// ---- yordamchi funksiyalar ----

func randomToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func randomCode() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	return strconv.Itoa(int(n.Int64()+1000000))[1:] // 6 xonali, oldida nol bo'lishi mumkin
}

func sessionToken(r *http.Request) string {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return r.Header.Get("X-Session")
}

func clientIP(r *http.Request) string {
	if f := r.Header.Get("X-Forwarded-For"); f != "" {
		return strings.TrimSpace(strings.Split(f, ",")[0])
	}
	return r.RemoteAddr
}

// currentUser sessiya bo'yicha foydalanuvchini qaytaradi (yo'q bo'lsa ok=false).
func (s *Server) currentUser(r *http.Request) (store.User, bool) {
	tok := sessionToken(r)
	if tok == "" {
		return store.User{}, false
	}
	u, err := s.store.SessionUser(r.Context(), tok)
	if err != nil || !u.Active {
		return store.User{}, false
	}
	return u, true
}

// ---- Login oqimi ----

// POST /api/auth/login {email, password} → kod yuboradi
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct{ Email, Password string }
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "email va parol kerak")
		return
	}
	u, err := s.store.UserByEmail(r.Context(), strings.TrimSpace(body.Email))
	if err != nil || !u.Active {
		writeErr(w, http.StatusUnauthorized, "email yoki parol noto'g'ri")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(body.Password)) != nil {
		writeErr(w, http.StatusUnauthorized, "email yoki parol noto'g'ri")
		return
	}
	code := randomCode()
	if err := s.store.CreateLoginCode(r.Context(), u.ID, code, 10*time.Minute); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.email.SendCode(u.Email, code); err != nil {
		writeErr(w, http.StatusInternalServerError, "email yuborilmadi: "+err.Error())
		return
	}
	resp := map[string]any{"pending": true, "email": u.Email}
	if !s.email.Configured() {
		resp["dev_code"] = code // SMTP yo'q — test uchun kodni qaytaramiz
	}
	writeJSON(w, http.StatusOK, resp)
}

// POST /api/auth/verify {email, code} → sessiya yaratadi
func (s *Server) handleVerify(w http.ResponseWriter, r *http.Request) {
	var body struct{ Email, Code string }
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "email va kod kerak")
		return
	}
	u, err := s.store.UserByEmail(r.Context(), strings.TrimSpace(body.Email))
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "noto'g'ri")
		return
	}
	ok, err := s.store.CheckLoginCode(r.Context(), u.ID, strings.TrimSpace(body.Code))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		writeErr(w, http.StatusUnauthorized, "kod noto'g'ri yoki muddati o'tgan")
		return
	}
	token := randomToken()
	if err := s.store.CreateSession(r.Context(), token, u.ID, r.UserAgent(), clientIP(r)); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "user": u})
}

// GET /api/auth/me → joriy foydalanuvchi
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	u, ok := s.currentUser(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "sessiya yo'q")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

// POST /api/auth/logout → joriy sessiyani o'chiradi
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if tok := sessionToken(r); tok != "" {
		s.store.DeleteSession(r.Context(), tok)
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// ---- Admin: foydalanuvchilar ----

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.store.ListUsers(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email, Password, Name, Role, Ext string
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Email == "" || body.Password == "" {
		writeErr(w, http.StatusBadRequest, "email va parol majburiy")
		return
	}
	if body.Role != "admin" {
		body.Role = "operator"
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	u, err := s.store.CreateUser(r.Context(), strings.TrimSpace(body.Email), string(hash),
		strings.TrimSpace(body.Name), body.Role, strings.TrimSpace(body.Ext))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "yaratib bo'lmadi (email band bo'lishi mumkin)")
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id noto'g'ri")
		return
	}
	var body struct {
		Name, Role, Ext, Password *string
		Active                    *bool
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "json noto'g'ri")
		return
	}
	var hash *string
	if body.Password != nil && *body.Password != "" {
		h, _ := bcrypt.GenerateFromPassword([]byte(*body.Password), bcrypt.DefaultCost)
		hs := string(h)
		hash = &hs
		// parol o'zgarsa — barcha sessiyalarni bekor qilamiz
		s.store.DeleteUserSessions(r.Context(), id)
	}
	if body.Role != nil && *body.Role != "admin" {
		op := "operator"
		body.Role = &op
	}
	u, err := s.store.UpdateUser(r.Context(), id, body.Name, body.Role, body.Ext, body.Active, hash)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id noto'g'ri")
		return
	}
	if err := s.store.DeleteUser(r.Context(), id); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// ---- Admin: sessiyalar ----

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := s.store.ListSessions(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sessions)
}

func (s *Server) handleRevokeSession(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if err := s.store.DeleteSession(r.Context(), token); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
