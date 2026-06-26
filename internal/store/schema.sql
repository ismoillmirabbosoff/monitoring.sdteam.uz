-- Qo'ng'iroqlar (OnlinePBX history cache'i). uuid bo'yicha upsert qilinadi.
CREATE TABLE IF NOT EXISTS calls (
    uuid                TEXT PRIMARY KEY,
    gateway             TEXT        NOT NULL DEFAULT '',
    accountcode         TEXT        NOT NULL DEFAULT '',
    direction           TEXT        NOT NULL DEFAULT '',
    caller_id_number    TEXT        NOT NULL DEFAULT '',
    caller_id_name      TEXT        NOT NULL DEFAULT '',
    destination_number  TEXT        NOT NULL DEFAULT '',
    start_stamp         BIGINT      NOT NULL DEFAULT 0,
    end_stamp           BIGINT      NOT NULL DEFAULT 0,
    duration            BIGINT      NOT NULL DEFAULT 0,
    user_talk_time      BIGINT      NOT NULL DEFAULT 0,
    hangup_cause        TEXT        NOT NULL DEFAULT '',
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_calls_start_stamp ON calls (start_stamp);
CREATE INDEX IF NOT EXISTS idx_calls_gateway     ON calls (gateway);
CREATE INDEX IF NOT EXISTS idx_calls_direction   ON calls (direction);

-- Sinx holatini saqlash (oxirgi muvaffaqiyatli tortilgan vaqt).
CREATE TABLE IF NOT EXISTS sync_state (
    id              INT PRIMARY KEY DEFAULT 1,
    last_synced_to  BIGINT NOT NULL DEFAULT 0,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO sync_state (id) VALUES (1) ON CONFLICT (id) DO NOTHING;

-- Xodimlar: OnlinePBX operatorlari (source=operator, ext bilan) + admin qo'shganlar (manual).
CREATE TABLE IF NOT EXISTS employees (
    id          SERIAL PRIMARY KEY,
    name        TEXT        NOT NULL,
    ext         TEXT        UNIQUE,            -- operator extension (manual'da NULL)
    company     TEXT        NOT NULL DEFAULT '', -- salesdoc | ibox | ''
    source      TEXT        NOT NULL DEFAULT 'manual',
    hidden      BOOLEAN     NOT NULL DEFAULT false, -- dashboard/TV'dan yashirish
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
ALTER TABLE employees ADD COLUMN IF NOT EXISTS hidden BOOLEAN NOT NULL DEFAULT false;

-- Serverlar/proyektlar: xodimga biriktiriladi, assigned_at'dan yosh hisoblanadi.
CREATE TABLE IF NOT EXISTS servers (
    id           SERIAL PRIMARY KEY,
    name         TEXT        NOT NULL,
    company      TEXT        NOT NULL DEFAULT '', -- salesdoc | ibox | ''
    employee_id  INTEGER     REFERENCES employees(id) ON DELETE SET NULL,
    assigned_at  TIMESTAMPTZ NOT NULL DEFAULT now(),  -- ish boshlangan vaqt (yosh shu'dan)
    active       BOOLEAN     NOT NULL DEFAULT true,    -- faol / nofaol
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- mavjud jadvalga active ustunini qo'shish (idempotent)
ALTER TABLE servers ADD COLUMN IF NOT EXISTS active BOOLEAN NOT NULL DEFAULT true;
CREATE INDEX IF NOT EXISTS idx_servers_employee ON servers (employee_id);

-- Foydalanuvchilar (admin / operator) — email + parol bilan login.
CREATE TABLE IF NOT EXISTS users (
    id            SERIAL PRIMARY KEY,
    email         TEXT        UNIQUE NOT NULL,
    password_hash TEXT        NOT NULL,
    name          TEXT        NOT NULL DEFAULT '',
    role          TEXT        NOT NULL DEFAULT 'operator', -- admin | operator
    ext           TEXT,                                    -- operator extension (ixtiyoriy)
    active        BOOLEAN     NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Email tasdiqlash kodlari (login 2-bosqich).
CREATE TABLE IF NOT EXISTS login_codes (
    id         SERIAL PRIMARY KEY,
    user_id    INTEGER     NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code       TEXT        NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_login_codes_user ON login_codes (user_id);

-- Sessiyalar (logout yo'q — faqat admin bekor qiladi).
CREATE TABLE IF NOT EXISTS sessions (
    token      TEXT        PRIMARY KEY,
    user_id    INTEGER     NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_agent TEXT        NOT NULL DEFAULT '',
    ip         TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions (user_id);

-- Anketa savollari (admin sozlaydi).
CREATE TABLE IF NOT EXISTS survey_questions (
    id         SERIAL PRIMARY KEY,
    label      TEXT        NOT NULL,
    type       TEXT        NOT NULL DEFAULT 'text', -- text | choice | rating | yesno
    options    JSONB       NOT NULL DEFAULT '[]',
    required   BOOLEAN     NOT NULL DEFAULT false,
    position   INTEGER     NOT NULL DEFAULT 0,
    active     BOOLEAN     NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- To'ldirilgan anketalar (har qo'ng'iroqqa bittadan).
CREATE TABLE IF NOT EXISTS survey_responses (
    id           SERIAL PRIMARY KEY,
    call_uuid    TEXT        UNIQUE NOT NULL,
    operator_ext TEXT        NOT NULL DEFAULT '',
    user_id      INTEGER     REFERENCES users(id) ON DELETE SET NULL,
    answers      JSONB       NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_survey_resp_created ON survey_responses (created_at);
