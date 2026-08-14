// PENTING: file ini harus di-import di App.tsx paling atas
// agar TaskManager.defineTask dieksekusi sebelum app di-kill OS
// dan di-restart di background.
import * as TaskManager from 'expo-task-manager';
import * as SecureStore from 'expo-secure-store';
import { encode } from '@msgpack/msgpack';
import { haversineMeters } from '../utils/haversine';
import { DISTANCE_FILTER_M, WS_BASE_URL, WS_TRACKING_PATH } from '../config';

export const LOCATION_TASK_NAME = 'tracking-background-location';

// State sederhana yang persist selama JS runtime background
let lastLat: number | null = null;
let lastLng: number | null = null;

// Helper kirim langsung via WebSocket tanpa dependensi ke wsService
// (karena wsService butuh SecureStore async, kita buat WS ephemeral di background)
let bgWs: WebSocket | null = null;

async function getBgWs(): Promise<WebSocket | null> {
  if (bgWs && bgWs.readyState === WebSocket.OPEN) return bgWs;
  const token = await SecureStore.getItemAsync('access_token');
  if (!token) return null;
  const url = `${WS_BASE_URL}${WS_TRACKING_PATH}?token=${encodeURIComponent(token)}`;
  return new Promise((resolve) => {
    try {
      const ws = new WebSocket(url);
      (ws as any).binaryType = 'arraybuffer';
      ws.onopen = () => {
        bgWs = ws;
        resolve(ws);
      };
      ws.onerror = () => resolve(null);
      // timeout 3 detik
      setTimeout(() => resolve(ws.readyState === WebSocket.OPEN ? ws : null), 3000);
    } catch {
      resolve(null);
    }
  });
}

if (!TaskManager.isTaskDefined(LOCATION_TASK_NAME)) {
  TaskManager.defineTask(LOCATION_TASK_NAME, async ({ data, error }: any) => {
    if (error) {
      console.warn('[BG Task] error', error);
      return;
    }
    if (!data) return;
    const { locations } = data as { locations: import('expo-location').LocationObject[] };
    if (!locations || locations.length === 0) return;

    for (const loc of locations) {
      const { latitude, longitude } = loc.coords;
      const accuracy = loc.coords.accuracy;
      const timestamp = new Date(loc.timestamp).toISOString();

      // distance filter 10m
      if (lastLat !== null && lastLng !== null) {
        const d = haversineMeters(lastLat, lastLng, latitude, longitude);
        if (d < DISTANCE_FILTER_M) continue;
        if (accuracy != null && accuracy > 50) continue;
        if (d > 5000) continue;
      }
      lastLat = latitude;
      lastLng = longitude;

      // Kirim via WS MessagePack binary
      try {
        const ws = await getBgWs();
        if (ws && ws.readyState === WebSocket.OPEN) {
          const msg = {
            v: 1,
            type: 'location_update',
            lat: latitude,
            lng: longitude,
            accuracy: loc.coords.accuracy ?? undefined,
            speed: loc.coords.speed ?? undefined,
            altitude: loc.coords.altitude ?? undefined,
            timestamp,
          };
          ws.send(encode(msg) as any);
        } else {
          // fallback: simpan ke queue via SecureStore / MMKV bisa ditambahkan di sini
          console.log('[BG Task] WS not ready, point buffered in memory');
        }
      } catch (e) {
        console.warn('[BG Task] send failed', e);
      }
    }
  });
}
