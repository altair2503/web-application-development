CREATE TABLE IF NOT EXISTS wanted_criminals (
  id            SERIAL PRIMARY KEY,
  name          TEXT,                                   -- real name, may be unknown
  nickname      TEXT NOT NULL,
  reward        NUMERIC(14,2) NOT NULL CHECK (reward >= 0),
  latitude      DOUBLE PRECISION CHECK (latitude  BETWEEN -90  AND 90),
  longitude     DOUBLE PRECISION CHECK (longitude BETWEEN -180 AND 180),
  location_note TEXT,                                   -- text alternative to coordinates
  difficulty    SMALLINT NOT NULL CHECK (difficulty BETWEEN 1 AND 10),
  armed         BOOLEAN NOT NULL DEFAULT FALSE,
  has_crew      BOOLEAN NOT NULL DEFAULT FALSE,
  escape_master BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS hunt_logs (
  id          SERIAL PRIMARY KEY,
  criminal_id INTEGER NOT NULL REFERENCES wanted_criminals(id) ON DELETE CASCADE,
  success     BOOLEAN NOT NULL,
  notes       TEXT NOT NULL DEFAULT '',
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_hunt_logs_criminal ON hunt_logs(criminal_id);
