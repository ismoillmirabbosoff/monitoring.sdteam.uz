// Package store Postgres orqali qo'ng'iroqlar cache'ini boshqaradi.
package store

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salesdoc/monitoring-api/internal/onlinepbx"
)

//go:embed schema.sql
var schemaSQL string

type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("pgxpool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("db ping: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, schemaSQL)
	return err
}

func (s *Store) Close() { s.pool.Close() }

// UpsertCalls qo'ng'iroqlarni uuid bo'yicha yangilab/qo'shib qo'yadi.
func (s *Store) UpsertCalls(ctx context.Context, calls []onlinepbx.Call) (int, error) {
	if len(calls) == 0 {
		return 0, nil
	}
	batch := &pgx.Batch{}
	for _, c := range calls {
		batch.Queue(`
			INSERT INTO calls (uuid, gateway, accountcode, direction, caller_id_number,
				caller_id_name, destination_number, start_stamp, end_stamp, duration,
				user_talk_time, hangup_cause, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12, now())
			ON CONFLICT (uuid) DO UPDATE SET
				gateway=EXCLUDED.gateway, accountcode=EXCLUDED.accountcode,
				direction=EXCLUDED.direction, caller_id_number=EXCLUDED.caller_id_number,
				caller_id_name=EXCLUDED.caller_id_name,
				destination_number=EXCLUDED.destination_number,
				start_stamp=EXCLUDED.start_stamp, end_stamp=EXCLUDED.end_stamp,
				duration=EXCLUDED.duration, user_talk_time=EXCLUDED.user_talk_time,
				hangup_cause=EXCLUDED.hangup_cause, updated_at=now()`,
			c.UUID, c.Gateway, c.Accountcode, c.Direction, c.CallerIDNumber,
			c.CallerIDName, c.DestinationNumber, c.StartStamp, c.EndStamp,
			c.Duration, c.UserTalkTime, c.HangupCause)
	}
	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()
	for range calls {
		if _, err := br.Exec(); err != nil {
			return 0, err
		}
	}
	return len(calls), nil
}

// CallRow frontend kutgan JSON formatidagi yozuv.
type CallRow struct {
	Gateway           string `json:"gateway"`
	Accountcode       string `json:"accountcode"`
	Direction         string `json:"direction"`
	StartStamp        int64  `json:"start_stamp"`
	EndStamp          int64  `json:"end_stamp"`
	UserTalkTime      int64  `json:"user_talk_time"`
	Duration          int64  `json:"duration"`
	CallerIDNumber    string `json:"caller_id_number"`
	DestinationNumber string `json:"destination_number"`
	UUID              string `json:"uuid"`
}

const selectCols = `gateway, accountcode, direction, start_stamp, end_stamp,
	user_talk_time, duration, caller_id_number, destination_number, uuid`

func scanCalls(rows pgx.Rows) ([]CallRow, error) {
	defer rows.Close()
	out := []CallRow{}
	for rows.Next() {
		var r CallRow
		if err := rows.Scan(&r.Gateway, &r.Accountcode, &r.Direction, &r.StartStamp,
			&r.EndStamp, &r.UserTalkTime, &r.Duration, &r.CallerIDNumber,
			&r.DestinationNumber, &r.UUID); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// CallsByRange gateway + vaqt oralig'i bo'yicha (frontend /monitoring/data).
func (s *Store) CallsByRange(ctx context.Context, gateway string, from, to int64) ([]CallRow, error) {
	q := `SELECT ` + selectCols + ` FROM calls
	      WHERE start_stamp >= $1 AND start_stamp <= $2`
	args := []any{from, to}
	if gateway != "" {
		q += ` AND gateway = $3`
		args = append(args, gateway)
	}
	q += ` ORDER BY start_stamp ASC`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	return scanCalls(rows)
}

// CallsByDay bir kunlik barcha qo'ng'iroqlar (frontend /monitoring/bigData).
func (s *Store) CallsByDay(ctx context.Context, day time.Time) ([]CallRow, error) {
	from := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location()).Unix()
	to := from + 86400 - 1
	return s.CallsByRange(ctx, "", from, to)
}

// OperatorStat operator (gateway) bo'yicha kunlik yig'indi.
type OperatorStat struct {
	Gateway       string `json:"gateway"`
	IncomingCalls int64  `json:"incoming_calls"`
	OutgoingCalls int64  `json:"outgoing_calls"`
	TotalTalkTime int64  `json:"total_talk_time"` // soniya
}

// OperatorTime berilgan oraliqda har gateway uchun jamlanma (frontend /monitoring/operatorTime).
func (s *Store) OperatorTime(ctx context.Context, from, to int64) ([]OperatorStat, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT gateway,
		       COUNT(*) FILTER (WHERE direction = 'inbound')  AS incoming,
		       COUNT(*) FILTER (WHERE direction = 'outbound') AS outgoing,
		       COALESCE(SUM(user_talk_time), 0)               AS talk
		FROM calls
		WHERE start_stamp >= $1 AND start_stamp <= $2
		GROUP BY gateway
		ORDER BY gateway`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []OperatorStat{}
	for rows.Next() {
		var st OperatorStat
		if err := rows.Scan(&st.Gateway, &st.IncomingCalls, &st.OutgoingCalls, &st.TotalTalkTime); err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

func (s *Store) GetLastSyncedTo(ctx context.Context) (int64, error) {
	var v int64
	err := s.pool.QueryRow(ctx, `SELECT last_synced_to FROM sync_state WHERE id = 1`).Scan(&v)
	return v, err
}

func (s *Store) SetLastSyncedTo(ctx context.Context, ts int64) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE sync_state SET last_synced_to = $1, updated_at = now() WHERE id = 1`, ts)
	return err
}
