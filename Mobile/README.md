# Tracking App Mobile (React Native + Expo)

Aplikasi tracking sesuai PRD v1.0 — login otomatis start tracking, logout stop tracking, background service, WebSocket MessagePack, history & detail peta.

## Stack
- Expo SDK 52, React Native 0.76, TypeScript
- Navigation: @react-navigation/native-stack
- State: Zustand
- Secure storage: expo-secure-store
- Location: expo-location + expo-task-manager (background)
- Maps: react-native-maps (Google Maps)
- WebSocket binary MessagePack: @msgpack/msgpack
- HTTP: axios

## Setup

```bash
npm install
# set Google Maps API key di app.json -> android.config.googleMaps.apiKey
npx expo prebuild   # diperlukan untuk background location & maps native
npx expo run:android
# atau
npx expo run:ios
```

> Expo Go tidak mendukung background location — gunakan dev client / prebuild.

## Konfigurasi

- `src/config.ts` — `API_BASE_URL` (default: `https://attendee-dreary-hangover.ngrok-free.dev`), distance filter 10m, heartbeat 45s.
- `app.json` — permissions lokasi foreground/background + foregroundService.

## Alur Tracking (§3.4 PRD)

- Saat login sukses → request permission → `trackingService.start()` → start foreground service + `watchPositionAsync` + `startLocationUpdatesAsync` (task `tracking-background-location`).
- Distance filter 10m + filter accuracy >50m + jump >5km diabaikan.
- Titik lolos filter → coba kirim via WebSocket binary MessagePack (`wss://.../ws/tracking?token=...`), jika gagal → enqueue ke `queueService` dan flush via `POST /api/v1/tracking/points/batch` tiap 15 detik.
- Heartbeat MessagePack tiap 45 detik agar backend bedakan "diam tapi online" vs putus.
- Saat logout → `sendSessionEnd` → flush queue → `POST /tracking/session/end` → stop location updates → disconnect WS → hapus token.

## Screen

- Splash → cek token → Home / Login
- Login / Register
- Home (profil, badge tracking, ringkasan hari ini, logout)
- History (paginated, pull-to-refresh, empty state)
- Detail (Map polyline, marker start/end, tabs multi-sesi, info panel)

## Catatan

- Buat folder `assets/` dengan `icon.png`, `splash.png`, `adaptive-icon.png` sebelum build.
- Untuk production, ganti `YOUR_GOOGLE_MAPS_API_KEY` dan set `API_BASE_URL` ke domain produksi.
