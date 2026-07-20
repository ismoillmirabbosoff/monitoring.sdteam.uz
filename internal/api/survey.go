package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/salesdoc/monitoring-api/internal/store"
)

// requireAuth har qanday tizimga kirgan foydalanuvchini talab qiladi.
func (s *Server) requireAuth(next func(http.ResponseWriter, *http.Request, store.User)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := s.currentUser(r)
		if !ok {
			writeErr(w, http.StatusUnauthorized, "kirish kerak")
			return
		}
		next(w, r, u)
	}
}

// ---- Savollar ----

// GET /api/survey/questions — faol savollar (to'ldirish uchun, har qanday foydalanuvchi)
func (s *Server) handleActiveQuestions(w http.ResponseWriter, r *http.Request, _ store.User) {
	qs, err := s.store.ListQuestions(r.Context(), true)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, qs)
}

// GET /api/admin/survey/questions — barcha savollar
func (s *Server) handleListQuestions(w http.ResponseWriter, r *http.Request) {
	qs, err := s.store.ListQuestions(r.Context(), false)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, qs)
}

func (s *Server) handleCreateQuestion(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Label    string          `json:"label"`
		Type     string          `json:"type"`
		Options  json.RawMessage `json:"options"`
		Required bool            `json:"required"`
		Position int             `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Label) == "" {
		writeErr(w, http.StatusBadRequest, "savol matni majburiy")
		return
	}
	if body.Type == "" {
		body.Type = "text"
	}
	q, err := s.store.CreateQuestion(r.Context(), strings.TrimSpace(body.Label), body.Type, body.Options, body.Required, body.Position)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, q)
}

func (s *Server) handleUpdateQuestion(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id noto'g'ri")
		return
	}
	var body struct {
		Label    *string         `json:"label"`
		Type     *string         `json:"type"`
		Options  json.RawMessage `json:"options"`
		Required *bool           `json:"required"`
		Active   *bool           `json:"active"`
		Position *int            `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "json noto'g'ri")
		return
	}
	q, err := s.store.UpdateQuestion(r.Context(), id, body.Label, body.Type, body.Options, body.Required, body.Active, body.Position)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, q)
}

func (s *Server) handleDeleteQuestion(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id noto'g'ri")
		return
	}
	if err := s.store.DeleteQuestion(r.Context(), id); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// ---- Anketa konfiguratsiyasi ----

// GET /api/survey/config — to'ldirish formasi uchun (kirgan foydalanuvchi)
func (s *Server) handleActiveSurveyConfig(w http.ResponseWriter, r *http.Request, _ store.User) {
	cfg, err := s.store.GetSurveyConfig(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(cfg)
}

// GET /api/admin/survey/config — sozlash uchun (admin)
func (s *Server) handleGetSurveyConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.store.GetSurveyConfig(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(cfg)
}

// PUT /api/admin/survey/config — konfiguratsiyani saqlash (admin)
func (s *Server) handleSaveSurveyConfig(w http.ResponseWriter, r *http.Request) {
	var cfg json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeErr(w, http.StatusBadRequest, "json noto'g'ri")
		return
	}
	if err := s.store.SaveSurveyConfig(r.Context(), cfg); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// ---- Javoblar ----

// POST /api/survey/responses — anketani saqlash (kirgan foydalanuvchi)
func (s *Server) handleSaveResponse(w http.ResponseWriter, r *http.Request, u store.User) {
	var body struct {
		CallUUID    string          `json:"call_uuid"`
		OperatorExt string          `json:"operator_ext"`
		Answers     json.RawMessage `json:"answers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.CallUUID == "" {
		writeErr(w, http.StatusBadRequest, "call_uuid majburiy")
		return
	}
	ext := body.OperatorExt
	if ext == "" {
		ext = u.Ext
	}
	resp, err := s.store.SaveResponse(r.Context(), body.CallUUID, ext, &u.ID, body.Answers)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// GET /api/survey/responses?from=&to= — oraliqdagi javoblar (kirgan foydalanuvchi)
func (s *Server) handleListResponses(w http.ResponseWriter, r *http.Request, _ store.User) {
	from, _ := strconv.ParseInt(r.URL.Query().Get("from"), 10, 64)
	to, _ := strconv.ParseInt(r.URL.Query().Get("to"), 10, 64)
	if from == 0 || to == 0 {
		writeErr(w, http.StatusBadRequest, "from va to kerak")
		return
	}
	resp, err := s.store.ResponsesInRange(r.Context(), from, to)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
