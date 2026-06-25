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
