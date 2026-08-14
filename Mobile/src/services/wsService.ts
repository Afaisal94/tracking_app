import * as SecureStore from 'expo-secure-store';
import { encode, decode } from '@msgpack/msgpack';
import { WS_BASE_URL, WS_TRACKING_PATH, WS_RECONNECT_BASE_MS, WS_RECONNECT_MAX_MS } from '../config';

type WsStatus = 'disconnected' | 'connecting' | 'connected';

class WsService {
  private ws: WebSocket | null = null;
  private status: WsStatus = 'disconnected';
  private reconnectAttempts = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private shouldReconnect = true;
  private sessionId: string | null = null;

  getStatus() {
    return this.status;
  }

  setSessionId(id: string | null) {
    this.sessionId = id;
  }

  async connect() {
    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) return;
    this.shouldReconnect = true;
    this.status = 'connecting';
    const token = await SecureStore.getItemAsync('access_token');
    if (!token) {
      this.status = 'disconnected';
      return;
    }
    const url = `${WS_BASE_URL}${WS_TRACKING_PATH}?token=${encodeURIComponent(token)}`;
    try {
      this.ws = new WebSocket(url);
      this.ws.binaryType = 'arraybuffer';
      this.ws.onopen = () => {
        this.status = 'connected';
        this.reconnectAttempts = 0;
        console.log('[WS] connected');
      };
      this.ws.onclose = () => {
        this.status = 'disconnected';
        console.log('[WS] closed');
        this.scheduleReconnect();
      };
      this.ws.onerror = (e) => {
        console.warn('[WS] error', e);
      };
      this.ws.onmessage = (ev) => {
        try {
          if (ev.data instanceof ArrayBuffer) {
            const decoded: any = decode(new Uint8Array(ev.data));
            // handle ACK if needed
            // console.log('[WS] msg', decoded);
          } else if (typeof ev.data === 'string') {
            // fallback JSON
            // console.log('[WS] text', ev.data);
          }
        } catch {}
      };
    } catch (e) {
      console.warn('[WS] connect failed', e);
      this.scheduleReconnect();
    }
  }

  private scheduleReconnect() {
    if (!this.shouldReconnect) return;
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    const delay = Math.min(WS_RECONNECT_BASE_MS * 2 ** this.reconnectAttempts, WS_RECONNECT_MAX_MS);
    this.reconnectAttempts += 1;
    this.reconnectTimer = setTimeout(() => this.connect(), delay);
  }

  disconnect() {
    this.shouldReconnect = false;
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    if (this.ws) {
      try {
        this.ws.close();
      } catch {}
      this.ws = null;
    }
    this.status = 'disconnected';
  }

  isConnected() {
    return this.ws?.readyState === WebSocket.OPEN;
  }

  sendLocation(payload: {
    session_id?: string;
    lat: number;
    lng: number;
    accuracy?: number | null;
    speed?: number | null;
    altitude?: number | null;
    timestamp: string;
  }) {
    if (!this.isConnected() || !this.ws) return false;
    const msg = {
      v: 1,
      type: 'location_update',
      session_id: payload.session_id ?? this.sessionId ?? undefined,
      lat: payload.lat,
      lng: payload.lng,
      accuracy: payload.accuracy ?? undefined,
      speed: payload.speed ?? undefined,
      altitude: payload.altitude ?? undefined,
      timestamp: payload.timestamp,
    };
    const bin = encode(msg);
    this.ws.send(bin as any);
    return true;
  }

  sendHeartbeat() {
    if (!this.isConnected() || !this.ws) return false;
    const msg = {
      v: 1,
      type: 'heartbeat',
      session_id: this.sessionId ?? undefined,
      timestamp: new Date().toISOString(),
    };
    const bin = encode(msg);
    this.ws.send(bin as any);
    return true;
  }

  sendSessionEnd() {
    if (!this.isConnected() || !this.ws) return false;
    const msg = {
      v: 1,
      type: 'session_end',
      session_id: this.sessionId ?? undefined,
      timestamp: new Date().toISOString(),
    };
    const bin = encode(msg);
    this.ws.send(bin as any);
    return true;
  }
}

export const wsService = new WsService();
