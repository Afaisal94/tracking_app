# Product Requirements Document (PRD)
## Aplikasi Tracking Mobile (React Native) + Backend (Golang, WebSocket, Redis, PostgreSQL)

**Versi:** 1.0
**Tanggal:** 10 Agustus 2026
**Status:** Draft

---

## 1. Latar Belakang & Tujuan

### 1.1 Latar Belakang
Perusahaan membutuhkan aplikasi mobile yang dapat melacak (tracking) lokasi/perjalanan user secara real-time selama user dalam status login. Tracking harus tetap berjalan meskipun aplikasi berada di background (minimized) maupun foreground, dan otomatis berhenti saat user logout.

### 1.2 Tujuan Produk
- Menyediakan aplikasi mobile yang dapat merekam jejak perjalanan user secara akurat dan efisien (hemat baterai & kuota).
- Menyediakan histori perjalanan per tanggal beserta total jarak tempuh (mileage).
- Menyediakan visualisasi rute perjalanan pada peta untuk setiap sesi tracking.
- Backend yang scalable, real-time, dan reliable untuk menerima, menyimpan, dan menyajikan data lokasi dalam jumlah besar.

### 1.3 Ruang Lingkup (Scope)
**In-scope:**
- Autentikasi user (login/register)
- Tracking lokasi otomatis (start saat login, stop saat logout)
- Tracking di foreground & background
- Riwayat tracking (list per tanggal + total mileage)
- Detail tracking per tanggal (peta rute)
- Backend service (REST API + WebSocket) dengan Redis & PostgreSQL

**Out-of-scope (v1):**
- Geofencing / notifikasi berbasis lokasi
- Multi-user role/admin dashboard (dapat menjadi fase berikutnya)
- Sharing lokasi real-time antar user
- Tracking mode offline penuh dengan sync (opsional, lihat catatan di §4.6)

---

## 2. User Persona

| Persona | Deskripsi | Kebutuhan Utama |
|---|---|---|
| End User | Individu yang perjalanannya perlu dilacak (misal: kurir, sales lapangan, driver) | Tracking otomatis tanpa perlu aksi manual, melihat histori perjalanan |
| Admin/Operator (future) | Melihat data tracking banyak user | Dashboard monitoring (di luar scope v1) |

---

## 3. PRD — Mobile Application (React Native)

### 3.1 Daftar Screen

1. **Splash Screen**
2. **Login**
3. **Register**
4. **Home** (berisi profil user)
5. **Tracking History** (daftar tanggal + total mileage)
6. **Detail Tracking** (peta rute berdasarkan tanggal)

### 3.2 Functional Requirements per Screen

#### 3.2.1 Splash Screen
- Menampilkan logo aplikasi selama proses inisialisasi (± 1.5–2 detik).
- Mengecek token/session yang tersimpan secara lokal (secure storage).
  - Jika token valid → redirect ke **Home** dan **auto-resume tracking service** jika sebelumnya aktif.
  - Jika token tidak ada/expired → redirect ke **Login**.
- Melakukan pengecekan permission dasar (lokasi, notifikasi) di awal (tanpa memaksa request di splash, cukup pengecekan status).

#### 3.2.2 Login
- Input: email/username, password.
- Validasi input di client-side (format email, panjang password).
- Memanggil API `POST /api/v1/auth/login`.
- Menyimpan `access_token` & `refresh_token` di secure storage (Keychain/Keystore, misal `react-native-keychain`).
- Setelah login sukses:
  - Request permission lokasi (foreground & background) jika belum diberikan.
  - **Trigger start tracking service** (lihat §3.4).
  - Redirect ke Home.
- Menampilkan pesan error yang jelas (kredensial salah, koneksi gagal, dsb).
- Link ke halaman Register.
- (Opsional v1.1) "Forgot Password".

#### 3.2.3 Register
- Input: nama lengkap, email, no. HP, password, konfirmasi password.
- Validasi client-side (email valid, password match, kekuatan password minimal).
- Memanggil API `POST /api/v1/auth/register`.
- Setelah sukses, arahkan ke Login (atau auto-login sesuai keputusan bisnis).
- Menampilkan error (email sudah terdaftar, dsb).

#### 3.2.4 Home
- Menampilkan profil user: foto (jika ada), nama, email, no. HP.
- Menampilkan status tracking saat ini (aktif/tidak, indikator visual — misal badge hijau "Tracking Aktif").
- Menampilkan ringkasan singkat (misal: jarak tempuh hari ini, jumlah trip hari ini) — opsional, nilai tambah UX.
- Navigasi ke Tracking History.
- Tombol Logout:
  - Konfirmasi dialog sebelum logout.
  - **Trigger stop tracking service.**
  - Hapus token dari secure storage.
  - Panggil API `POST /api/v1/auth/logout` (invalidasi refresh token & tutup koneksi WebSocket di server).
  - Redirect ke Login.
- (Opsional) Edit profil.

#### 3.2.5 Tracking History
- Menampilkan list tanggal-tanggal yang memiliki data tracking, diurutkan dari terbaru.
- Setiap item menampilkan:
  - Tanggal
  - Total mileage (jarak tempuh) hari itu, dalam km
  - Durasi total tracking (opsional)
  - Jumlah trip/sesi (opsional, jika dalam 1 hari ada beberapa sesi login/logout)
- Pagination / infinite scroll untuk data historis yang panjang.
- Pull-to-refresh.
- Tap pada item → navigasi ke **Detail Tracking** dengan parameter tanggal.
- Data diambil dari API `GET /api/v1/tracking/history?month=&year=` atau dengan pagination cursor.
- Menampilkan state kosong (empty state) jika belum ada data.

#### 3.2.6 Detail Tracking
- Menampilkan peta (Google Maps / Mapbox) dengan polyline rute perjalanan pada tanggal yang dipilih.
- Marker start point & end point.
- Info panel: total jarak, durasi, waktu mulai & selesai, kecepatan rata-rata (opsional).
- Jika dalam 1 hari terdapat multiple sesi (user logout-login beberapa kali), tampilkan sebagai multiple polylines dengan pemilihan sesi (tab/segmented control), atau gabungkan sebagai satu daftar timeline.
- Data diambil dari API `GET /api/v1/tracking/detail?date=YYYY-MM-DD`.
- Loading state & error handling (misal data tidak ditemukan / koneksi gagal).

### 3.3 Navigasi (Information Architecture)

```
Splash
 ├─ (token valid) → Home
 │                    ├─ Tracking History
 │                    │      └─ Detail Tracking
 │                    └─ Logout → Login
 └─ (token invalid) → Login ⇄ Register
                          └─ (login sukses) → Home
```

### 3.4 Mekanisme Tracking (Foreground & Background)

**Prinsip:** Tracking = *side-effect* dari status autentikasi, bukan aksi manual user (tidak ada tombol start/stop tracking terpisah — start otomatis saat login, stop otomatis saat logout).

**Alur:**
1. Saat login sukses → aplikasi meminta permission lokasi:
   - Android: `ACCESS_FINE_LOCATION`, `ACCESS_BACKGROUND_LOCATION` (Android 10+), `FOREGROUND_SERVICE`, `FOREGROUND_SERVICE_LOCATION` (Android 14+).
   - iOS: `NSLocationWhenInUseUsageDescription` & `NSLocationAlwaysAndWhenInUseUsageDescription`, background mode "Location updates".
2. Setelah permission diberikan, aplikasi menjalankan **background location service**:
   - Android: Foreground Service dengan notifikasi persisten ("Tracking sedang berjalan") — **wajib** di Android 8+ agar service tidak dibunuh sistem.
   - iOS: Background Location Mode + `startUpdatingLocation` dengan `allowsBackgroundLocationUpdates = true`.
3. Service mengambil koordinat GPS secara periodik, namun **titik hanya disimpan & dikirim apabila jarak dari titik terakhir yang tersimpan ≥ 10 meter** (*distance filter* = 10m). Jika user diam (delta < 10m), tidak ada titik baru yang tercatat — ini mengurangi jumlah data yang dikirim & disimpan secara signifikan saat user tidak bergerak.
   - GPS tetap dapat "dibaca" oleh OS lebih sering (untuk keperluan filter jarak), tapi yang benar-benar disimpan ke local queue & dikirim ke server hanya titik yang lolos threshold 10m ini.
   - Trade-off yang perlu disadari: pergerakan kecil di bawah 10m (misal berjalan kaki di dalam gedung/area sempit) tidak akan tercatat sebagai mileage — dianggap acceptable untuk use case kendaraan/perjalanan jarak jauh.
4. Setiap titik lokasi (yang lolos distance filter):
   - Disimpan sementara di local queue (SQLite/MMKV) untuk menjaga data jika koneksi terputus.
   - Dikirim ke backend via **WebSocket** (real-time) jika koneksi tersedia.
   - Jika WebSocket terputus, data di-buffer lokal dan dikirim ulang saat koneksi pulih (retry dengan backoff), atau di-*flush* via REST API batch endpoint sebagai fallback.
5. Saat logout:
   - Service dihentikan (`stopUpdatingLocation`, hentikan foreground service).
   - Sisa data di buffer lokal di-flush terlebih dahulu ke server sebelum koneksi ditutup.
   - Kirim sinyal "end session" ke backend agar sesi tracking ditutup dan mileage difinalisasi.
6. Saat app di-kill paksa oleh user/OS: sesi tracking dianggap terputus; backend menutup sesi otomatis setelah timeout tertentu (lihat §4.5).

> **Catatan penting:** Karena titik lokasi hanya dikirim jika user berpindah ≥ 10 meter, saat user diam lama (misal parkir/istirahat) **tidak akan ada location update yang terkirim sama sekali** — ini normal, bukan indikasi koneksi putus. Oleh karena itu aplikasi tetap mengirim **heartbeat/ping** terpisah dari location update secara berkala (misal tiap 30–60 detik) selama sesi masih login & service berjalan, agar backend dapat membedakan "user diam tapi masih online" vs "koneksi benar-benar terputus/app mati" (lihat §4.5).

**Library rekomendasi (React Native):**
- `react-native-background-geolocation` (Transistor Software) — paling robust untuk kebutuhan background tracking lintas platform, punya built-in offline queue & battery optimization. (Berbayar untuk production, ada alternatif open-source.)
- Alternatif open-source: `react-native-background-fetch` + `@react-native-community/geolocation` / `expo-location` dengan `TaskManager` (jika pakai Expo, perlu custom dev client karena background location tidak didukung penuh di Expo Go).
- WebSocket client: native `WebSocket` API atau `socket.io-client` (jika backend pakai Socket.IO) / raw `gorilla/websocket` di Go tanpa Socket.IO protocol.
- Serialisasi payload: `@msgpack/msgpack` untuk encode/decode MessagePack di client (lihat §4.4.1 untuk pertimbangan).
- Map: `react-native-maps` (Google Maps provider di Android, Apple Maps/Google di iOS) atau Mapbox SDK.
- Secure storage token: `react-native-keychain`.
- State management: Redux Toolkit / Zustand.
- Local DB/queue: `react-native-mmkv` atau `WatermelonDB`/`op-sqlite`.

### 3.5 Non-Functional Requirements (Mobile)
- **Battery efficiency:** gunakan adaptive interval, **distance-filter 10 meter** (§3.4), dan matikan GPS akurasi tinggi saat tidak diperlukan.
- **Data efficiency:** distance-filter 10m sudah menekan jumlah titik yang dikirim secara signifikan saat user diam; payload location update & heartbeat diserialisasi dengan **MessagePack** (bukan JSON teks) untuk mengurangi ukuran data per pesan (§4.4.1). Batching titik lokasi dilakukan jika jaringan lemah.
- **Reliability:** data tidak boleh hilang meski app di-background lama atau koneksi hilang (offline queue + retry).
- **Security:** komunikasi via HTTPS/WSS (TLS), token JWT dengan short-lived access token + refresh token, data lokasi dienkripsi saat disimpan lokal.
- **Permission UX:** jelaskan alasan kebutuhan permission lokasi background secara transparan (rationale dialog) sebelum meminta izin sistem, untuk memenuhi kebijakan Google Play & App Store.
- **Kompatibilitas:** Android 8+ (API 26+), iOS 14+.
- **Perf UI:** Home & History harus load < 1.5 detik pada koneksi normal.

### 3.6 Edge Cases yang Perlu Ditangani
- User mencabut permission lokasi setelah login → tampilkan notifikasi/banner bahwa tracking tidak aktif, arahkan ke Settings.
- User logout saat tidak ada koneksi internet → tetap hentikan service lokal, sinkronisasi sesi ke server saat online kembali.
- Multiple device login bersamaan (opsional: 1 user 1 sesi aktif, atau multi-device diperbolehkan — perlu keputusan bisnis).
- GPS signal hilang (indoor/tunnel) → interpolasi tidak dilakukan otomatis; simpan gap, tampilkan garis putus di peta bila jarak antar titik & waktu melebihi threshold.
- App force-close oleh OS → auto-resume tracking saat app dibuka kembali jika masih dalam status login.

---

## 4. PRD — Backend (Golang, WebSocket, Redis, PostgreSQL)

### 4.1 Arsitektur Umum

```
┌──────────────┐        HTTPS (REST)        ┌───────────────────┐
│  Mobile App  │ ─────────────────────────▶ │   API Gateway /    │
│ (React       │                             │   Go HTTP Service  │
│  Native)     │        WSS (WebSocket)      │  (Auth, CRUD,      │
│              │ ◀─────────────────────────▶ │   WS Handler)      │
└──────────────┘                             └─────────┬──────────┘
                                                         │
                                     ┌───────────────────┼────────────────────┐
                                     ▼                   ▼                    ▼
                               ┌───────────┐      ┌─────────────┐     ┌──────────────┐
                               │   Redis   │      │  PostgreSQL │     │  Worker/Cron  │
                               │ (session, │      │ (persistent │     │ (aggregasi    │
                               │ pub/sub,  │      │   data)     │     │  mileage,     │
                               │ buffer)   │      │             │     │  cleanup)     │
                               └───────────┘      └─────────────┘     └──────────────┘
```

**Komponen:**
- **Go HTTP/WS Service**: menangani REST API (auth, history, detail) dan koneksi WebSocket untuk streaming titik lokasi real-time. Gunakan `gorilla/websocket` atau `nhooyr.io/websocket`. Framework HTTP: `net/http` + `chi`/`gin`/`fiber` (pilih salah satu sesuai preferensi tim).
- **Redis**: 
  - Menyimpan status sesi tracking aktif per user (`tracking:active:{user_id}`) untuk deteksi cepat apakah user sedang tracking.
  - Buffer sementara titik lokasi terbaru sebelum di-flush batch ke PostgreSQL (mengurangi write load DB).
  - Pub/Sub (opsional) jika perlu broadcast posisi user ke consumer lain (misal dashboard monitoring live, di luar scope v1 tapi arsitektur mendukung).
  - Rate limiting & caching data profil/summary.
- **PostgreSQL**: penyimpanan permanen: users, tracking_sessions, tracking_points (atau tracking_points di TimescaleDB extension jika volume tinggi — opsional).
- **Worker/Cron (Go routine terjadwal atau service terpisah)**: agregasi mileage harian, menutup sesi yang "stale" (device mati tanpa sempat kirim logout event), pembersihan data buffer Redis kadaluarsa.

### 4.2 Autentikasi & Session
- JWT-based auth: `access_token` (short-lived, misal 15–60 menit) + `refresh_token` (long-lived, disimpan hashed di DB, rotate saat digunakan).
- Password di-hash dengan `bcrypt`/`argon2`.
- Endpoint:
  - `POST /api/v1/auth/register`
  - `POST /api/v1/auth/login`
  - `POST /api/v1/auth/refresh`
  - `POST /api/v1/auth/logout` → invalidasi refresh token, hapus key session Redis, tutup koneksi WebSocket aktif milik user tsb, tandai sesi tracking berjalan menjadi `ended`.

### 4.3 REST API Endpoints (Utama)

API BASE URL : https://attendee-dreary-hangover.ngrok-free.dev

| Method | Endpoint | Deskripsi |
|---|---|---|
| POST | `/api/v1/auth/register` | Registrasi user baru |
| POST | `/api/v1/auth/login` | Login, mulai eligibility tracking |
| POST | `/api/v1/auth/refresh` | Refresh access token |
| POST | `/api/v1/auth/logout` | Logout, tutup sesi tracking |
| GET | `/api/v1/users/me` | Ambil profil user |
| PUT | `/api/v1/users/me` | Update profil user |
| GET | `/api/v1/tracking/history` | List histori per tanggal + total mileage (paginated) |
| GET | `/api/v1/tracking/detail?date=YYYY-MM-DD` | Detail rute (list koordinat/polyline) untuk tanggal tsb |
| POST | `/api/v1/tracking/points/batch` | Fallback pengiriman titik lokasi via REST (jika WS tidak tersedia) |
| POST | `/api/v1/tracking/session/start` | (opsional eksplisit) Mulai sesi tracking baru |
| POST | `/api/v1/tracking/session/end` | Tutup sesi tracking aktif & finalisasi mileage |

### 4.4 WebSocket Protocol

**Endpoint:** `wss://api.domain.com/ws/tracking?token={access_token}`

**Alur:**
1. Client connect dengan JWT di query param/header saat handshake.
2. Server validasi token → jika valid, buat/lookup `tracking_session` aktif untuk user (buat baru bila belum ada sesi aktif hari ini/saat ini).
3. Client mengirim satu frame **binary (MessagePack-encoded)** per titik lokasi yang lolos distance-filter 10 meter (lihat §3.4). Struktur data (digambarkan dalam bentuk JSON agar mudah dibaca, tapi **dikirim dalam bentuk MessagePack binary**, lihat §4.4.1):
```json
{
  "type": "location_update",
  "session_id": "uuid",
  "lat": -6.401694,
  "lng": 106.821823,
  "accuracy": 8.5,
  "speed": 12.3,
  "altitude": 45.0,
  "timestamp": "2026-08-10T09:15:32Z"
}
```
4. Karena titik hanya terkirim saat user bergerak ≥10m, client juga mengirim frame **heartbeat** terpisah secara berkala (misal tiap 30–60 detik) selama service tracking berjalan, walau user sedang diam:
```json
{ "type": "heartbeat", "session_id": "uuid", "timestamp": "2026-08-10T09:16:02Z" }
```
5. Server:
   - Validasi & simpan titik lokasi ke Redis buffer (list/stream per session); untuk frame heartbeat, cukup update `last_seen` pada `session:active:{user_id}` di Redis tanpa insert ke `tracking_points`.
   - Worker background melakukan flush batch (misal tiap 10 detik atau tiap 50 titik) dari Redis ke PostgreSQL (`tracking_points`), sekaligus update running total distance (dihitung dengan formula Haversine antar titik berurutan) yang disimpan di `tracking_sessions.total_distance`.
   - Kirim ACK ringan ke client (opsional, untuk konfirmasi/telemetry).
6. Saat client kirim `{"type": "session_end"}` atau koneksi WS terputus tanpa `session_end` eksplisit:
   - Jika ada `session_end` eksplisit → tutup sesi langsung.
   - Jika terputus tiba-tiba (network loss/app kill) → sesi tetap "open" sampai worker cleanup mendeteksi *idle timeout berbasis heartbeat* (bukan berbasis location update — lihat §4.5) lalu menutup sesi otomatis.
7. Reconnect: client dapat reconnect dan melanjutkan `session_id` yang sama selama masih dalam window sesi yang sama (misal masih di hari yang sama & belum idle-timeout).

#### 4.4.1 Format Payload: MessagePack vs JSON

**Rekomendasi: gunakan MessagePack** untuk frame `location_update` dan `heartbeat` di WebSocket. Pertimbangan:

| Aspek | JSON (teks) | MessagePack (binary) |
|---|---|---|
| Ukuran payload per titik lokasi | ± 180–220 byte (dengan key string, tanda kutip, dsb) | ± 70–100 byte (key di-encode ringkas, angka dalam bentuk binary) — **penghematan ±40–55%** |
| Kemudahan debug | Mudah dibaca langsung (plain text) | Butuh tool/decoder untuk inspeksi manual |
| Overhead parsing | Text-parsing (sedikit lebih berat di CPU) | Binary parsing (lebih ringan & cepat) |
| Dukungan library | Native di semua platform | Perlu library tambahan (`@msgpack/msgpack` di RN, `vmihailenco/msgpack` atau `shamaton/msgpack` di Go) — sudah matang & stabil |
| Kompatibilitas | Universal | Perlu konsistensi versi schema di kedua sisi (client & server harus sepakat struktur field) |

**Mengapa relevan untuk aplikasi ini:**
- Volume pengiriman titik lokasi bisa tinggi untuk user yang sering bergerak (misal kurir/driver berjam-jam per hari) — penghematan 40–55% per pesan berdampak signifikan terhadap kuota data user, terutama yang menggunakan jaringan seluler terbatas.
- Distance-filter 10m sudah mengurangi *jumlah* pesan yang dikirim; MessagePack mengurangi *ukuran per pesan* — keduanya saling melengkapi (bukan saling menggantikan) untuk efisiensi kuota maksimal.
- Frame heartbeat juga diuntungkan karena dikirim rutin walau user diam (payload sangat kecil, tapi tetap ada overhead per-frame WebSocket yang bisa ditekan).

**Implementasi:**
- Mode WebSocket di-set ke **binary** (bukan text) pada kedua sisi.
- Definisikan schema/struct yang jelas di Go (`struct` dengan tag `msgpack:"..."`) dan di client (TypeScript interface + encoder) agar field & tipe data konsisten; sertakan versioning sederhana (misal field `v: 1`) untuk mengantisipasi perubahan schema di masa depan tanpa breaking existing client.
- REST API (auth, history, detail) **tetap menggunakan JSON** seperti biasa — MessagePack hanya diterapkan pada channel WebSocket yang frekuensinya tinggi, karena di situlah penghematan paling terasa. Mengubah seluruh REST API ke MessagePack tidak sepadan dengan kompleksitas tambahan (REST call jauh lebih jarang dibanding location update).
- Sebagai alternatif/pelengkap yang lebih sederhana: WebSocket **`permessage-deflate` compression** juga dapat diaktifkan di atas MessagePack untuk penghematan tambahan, terutama jika volume data terus bertambah di kemudian hari.

### 4.5 Reliability & Session Management
- **Heartbeat/ping-pong** WebSocket (level aplikasi, frame `heartbeat` di §4.4, terpisah dari WS ping-pong protokol level TCP) untuk deteksi koneksi mati lebih cepat daripada timeout OS, sekaligus menjadi sinyal "user masih online walau diam" karena location_update tidak selalu terkirim (lihat catatan distance-filter di §3.4).
- **Idle session cleanup**: cron job tiap 1–5 menit mengecek `last_seen` (di-update oleh **heartbeat maupun location_update, mana saja yang terakhir diterima**) pada `session:active:{user_id}` di Redis. Jika `last_seen` melebihi threshold (misal 10–15 menit tanpa heartbeat maupun location update sama sekali) → sesi dianggap terputus/mati dan ditutup otomatis dengan status `disconnected`. **Penting:** cleanup TIDAK boleh hanya berdasarkan location_update saja, karena user yang diam lama (namun masih login & terkoneksi) secara sah tidak akan mengirim location_update sama sekali.
- **Force logout cleanup**: saat endpoint logout dipanggil, pastikan idempoten (aman dipanggil walau WS sudah terputus duluan).
- **Horizontal scaling WebSocket**: karena WS bersifat stateful per koneksi, gunakan Redis Pub/Sub atau message broker agar beberapa instance Go service dapat saling koordinasi (misal untuk broadcast "force disconnect" saat logout dari instance lain). Load balancer perlu sticky session atau desain stateless dengan session state di Redis.

### 4.6 Strategi Offline (Opsional tapi Direkomendasikan)
- Backend menyediakan endpoint batch (`POST /tracking/points/batch`) agar mobile app bisa mengirim data yang tertunda (buffer lokal) saat koneksi kembali online, mengisi gap data yang tidak sempat terkirim via WebSocket.
- Endpoint ini menerima array titik dengan timestamp masing-masing agar urutan tetap benar meski dikirim terlambat.

### 4.7 Skema Database (PostgreSQL)

```sql
-- users
CREATE TABLE users (
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
CREATE TABLE refresh_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked BOOLEAN DEFAULT false,
  created_at TIMESTAMPTZ DEFAULT now()
);

-- tracking_sessions (satu baris = satu sesi tracking, dari login/start s.d. logout/idle-timeout)
CREATE TABLE tracking_sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  started_at TIMESTAMPTZ NOT NULL,
  ended_at TIMESTAMPTZ,
  status VARCHAR(20) NOT NULL DEFAULT 'active', -- active | ended | disconnected
  total_distance_m DOUBLE PRECISION DEFAULT 0,   -- dalam meter
  session_date DATE NOT NULL,                    -- untuk grouping histori per tanggal
  created_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX idx_sessions_user_date ON tracking_sessions(user_id, session_date);

-- tracking_points (detail koordinat)
CREATE TABLE tracking_points (
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
CREATE INDEX idx_points_session_time ON tracking_points(session_id, recorded_at);

-- daily_mileage_summary (materialized/aggregated, opsional untuk mempercepat query history)
CREATE TABLE daily_mileage_summary (
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  summary_date DATE NOT NULL,
  total_distance_m DOUBLE PRECISION DEFAULT 0,
  total_duration_sec INT DEFAULT 0,
  session_count INT DEFAULT 0,
  PRIMARY KEY (user_id, summary_date)
);
```

> Catatan: `tracking_points` berpotensi tumbuh sangat besar. Untuk skala produksi jangka panjang, pertimbangkan partisi tabel per bulan (native PostgreSQL partitioning) atau migrasi ke ekstensi **TimescaleDB** untuk time-series data.

### 4.8 Struktur Data Redis

| Key Pattern | Tipe | Fungsi |
|---|---|---|
| `session:active:{user_id}` | String/Hash | Menyimpan `session_id` aktif user saat ini + last_seen timestamp |
| `points:buffer:{session_id}` | List/Stream | Buffer titik lokasi sebelum di-flush ke PostgreSQL |
| `ws:conn:{user_id}` | String | Node/instance ID tempat koneksi WS user berada (untuk multi-instance coordination) |
| `refresh_token:blacklist:{token_id}` | String (TTL) | Token yang sudah direvoke |

### 4.9 Non-Functional Requirements (Backend)
- **Scalability:** Go service stateless untuk REST; WebSocket layer perlu strategi horizontal scaling (Redis Pub/Sub sebagai message bus antar instance).
- **Latency:** titik lokasi harus diterima & di-ACK < 500ms pada kondisi jaringan normal.
- **Availability:** target uptime 99.5%+; graceful shutdown untuk WS agar tidak kehilangan data saat deploy.
- **Security:** 
  - TLS wajib untuk semua koneksi (HTTPS & WSS).
  - Rate limiting di endpoint auth (mencegah brute force).
  - Validasi & sanitasi seluruh input.
  - JWT signing dengan algoritma aman (RS256/HS256 dengan secret kuat, rotasi key berkala).
- **Observability:** structured logging, metrics (Prometheus), tracing (OpenTelemetry) khususnya untuk WS connection count, message throughput, DB write latency.
- **Data retention:** kebijakan retensi `tracking_points` (misal simpan raw points 6–12 bulan, setelah itu hanya simpan summary) untuk mengontrol pertumbuhan storage.

### 4.10 Perhitungan Mileage
- Dihitung server-side menggunakan formula **Haversine** antara dua titik koordinat berurutan dalam satu sesi.
- Filter noise: abaikan pasangan titik dengan `accuracy` buruk (misal > 50m) atau delta jarak tidak wajar (GPS jump) untuk menghindari mileage yang salah.
- Update `total_distance_m` di `tracking_sessions` secara incremental setiap batch flush, dan agregasi ke `daily_mileage_summary` via job terjadwal atau trigger.

---

## 5. Tech Stack Ringkasan

| Layer | Teknologi |
|---|---|
| Mobile | React Native (TypeScript), react-native-background-geolocation, react-native-maps, Redux Toolkit/Zustand, react-native-keychain |
| Backend API | Golang (net/http + Gin/Fiber/Chi), gorilla/websocket |
| Cache/Realtime buffer | Redis |
| Database | PostgreSQL (+ opsional TimescaleDB) |
| Auth | JWT (access + refresh token) |
| Infrastruktur | Docker, (opsional) Kubernetes untuk scaling WS, Nginx/Envoy sebagai reverse proxy dengan WS support |
| Monitoring | Prometheus + Grafana, OpenTelemetry |

---

## 6. Metrik Keberhasilan (Success Metrics)
- Akurasi mileage terekam vs jarak aktual (target margin error < 5%).
- Tracking loss rate (persentase gap data akibat koneksi/app kill) < 2%.
- Battery drain akibat tracking < 5% per jam pemakaian aktif.
- Waktu load Tracking History & Detail < 1.5 detik (P95).
- Uptime backend ≥ 99.5%.

## 7. Roadmap Bertahap (Saran)
1. **Fase 1 (MVP):** Auth, tracking dasar (foreground only), history & detail sederhana.
2. **Fase 2:** Background tracking penuh + offline buffer/sync.
3. **Fase 3:** Optimasi battery, agregasi mileage otomatis, monitoring/observability.
4. **Fase 4 (opsional):** Dashboard admin, geofencing, live sharing lokasi.

## 8. Open Questions (Perlu Konfirmasi Stakeholder)
- Apakah 1 user boleh login di lebih dari 1 device secara bersamaan? Tidak
- Interval pengambilan titik GPS yang diinginkan (trade-off akurasi vs baterai)?
- Apakah dibutuhkan mode "pause tracking" tanpa logout (misal saat istirahat)? Tidak
- Berapa lama data raw tracking_points perlu disimpan (kebijakan retensi & kepatuhan privasi data lokasi)?
- Peta menggunakan Google Maps (berbayar setelah kuota) atau alternatif seperti Mapbox/OpenStreetMap?
