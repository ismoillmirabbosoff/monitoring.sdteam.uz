package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/salesdoc/monitoring-api/internal/store"
)

// ---- Ish vaqti (work_hours) ----

// GET /api/admin/work-hours
func (s *Server) handleListWorkHours(w http.ResponseWriter, r *http.Request) {
	wh, err := s.store.ListWorkHours(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, wh)
}

// POST /api/admin/work-hours — massiv: [{company,weekday,start_hour,end_hour,active}]
func (s *Server) handleSaveWorkHours(w http.ResponseWriter, r *http.Request) {
	var body []store.WorkHour
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "json noto'g'ri")
		return
	}
	for _, wh := range body {
		if wh.Weekday < 0 || wh.Weekday > 6 {
			continue
		}
		if err := s.store.UpsertWorkHour(r.Context(), wh); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// ---- Audit log ----

// GET /api/admin/audit-log?action=&q=&limit=
func (s *Server) handleListAudit(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	entries, err := s.store.ListAudit(r.Context(), q.Get("action"), q.Get("q"), limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	actions, _ := s.store.AuditActions(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"entries": entries, "actions": actions})
}

// ---- Mijoz baholari (otzyvlar) ----

// POST /api/feedback (public) — {call_uuid, phone, score, comment}
func (s *Server) handleSubmitFeedback(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CallUUID string `json:"call_uuid"`
		Phone    string `json:"phone"`
		Score    int    `json:"score"`
		Comment  string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "json noto'g'ri")
		return
	}
	if body.Score < 1 || body.Score > 5 {
		writeErr(w, http.StatusBadRequest, "score 1-5 bo'lishi kerak")
		return
	}
	f, err := s.store.SaveFeedback(r.Context(), body.CallUUID, strings.TrimSpace(body.Phone), body.Score, strings.TrimSpace(body.Comment))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, f)
}

// GET /api/admin/feedback?from=&to=&score=
func (s *Server) handleListFeedback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, _ := strconv.ParseInt(q.Get("from"), 10, 64)
	to, _ := strconv.ParseInt(q.Get("to"), 10, 64)
	score, _ := strconv.Atoi(q.Get("score"))
	list, err := s.store.ListFeedback(r.Context(), from, to, score)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// ---- Operator ballari (scores / avtomatizatsiya) ----

// GET /api/admin/scores → {leaderboard, recent}
func (s *Server) handleListScores(w http.ResponseWriter, r *http.Request) {
	lb, err := s.store.ScoreLeaderboard(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	recent, _ := s.store.ListScores(r.Context(), 100)
	writeJSON(w, http.StatusOK, map[string]any{"leaderboard": lb, "recent": recent})
}

// POST /api/admin/scores — {ext, points, reason}
func (s *Server) handleAddScore(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Ext    string `json:"ext"`
		Points int    `json:"points"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Ext) == "" {
		writeErr(w, http.StatusBadRequest, "ext majburiy")
		return
	}
	by := "admin"
	if u, ok := s.currentUser(r); ok {
		if u.Name != "" {
			by = u.Name
		} else {
			by = u.Email
		}
	}
	sc, err := s.store.AddScore(r.Context(), strings.TrimSpace(body.Ext), body.Points, strings.TrimSpace(body.Reason), by)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, sc)
}

// DELETE /api/admin/scores/{id}
func (s *Server) handleDeleteScore(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id noto'g'ri")
		return
	}
	if err := s.store.DeleteScore(r.Context(), id); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// ---- Audit logging helper (requireAdmin ichida chaqiriladi) ----

// auditMutation o'zgartiruvchi (POST/PATCH/PUT/DELETE) admin so'rovini yozadi.
func (s *Server) auditMutation(r *http.Request, uid *int, name string) {
	if r.Method == http.MethodGet || r.Method == http.MethodOptions {
		return
	}
	// audit-log endpointining o'zini yozmaymiz
	if strings.HasPrefix(r.URL.Path, "/api/admin/audit-log") {
		return
	}
	s.store.LogAudit(r.Context(), uid, name, auditAction(r.Method, r.URL.Path), r.Method, r.URL.Path, clientIP(r))
}

// auditAction method+path'dan o'qiladigan amal nomini yasaydi (masalan servers_create).
func auditAction(method, path string) string {
	p := strings.TrimPrefix(path, "/api/admin/")
	base := p
	if i := strings.IndexByte(p, '/'); i >= 0 {
		base = p[:i]
	}
	verb := map[string]string{"POST": "create", "PATCH": "update", "PUT": "update", "DELETE": "delete"}[method]
	if verb == "" {
		verb = strings.ToLower(method)
	}
	if base == "" {
		return verb
	}
	return base + "_" + verb
}
