-- Enable pgcrypto for gen_random_uuid
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- users
CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(255) NOT NULL,
  email VARCHAR(255) UNIQUE NOT NULL,
  phone VARCHAR(30),
  password_hash TEXT NOT NULL,
  avatar_url TEXT,
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
);

-- refresh_tokens
CREATE TABLE IF NOT EXISTS refresh_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked BOOLEAN DEFAULT false,
  created_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_hash ON refresh_tokens(token_hash);

-- tracking_sessions
CREATE TABLE IF NOT EXISTS tracking_sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  started_at TIMESTAMPTZ NOT NULL,
  ended_at TIMESTAMPTZ,
  status VARCHAR(20) NOT NULL DEFAULT 'active',
  total_distance_m DOUBLE PRECISION DEFAULT 0,
  session_date DATE NOT NULL,
  created_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_sessions_user_date ON tracking_sessions(user_id, session_date);
CREATE INDEX IF NOT EXISTS idx_sessions_user_status ON tracking_sessions(user_id, status);
CREATE INDEX IF NOT EXISTS idx_sessions_status ON tracking_sessions(status);

-- tracking_points
CREATE TABLE IF NOT EXISTS tracking_points (
  id BIGSERIAL PRIMARY KEY,
  session_id UUID REFERENCES tracking_sessions(id) ON DELETE CASCADE,
  lat DOUBLE PRECISION NOT NULL,
  lng DOUBLE PRECISION NOT NULL,
  accuracy DOUBLE PRECISION,
  speed DOUBLE PRECISION,
  altitude DOUBLE PRECISION,
  recorded_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_points_session_time ON tracking_points(session_id, recorded_at);
CREATE INDEX IF NOT EXISTS idx_points_recorded_at ON tracking_points(recorded_at);

-- daily_mileage_summary
CREATE TABLE IF NOT EXISTS daily_mileage_summary (
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  summary_date DATE NOT NULL,
  total_distance_m DOUBLE PRECISION DEFAULT 0,
  total_duration_sec INT DEFAULT 0,
  session_count INT DEFAULT 0,
  PRIMARY KEY (user_id, summary_date)
);
