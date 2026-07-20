package store

import (
	"context"
	"time"
)

// ---- Employees ----

type Employee struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Ext         string    `json:"ext"`
	Company     string    `json:"company"`
	Source      string    `json:"source"`
	Hidden      bool      `json:"hidden"`
	ServerCount int       `json:"server_count"`
	CreatedAt   time.Time `json:"created_at"`
}

// UpsertOperator OnlinePBX operatorini employees jadvaliga qo'shadi/yangilaydi (ext bo'yicha).
func (s *Store) UpsertOperator(ctx context.Context, name, ext, company string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO employees (name, ext, company, source)
		VALUES ($1, $2, $3, 'operator')
		ON CONFLICT (ext) DO UPDATE SET name = EXCLUDED.name, company = EXCLUDED.company`,
		name, ext, company)
	return err
}

// AddEmployee qo'lda yangi xodim qo'shadi.
func (s *Store) AddEmployee(ctx context.Context, name, company string) (Employee, error) {
	var e Employee
	err := s.pool.QueryRow(ctx, `
		INSERT INTO employees (name, company, source) VALUES ($1, $2, 'manual')
		RETURNING id, name, COALESCE(ext,''), company, source, created_at`,
		name, company).Scan(&e.ID, &e.Name, &e.Ext, &e.Company, &e.Source, &e.CreatedAt)
	return e, err
}

func (s *Store) ListEmployees(ctx context.Context) ([]Employee, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT e.id, e.name, COALESCE(e.ext,''), e.company, e.source, e.hidden, e.created_at,
		       COUNT(sv.id) AS server_count
		FROM employees e
		LEFT JOIN servers sv ON sv.employee_id = e.id
		GROUP BY e.id
		ORDER BY e.source DESC, e.name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Employee{}
	for rows.Next() {
		var e Employee
		if err := rows.Scan(&e.ID, &e.Name, &e.Ext, &e.Company, &e.Source, &e.Hidden, &e.CreatedAt, &e.ServerCount); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) GetEmployee(ctx context.Context, id int) (Employee, error) {
	var e Employee
	err := s.pool.QueryRow(ctx, `
		SELECT e.id, e.name, COALESCE(e.ext,''), e.company, e.source, e.hidden, e.created_at,
		       COUNT(sv.id) AS server_count
		FROM employees e
		LEFT JOIN servers sv ON sv.employee_id = e.id
		WHERE e.id = $1
		GROUP BY e.id`, id).
		Scan(&e.ID, &e.Name, &e.Ext, &e.Company, &e.Source, &e.Hidden, &e.CreatedAt, &e.ServerCount)
	return e, err
}

// SetEmployeeHidden xodimni dashboard/TV'dan yashiradi yoki ko'rsatadi.
func (s *Store) SetEmployeeHidden(ctx context.Context, id int, hidden bool) error {
	_, err := s.pool.Exec(ctx, `UPDATE employees SET hidden = $1 WHERE id = $2`, hidden, id)
	return err
}

// ExtCompanyMap operator extension → kompaniya (employees jadvalidan).
// Kiruvchi qo'ng'iroqlarning gateway'i kompaniyani ko'rsatmaganda fallback sifatida ishlatiladi.
func (s *Store) ExtCompanyMap(ctx context.Context) (map[string]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT ext, company FROM employees
		WHERE ext IS NOT NULL AND ext <> '' AND company <> ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var ext, company string
		if err := rows.Scan(&ext, &company); err != nil {
			return nil, err
		}
		out[ext] = company
	}
	return out, rows.Err()
}

// ServerCountByExt operator extension → biriktirilgan serverlar soni.
func (s *Store) ServerCountByExt(ctx context.Context) (map[string]int, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT e.ext, COUNT(sv.id)
		FROM employees e
		JOIN servers sv ON sv.employee_id = e.id
		WHERE e.ext IS NOT NULL AND e.ext <> ''
		GROUP BY e.ext`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var ext string
		var n int
		if err := rows.Scan(&ext, &n); err != nil {
			return nil, err
		}
		out[ext] = n
	}
	return out, rows.Err()
}

// SetHiddenByExt operatorni extension bo'yicha yashiradi/ko'rsatadi (Hodimlar bo'limidan).
func (s *Store) SetHiddenByExt(ctx context.Context, ext string, hidden bool) error {
	_, err := s.pool.Exec(ctx, `UPDATE employees SET hidden = $1 WHERE ext = $2`, hidden, ext)
	return err
}

// HiddenExts yashiringan operatorlarning extension'lari (public dashboard uchun).
func (s *Store) HiddenExts(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT ext FROM employees WHERE hidden = true AND ext IS NOT NULL AND ext <> ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var ext string
		if err := rows.Scan(&ext); err != nil {
			return nil, err
		}
		out = append(out, ext)
	}
	return out, rows.Err()
}

// ---- Servers ----

type Server struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Company      string    `json:"company"`
	EmployeeID   *int      `json:"employee_id"`
	EmployeeName string    `json:"employee_name"`
	AssignedAt   time.Time `json:"assigned_at"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	// hisoblanadigan maydonlar
	Days   int `json:"days"`   // ish boshlanганидан beri kunlar
	Months int `json:"months"` // to'liq oylar
	Column int `json:"column"` // 1..3 (yosh bo'yicha kalonka)
}

func (s *Store) AddServer(ctx context.Context, name, company string, employeeID *int, assignedAt time.Time) (Server, error) {
	var sv Server
	err := s.pool.QueryRow(ctx, `
		INSERT INTO servers (name, company, employee_id, assigned_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, company, employee_id, assigned_at, active, created_at`,
		name, company, employeeID, assignedAt).
		Scan(&sv.ID, &sv.Name, &sv.Company, &sv.EmployeeID, &sv.AssignedAt, &sv.Active, &sv.CreatedAt)
	if err == nil {
		enrich(&sv)
	}
	return sv, err
}

// UpdateServer xodim/active/nom/kompaniya/sana ni yangilaydi (nil — o'zgarmaydi).
func (s *Store) UpdateServer(ctx context.Context, id int, employeeID *int, setEmployee bool, active *bool, name, company *string, assignedAt *time.Time) (Server, error) {
	if setEmployee {
		if _, err := s.pool.Exec(ctx, `UPDATE servers SET employee_id = $1 WHERE id = $2`, employeeID, id); err != nil {
			return Server{}, err
		}
	}
	if active != nil {
		if _, err := s.pool.Exec(ctx, `UPDATE servers SET active = $1 WHERE id = $2`, *active, id); err != nil {
			return Server{}, err
		}
	}
	if name != nil {
		s.pool.Exec(ctx, `UPDATE servers SET name = $1 WHERE id = $2`, *name, id)
	}
	if company != nil {
		s.pool.Exec(ctx, `UPDATE servers SET company = $1 WHERE id = $2`, *company, id)
	}
	if assignedAt != nil {
		s.pool.Exec(ctx, `UPDATE servers SET assigned_at = $1 WHERE id = $2`, *assignedAt, id)
	}
	var sv Server
	err := s.pool.QueryRow(ctx, `
		SELECT sv.id, sv.name, sv.company, sv.employee_id, COALESCE(e.name,''),
		       sv.assigned_at, sv.active, sv.created_at
		FROM servers sv LEFT JOIN employees e ON e.id = sv.employee_id
		WHERE sv.id = $1`, id).
		Scan(&sv.ID, &sv.Name, &sv.Company, &sv.EmployeeID, &sv.EmployeeName,
			&sv.AssignedAt, &sv.Active, &sv.CreatedAt)
	if err == nil {
		enrich(&sv)
	}
	return sv, err
}

func (s *Store) ListServers(ctx context.Context) ([]Server, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT sv.id, sv.name, sv.company, sv.employee_id,
		       COALESCE(e.name,''), sv.assigned_at, sv.active, sv.created_at
		FROM servers sv
		LEFT JOIN employees e ON e.id = sv.employee_id
		ORDER BY sv.assigned_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Server{}
	for rows.Next() {
		var sv Server
		if err := rows.Scan(&sv.ID, &sv.Name, &sv.Company, &sv.EmployeeID,
			&sv.EmployeeName, &sv.AssignedAt, &sv.Active, &sv.CreatedAt); err != nil {
			return nil, err
		}
		enrich(&sv)
		out = append(out, sv)
	}
	return out, rows.Err()
}

func (s *Store) ServersByEmployee(ctx context.Context, employeeID int) ([]Server, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT sv.id, sv.name, sv.company, sv.employee_id,
		       COALESCE(e.name,''), sv.assigned_at, sv.active, sv.created_at
		FROM servers sv
		LEFT JOIN employees e ON e.id = sv.employee_id
		WHERE sv.employee_id = $1
		ORDER BY sv.assigned_at ASC`, employeeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Server{}
	for rows.Next() {
		var sv Server
		if err := rows.Scan(&sv.ID, &sv.Name, &sv.Company, &sv.EmployeeID,
			&sv.EmployeeName, &sv.AssignedAt, &sv.Active, &sv.CreatedAt); err != nil {
			return nil, err
		}
		enrich(&sv)
		out = append(out, sv)
	}
	return out, rows.Err()
}

func (s *Store) DeleteServer(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM servers WHERE id = $1`, id)
	return err
}

// enrich server yoshini va kalonkasini hisoblaydi.
// 1-oy -> 1-kalonka, 2-oy -> 2-kalonka, 3-oy va undan ko'p -> 3-kalonka.
func enrich(sv *Server) {
	d := time.Since(sv.AssignedAt)
	if d < 0 {
		d = 0
	}
	days := int(d.Hours() / 24)
	sv.Days = days
	sv.Months = days / 30
	col := sv.Months + 1
	if col < 1 {
		col = 1
	}
	if col > 3 {
		col = 3
	}
	sv.Column = col
}
