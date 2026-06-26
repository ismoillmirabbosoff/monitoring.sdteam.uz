package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// requireAdmin admin sessiyasini yoki legacy admin parolini tekshiradi.
func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1) Sessiya orqali admin
		if u, ok := s.currentUser(r); ok && u.Role == "admin" {
			next(w, r)
			return
		}
		// 2) Legacy admin parol (bridge)
		pass := r.Header.Get("X-Admin-Password")
		if pass == "" {
			pass = r.URL.Query().Get("admin_password")
		}
		if s.adminPass != "" && pass == s.adminPass {
			next(w, r)
			return
		}
		writeErr(w, http.StatusUnauthorized, "admin huquqi kerak")
	}
}

func (s *Server) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password string `json:"password"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if s.adminPass == "" || body.Password != s.adminPass {
		writeErr(w, http.StatusUnauthorized, "parol noto'g'ri")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// --- Employees ---

func (s *Server) handleListEmployees(w http.ResponseWriter, r *http.Request) {
	emps, err := s.store.ListEmployees(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, emps)
}

func (s *Server) handleAddEmployee(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name    string `json:"name"`
		Company string `json:"company"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		writeErr(w, http.StatusBadRequest, "name majburiy")
		return
	}
	e, err := s.store.AddEmployee(r.Context(), strings.TrimSpace(body.Name), body.Company)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

func (s *Server) handleEmployeeDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id noto'g'ri")
		return
	}
	emp, err := s.store.GetEmployee(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "xodim topilmadi")
		return
	}
	servers, err := s.store.ServersByEmployee(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"employee": emp, "servers": servers})
}

func (s *Server) handleUpdateEmployee(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id noto'g'ri")
		return
	}
	var body struct {
		Hidden *bool `json:"hidden"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "json noto'g'ri")
		return
	}
	if body.Hidden != nil {
		if err := s.store.SetEmployeeHidden(r.Context(), id, *body.Hidden); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	emp, err := s.store.GetEmployee(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "xodim topilmadi")
		return
	}
	writeJSON(w, http.StatusOK, emp)
}

// --- Servers ---

func (s *Server) handleListServers(w http.ResponseWriter, r *http.Request) {
	servers, err := s.store.ListServers(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, servers)
}

func (s *Server) handleAddServer(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name       string `json:"name"`
		Company    string `json:"company"`
		EmployeeID *int   `json:"employee_id"`
		AssignedAt string `json:"assigned_at"` // ixtiyoriy YYYY-MM-DD
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		writeErr(w, http.StatusBadRequest, "name majburiy")
		return
	}
	assignedAt := time.Now()
	if body.AssignedAt != "" {
		if t, err := time.Parse("2006-01-02", body.AssignedAt); err == nil {
			assignedAt = t
		}
	}
	sv, err := s.store.AddServer(r.Context(), strings.TrimSpace(body.Name), body.Company, body.EmployeeID, assignedAt)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, sv)
}

func (s *Server) handleUpdateServer(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id noto'g'ri")
		return
	}
	var body struct {
		EmployeeID *int  `json:"employee_id"`
		Active     *bool `json:"active"`
	}
	// employee_id maydoni umuman berilganini aniqlash uchun xom JSON'ni tekshiramiz
	raw := map[string]json.RawMessage{}
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeErr(w, http.StatusBadRequest, "json noto'g'ri")
		return
	}
	setEmployee := false
	if v, ok := raw["employee_id"]; ok {
		setEmployee = true
		_ = json.Unmarshal(v, &body.EmployeeID) // null bo'lsa nil (biriktirishni olib tashlash)
	}
	if v, ok := raw["active"]; ok {
		_ = json.Unmarshal(v, &body.Active)
	}
	sv, err := s.store.UpdateServer(r.Context(), id, body.EmployeeID, setEmployee, body.Active)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sv)
}

func (s *Server) handleDeleteServer(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id noto'g'ri")
		return
	}
	if err := s.store.DeleteServer(r.Context(), id); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
