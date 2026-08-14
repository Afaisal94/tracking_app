import * as Location from 'expo-location';
import { Platform } from 'react-native';
import { wsService } from './wsService';
import { queueService } from './queueService';
import { api } from '../api/client';
import { DISTANCE_FILTER_M, HEARTBEAT_INTERVAL_MS } from '../config';
import { haversineMeters } from '../utils/haversine';
import { LOCATION_TASK_NAME } from './backgroundTask';

type TrackingState = {
  isTracking: boolean;
  lastLat: number | null;
  lastLng: number | null;
  sessionId: string | null;
};

class TrackingService {
  private state: TrackingState = { isTracking: false, lastLat: null, lastLng: null, sessionId: null };
  private foregroundSub: Location.LocationSubscription | null = null;
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null;
  private flushTimer: ReturnType<typeof setInterval> | null = null;
  private listeners: Set<(s: TrackingState) => void> = new Set();

  constructor() {
    // Task sudah didefinisikan di backgroundTask.ts (di-import di App.tsx paling atas)
    // Jangan define lagi di sini agar tidak double-define saat hot reload
  }

  subscribe(cb: (s: TrackingState) => void) {
    this.listeners.add(cb);
    return () => this.listeners.delete(cb);
  }

  private emit() {
    this.listeners.forEach((cb) => cb({ ...this.state }));
  }

  getState() {
    return { ...this.state };
  }

  async requestPermissions(): Promise<boolean> {
    const fg = await Location.requestForegroundPermissionsAsync();
    if (fg.status !== 'granted') return false;
    const bg = await Location.requestBackgroundPermissionsAsync();
    if (bg.status !== 'granted') {
      // On iOS, background permission may be not required immediately; still allow foreground tracking
      console.warn('[Tracking] background permission not granted');
      // For Android, we need it; return false to show rationale
      if (Platform.OS === 'android') return false;
    }
    return true;
  }

  async start() {
    if (this.state.isTracking) return;
    const granted = await this.requestPermissions();
    if (!granted) throw new Error('Izin lokasi diperlukan untuk tracking');

    // Try to start session explicitly (optional endpoint)
    try {
      const res = await api.post('/api/v1/tracking/session/start');
      const sid = res.data?.data?.session_id ?? res.data?.session_id ?? null;
      if (sid) {
        this.state.sessionId = sid;
        wsService.setSessionId(sid);
      }
    } catch {
      // ignore, server may auto-create on WS connect
    }

    // Connect WebSocket
    wsService.connect();

    // Start background updates
    const hasBg = await Location.hasStartedLocationUpdatesAsync(LOCATION_TASK_NAME).catch(() => false);
    if (!hasBg) {
      await Location.startLocationUpdatesAsync(LOCATION_TASK_NAME, {
        accuracy: Location.Accuracy.Balanced,
        distanceInterval: DISTANCE_FILTER_M,
        timeInterval: 5000,
        foregroundService: {
          notificationTitle: 'Tracking Aktif',
          notificationBody: 'Perjalanan Anda sedang direkam',
          notificationColor: '#0F172A',
        },
        pausesUpdatesAutomatically: false,
        showsBackgroundLocationIndicator: true,
      });
    }

    // Foreground watcher for immediate feedback & distance filter
    this.foregroundSub = await Location.watchPositionAsync(
      {
        accuracy: Location.Accuracy.Balanced,
        distanceInterval: DISTANCE_FILTER_M,
        timeInterval: 5000,
      },
      (loc) => this.handleLocation(loc)
    );

    this.state.isTracking = true;
    this.emit();

    // Heartbeat every 45s
    this.heartbeatTimer = setInterval(() => {
      if (wsService.isConnected()) wsService.sendHeartbeat();
    }, HEARTBEAT_INTERVAL_MS);

    // Flush queue every 15s
    this.flushTimer = setInterval(() => {
      if (queueService.size() > 0) queueService.flush();
    }, 15000);
  }

  async stop() {
    // Send session_end via WS before disconnect
    try {
      wsService.sendSessionEnd();
    } catch {}
    // Flush remaining queue via REST
    try {
      await queueService.flush();
    } catch {}
    try {
      await api.post('/api/v1/tracking/session/end', { session_id: this.state.sessionId }).catch(() => {});
    } catch {}

    if (this.foregroundSub) {
      this.foregroundSub.remove();
      this.foregroundSub = null;
    }
    const hasBg = await Location.hasStartedLocationUpdatesAsync(LOCATION_TASK_NAME).catch(() => false);
    if (hasBg) {
      await Location.stopLocationUpdatesAsync(LOCATION_TASK_NAME).catch(() => {});
    }
    if (this.heartbeatTimer) clearInterval(this.heartbeatTimer);
    if (this.flushTimer) clearInterval(this.flushTimer);
    this.heartbeatTimer = null;
    this.flushTimer = null;
    wsService.disconnect();
    wsService.setSessionId(null);
    this.state = { isTracking: false, lastLat: null, lastLng: null, sessionId: null };
    this.emit();
  }

  private async handleLocation(loc: Location.LocationObject) {
    const { latitude, longitude } = loc.coords;
    const accuracy = loc.coords.accuracy;
    const speed = loc.coords.speed;
    const altitude = loc.coords.altitude;
    const timestamp = new Date(loc.timestamp).toISOString();

    // Distance filter 10m
    if (this.state.lastLat !== null && this.state.lastLng !== null) {
      const d = haversineMeters(this.state.lastLat, this.state.lastLng, latitude, longitude);
      if (d < DISTANCE_FILTER_M) return;
      // Filter GPS jump / bad accuracy
      if (accuracy != null && accuracy > 50) return;
      if (d > 5000) return; // unrealistic jump >5km in one update, ignore
    }

    this.state.lastLat = latitude;
    this.state.lastLng = longitude;
    this.emit();

    const point = {
      lat: latitude,
      lng: longitude,
      accuracy,
      speed,
      altitude,
      timestamp,
      session_id: this.state.sessionId ?? undefined,
    };

    // Try WS first
    const sent = wsService.sendLocation(point);
    if (!sent) {
      queueService.enqueue(point);
    }
  }
}

export const trackingService = new TrackingService();
