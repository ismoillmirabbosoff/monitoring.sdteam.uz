package store

import (
	"context"
	"strconv"
	"time"
)

// ---- Ish vaqti (work_hours) ----

type WorkHour struct {
	Company   string `json:"company"`
	Weekday   int    `json:"weekday"`
	StartHour int    `json:"start_hour"`
	EndHour   int    `json:"end_hour"`
	Active    bool   `json:"active"`
}

func (s *Store) ListWorkHours(ctx context.Context) ([]WorkHour, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT company, weekday, start_hour, end_hour, active
		FROM work_hours ORDER BY company, weekday`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []WorkHour{}
	for rows.Next() {
		var w WorkHour
		if err := rows.Scan(&w.Company, &w.Weekday, &w.StartHour, &w.EndHour, &w.Active); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *Store) UpsertWorkHour(ctx context.Context, w WorkHour) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO work_hours (company, weekday, start_hour, end_hour, active)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (company, weekday) DO UPDATE SET
			start_hour = EXCLUDED.start_hour, end_hour = EXCLUDED.end_hour, active = EXCLUDED.active`,
		w.Company, w.Weekday, w.StartHour, w.EndHour, w.Active)
	return err
}

// ---- Audit log ----

type AuditEntry struct {
	ID        int       `json:"id"`
	UserID    *int      `json:"user_id"`
	UserName  string    `json:"user_name"`
	Action    string    `json:"action"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	IP        string    `json:"ip"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Store) LogAudit(ctx context.Context, userID *int, userName, action, method, path, ip string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO audit_logs (user_id, user_name, action, method, path, ip)
		VALUES ($1,$2,$3,$4,$5,$6)`, userID, userName, action, method, path, ip)
	return err
}

func (s *Store) ListAudit(ctx context.Context, action, q string, limit int) ([]AuditEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	sql := `SELECT id, user_id, user_name, action, method, path, ip, created_at FROM audit_logs WHERE 1=1`
	args := []any{}
	if action != "" {
		args = append(args, action)
		sql += ` AND action = $1`
	}
	if q != "" {
		args = append(args, "%"+q+"%")
		sql += ` AND (user_name ILIKE $` + strconv.Itoa(len(args)) + ` OR path ILIKE $` + strconv.Itoa(len(args)) + `)`
	}
	args = append(args, limit)
	sql += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(len(args))
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AuditEntry{}
	for rows.Next() {
		var a AuditEntry
		if err := rows.Scan(&a.ID, &a.UserID, &a.UserName, &a.Action, &a.Method, &a.Path, &a.IP, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ---- Mijoz baholari (client_feedback) ----

type Feedback struct {
	ID        int       `json:"id"`
	CallUUID  string    `json:"call_uuid"`
	Phone     string    `json:"phone"`
	Score     int       `json:"score"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Store) SaveFeedback(ctx context.Context, callUUID, phone string, score int, comment string) (Feedback, error) {
	var f Feedback
	err := s.pool.QueryRow(ctx, `
		INSERT INTO client_feedback (call_uuid, phone, score, comment)
		VALUES ($1,$2,$3,$4)
		RETURNING id, COALESCE(call_uuid,''), phone, score, comment, created_at`,
		callUUID, phone, score, comment).
		Scan(&f.ID, &f.CallUUID, &f.Phone, &f.Score, &f.Comment, &f.CreatedAt)
	return f, err
}

func (s *Store) ListFeedback(ctx context.Context, from, to int64, minScore int) ([]Feedback, error) {
	sql := `SELECT id, COALESCE(call_uuid,''), phone, score, comment, created_at FROM client_feedback WHERE 1=1`
	args := []any{}
	if from > 0 {
		args = append(args, from)
		sql += ` AND extract(epoch from created_at) >= $` + strconv.Itoa(len(args))
	}
	if to > 0 {
		args = append(args, to)
		sql += ` AND extract(epoch from created_at) <= $` + strconv.Itoa(len(args))
	}
	if minScore > 0 {
		args = append(args, minScore)
		sql += ` AND score = $` + strconv.Itoa(len(args))
	}
	sql += ` ORDER BY created_at DESC LIMIT 500`
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Feedback{}
	for rows.Next() {
		var f Feedback
		if err := rows.Scan(&f.ID, &f.CallUUID, &f.Phone, &f.Score, &f.Comment, &f.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// ---- Operator ballari (scores / avtomatizatsiya) ----

type Score struct {
	ID        int       `json:"id"`
	Ext       string    `json:"ext"`
	Points    int       `json:"points"`
	Reason    string    `json:"reason"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type ScoreTotal struct {
	Ext   string `json:"ext"`
	Total int    `json:"total"`
	Count int    `json:"count"`
}

func (s *Store) AddScore(ctx context.Context, ext string, points int, reason, by string) (Score, error) {
	var sc Score
	err := s.pool.QueryRow(ctx, `
		INSERT INTO operator_scores (ext, points, reason, created_by)
		VALUES ($1,$2,$3,$4)
		RETURNING id, ext, points, reason, created_by, created_at`,
		ext, points, reason, by).
		Scan(&sc.ID, &sc.Ext, &sc.Points, &sc.Reason, &sc.CreatedBy, &sc.CreatedAt)
	return sc, err
}

func (s *Store) ListScores(ctx context.Context, limit int) ([]Score, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, ext, points, reason, created_by, created_at
		FROM operator_scores ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Score{}
	for rows.Next() {
		var sc Score
		if err := rows.Scan(&sc.ID, &sc.Ext, &sc.Points, &sc.Reason, &sc.CreatedBy, &sc.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, sc)
	}
	return out, rows.Err()
}

func (s *Store) ScoreLeaderboard(ctx context.Context) ([]ScoreTotal, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT ext, COALESCE(SUM(points),0) AS total, COUNT(*) AS cnt
		FROM operator_scores GROUP BY ext ORDER BY total DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ScoreTotal{}
	for rows.Next() {
		var t ScoreTotal
		if err := rows.Scan(&t.Ext, &t.Total, &t.Count); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) DeleteScore(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM operator_scores WHERE id = $1`, id)
	return err
}

// AuditActions distinct action ro'yxati (filtr dropdown uchun).
func (s *Store) AuditActions(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT action FROM audit_logs WHERE action <> '' ORDER BY action`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

