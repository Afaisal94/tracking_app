export const API_BASE_URL = 'https://attendee-dreary-hangover.ngrok-free.dev';
export const WS_BASE_URL = API_BASE_URL.replace(/^http/, 'ws'); // https -> wss
export const WS_TRACKING_PATH = '/ws/tracking';

export const DISTANCE_FILTER_M = 10; // meters
export const HEARTBEAT_INTERVAL_MS = 45_000; // 45s
export const LOCATION_INTERVAL_MS = 5000;
export const WS_RECONNECT_BASE_MS = 2000;
export const WS_RECONNECT_MAX_MS = 30000;
