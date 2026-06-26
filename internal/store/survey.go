package store

import (
	"context"
	"encoding/json"
	"time"
)

// ---- Anketa savollari ----

type Question struct {
	ID        int             `json:"id"`
	Label     string          `json:"label"`
	Type      string          `json:"type"` // text | choice | rating | yesno
	Options   json.RawMessage `json:"options"`
	Required  bool            `json:"required"`
	Position  int             `json:"position"`
	Active    bool            `json:"active"`
	CreatedAt time.Time       `json:"created_at"`
}

func (s *Store) ListQuestions(ctx context.Context, activeOnly bool) ([]Question, error) {
	q := `SELECT id, label, type, options, required, position, active, created_at FROM survey_questions`
	if activeOnly {
		q += ` WHERE active = true`
	}
	q += ` ORDER BY position ASC, id ASC`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Question{}
	for rows.Next() {
		var x Question
		if err := rows.Scan(&x.ID, &x.Label, &x.Type, &x.Options, &x.Required, &x.Position, &x.Active, &x.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (s *Store) CreateQuestion(ctx context.Context, label, typ string, options json.RawMessage, required bool, position int) (Question, error) {
	if len(options) == 0 {
		options = json.RawMessage("[]")
	}
	var x Question
	err := s.pool.QueryRow(ctx, `
		INSERT INTO survey_questions (label, type, options, required, position)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id, label, type, options, required, position, active, created_at`,
		label, typ, options, required, position).
		Scan(&x.ID, &x.Label, &x.Type, &x.Options, &x.Required, &x.Position, &x.Active, &x.CreatedAt)
	return x, err
}

func (s *Store) UpdateQuestion(ctx context.Context, id int, label, typ *string, options json.RawMessage, required, active *bool, position *int) (Question, error) {
	if label != nil {
		s.pool.Exec(ctx, `UPDATE survey_questions SET label=$1 WHERE id=$2`, *label, id)
	}
	if typ != nil {
		s.pool.Exec(ctx, `UPDATE survey_questions SET type=$1 WHERE id=$2`, *typ, id)
	}
	if options != nil {
		s.pool.Exec(ctx, `UPDATE survey_questions SET options=$1 WHERE id=$2`, options, id)
	}
	if required != nil {
		s.pool.Exec(ctx, `UPDATE survey_questions SET required=$1 WHERE id=$2`, *required, id)
	}
	if active != nil {
		s.pool.Exec(ctx, `UPDATE survey_questions SET active=$1 WHERE id=$2`, *active, id)
	}
	if position != nil {
		s.pool.Exec(ctx, `UPDATE survey_questions SET position=$1 WHERE id=$2`, *position, id)
	}
	var x Question
	err := s.pool.QueryRow(ctx, `SELECT id, label, type, options, required, position, active, created_at FROM survey_questions WHERE id=$1`, id).
		Scan(&x.ID, &x.Label, &x.Type, &x.Options, &x.Required, &x.Position, &x.Active, &x.CreatedAt)
	return x, err
}

func (s *Store) DeleteQuestion(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM survey_questions WHERE id=$1`, id)
	return err
}

// ---- Anketa javoblari ----

type Response struct {
	ID          int             `json:"id"`
	CallUUID    string          `json:"call_uuid"`
	OperatorExt string          `json:"operator_ext"`
	UserID      *int            `json:"user_id"`
	Answers     json.RawMessage `json:"answers"`
	CreatedAt   time.Time       `json:"created_at"`
}

// SaveResponse anketani saqlaydi (call_uuid bo'yicha upsert).
func (s *Store) SaveResponse(ctx context.Context, callUUID, operatorExt string, userID *int, answers json.RawMessage) (Response, error) {
	if len(answers) == 0 {
		answers = json.RawMessage("{}")
	}
	var r Response
	err := s.pool.QueryRow(ctx, `
		INSERT INTO survey_responses (call_uuid, operator_ext, user_id, answers)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (call_uuid) DO UPDATE SET operator_ext=EXCLUDED.operator_ext,
			user_id=EXCLUDED.user_id, answers=EXCLUDED.answers, created_at=now()
		RETURNING id, call_uuid, operator_ext, user_id, answers, created_at`,
		callUUID, operatorExt, userID, answers).
		Scan(&r.ID, &r.CallUUID, &r.OperatorExt, &r.UserID, &r.Answers, &r.CreatedAt)
	return r, err
}

// ResponsesInRange berilgan vaqt oralig'idagi javoblarni qaytaradi.
func (s *Store) ResponsesInRange(ctx context.Context, from, to int64) ([]Response, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, call_uuid, operator_ext, user_id, answers, created_at
		FROM survey_responses
		WHERE extract(epoch from created_at) >= $1 AND extract(epoch from created_at) <= $2
		ORDER BY created_at DESC`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Response{}
	for rows.Next() {
		var r Response
		if err := rows.Scan(&r.ID, &r.CallUUID, &r.OperatorExt, &r.UserID, &r.Answers, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ResponseByCall bitta qo'ng'iroq anketasini qaytaradi (yo'q bo'lsa ErrNotFound).
func (s *Store) ResponseByCall(ctx context.Context, callUUID string) (Response, error) {
	var r Response
	err := s.pool.QueryRow(ctx, `
		SELECT id, call_uuid, operator_ext, user_id, answers, created_at
		FROM survey_responses WHERE call_uuid = $1`, callUUID).
		Scan(&r.ID, &r.CallUUID, &r.OperatorExt, &r.UserID, &r.Answers, &r.CreatedAt)
	return r, err
}
