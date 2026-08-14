# Tracking Backend (Golang)

Backend untuk Aplikasi Tracking Mobile sesuai PRD v1.0.

## Tech Stack
- **Go 1.21** + `chi` router
- **PostgreSQL 16** (persistent storage)
- **Redis 7** (session, buffer, pub/sub)
- **WebSocket** (`gorilla/websocket`) dengan **MessagePack** binary frame
- **JWT** (access + refresh token, bcrypt)

## Quick Start

### Dengan Docker (recommended)
```bash
cp .env.example .env
# edit .env -> set JWT_SECRET yang kuat
docker-compose up --build
# API: http://localhost:8080
# Health: http://localhost:8080/health
```

### Lokal (tanpa Docker)
```bash
# Pastikan PostgreSQL & Redis berjalan
cp .env.example .env
go mod download
go run ./cmd/server
```

## Endpoints

| Method | Endpoint | Auth | Deskripsi |
|--------|----------|------|-----------|
| POST | `/api/v1/auth/register` | No | Registrasi |
| POST | `/api/v1/auth/login` | No | Login (auto-start tracking session) |
| POST | `/api/v1/auth/refresh` | No | Refresh access token |
| POST | `/api/v1/auth/logout` | No* | Logout (revoke + end session) |
| GET | `/api/v1/users/me` | Yes | Profil |
| PUT | `/api/v1/users/me` | Yes | Update profil |
| GET | `/api/v1/tracking/history?page=1&per_page=20` | Yes | Histori per tanggal |
| GET | `/api/v1/tracking/detail?date=YYYY-MM-DD` | Yes | Detail rute + polyline |
| POST | `/api/v1/tracking/points/batch` | Yes | Fallback batch upload |
| POST | `/api/v1/tracking/session/start` | Yes | Start session eksplisit |
| POST | `/api/v1/tracking/session/end` | Yes | End session eksplisit |
| GET | `/ws/tracking?token={access_token}` | WS | WebSocket tracking |

## WebSocket Protocol

- **Auth**: `wss://host/ws/tracking?token=JWT` atau header `Authorization: Bearer JWT`
- **Binary frame** (MessagePack) untuk `location_update` & `heartbeat`
- **Text frame** (JSON) juga didukung untuk kemudahan debug

### Client -> Server

```json
// location_update (binary MessagePack, distance-filter 10m di client)
{"type":"location_update","session_id":"uuid","lat":-6.2,"lng":106.8,"accuracy":8.5,"speed":12.3,"altitude":45,"timestamp":"2026-08-10T09:15:32Z","v":1}

// heartbeat (tiap 30-60 detik saat diam)
{"type":"heartbeat","session_id":"uuid","timestamp":"2026-08-10T09:16:02Z"}

// session_end
{"type":"session_end","session_id":"uuid"}
```

### Server -> Client
```json
{"type":"connected","session_id":"uuid"}
{"type":"ack","status":"ok","session_id":"uuid"}
```

## Konfigurasi (.env)

Lihat `.env.example` untuk semua variabel. Key penting:
- `JWT_SECRET` - minimal 32 char, ganti di production
- `WS_IDLE_TIMEOUT` - default 15m (sesi idle auto-disconnected)
- `WS_FLUSH_INTERVAL` / `WS_FLUSH_BATCH_SIZE` - tuning buffer Redis -> Postgres

## Arsitektur

```
Mobile --HTTPS--> Go Service --+--> PostgreSQL
       --WSS----> WS Handler --+--> Redis (buffer, session, pub/sub)
                              +--> Worker (flush batch, idle cleanup)
```

- **Single device per user** (koneksi WS baru akan menutup yang lama)
- **Horizontal scaling**: Redis Pub/Sub untuk `force_disconnect` antar instance
- **Mileage**: Haversine incremental per batch flush, filter accuracy >50m & jump >5000m
