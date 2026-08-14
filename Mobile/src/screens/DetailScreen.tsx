import React, { useEffect, useState, useMemo } from 'react';
import { View, Text, ActivityIndicator, ScrollView, TouchableOpacity } from 'react-native';
import { useRoute } from '@react-navigation/native';
import { WebView } from 'react-native-webview';
import { api } from '../api/client';
import { formatDistance, formatDuration, haversineMeters } from '../utils/haversine';

type Point = { lat: number; lng: number; recorded_at?: string; timestamp?: string; accuracy?: number; speed?: number };
type SessionGroup = { session_id: string; points: Point[]; total_distance_m?: number; started_at?: string; ended_at?: string };

function toNumber(v: any): number | null {
  if (v === null || v === undefined || v === '') return null;
  const n = typeof v === 'number' ? v : parseFloat(String(v));
  return Number.isFinite(n) ? n : null;
}

function normalizePoint(p: any): Point | null {
  if (Array.isArray(p) && p.length >= 2 && typeof p[0] === 'number' && typeof p[1] === 'number') {
    return { lat: p[1], lng: p[0], recorded_at: undefined };
  }
  const lat = toNumber(p.lat ?? p.latitude ?? p.latitud ?? p.y ?? p.Lat ?? p.Latitude);
  const lng = toNumber(p.lng ?? p.lon ?? p.long ?? p.longitude ?? p.x ?? p.Lng ?? p.Longitude);
  if (lat === null || lng === null) return null;
  if (lat === 0 && lng === 0) return null;
  if (lat < -90 || lat > 90 || lng < -180 || lng > 180) return null;
  return {
    lat,
    lng,
    recorded_at: p.recorded_at ?? p.timestamp ?? p.created_at ?? p.time ?? p.recordedAt,
    accuracy: toNumber(p.accuracy ?? p.acc) ?? undefined,
    speed: toNumber(p.speed) ?? undefined,
  };
}

function computeDistance(points: Point[]): number {
  let d = 0;
  for (let i = 1; i < points.length; i++) d += haversineMeters(points[i - 1].lat, points[i - 1].lng, points[i].lat, points[i].lng);
  return d;
}

function extractGroups(raw: any): SessionGroup[] {
  if (!raw || typeof raw !== 'object') return [];
  let cur: any = raw;
  if (cur.data && !Array.isArray(cur.data) && typeof cur.data === 'object' && !cur.points && !cur.sessions && !cur.items) {
    const inner = cur.data;
    if (inner.data || inner.points || inner.sessions) cur = inner;
  }
  if (Array.isArray(cur)) {
    const pts = cur.map(normalizePoint).filter(Boolean) as Point[];
    return pts.length ? [{ session_id: 'default', points: pts }] : [];
  }
  if (Array.isArray(cur.points)) {
    const pts = cur.points.map(normalizePoint).filter(Boolean) as Point[];
    return [{ session_id: cur.session_id ?? cur.id ?? 'default', points: pts, total_distance_m: cur.total_distance_m ?? cur.total_distance }];
  }
  if (Array.isArray(cur.sessions)) {
    return cur.sessions.map((s: any) => ({
      session_id: s.session_id ?? s.id ?? s.sessionId ?? 'session',
      points: (s.points ?? s.coords ?? s.coordinates ?? s.path ?? []).map(normalizePoint).filter(Boolean) as Point[],
      total_distance_m: s.total_distance_m ?? s.total_distance ?? s.distance,
      started_at: s.started_at ?? s.startedAt,
      ended_at: s.ended_at ?? s.endedAt,
    })).filter((g: SessionGroup) => g.points.length > 0);
  }
  if (Array.isArray(cur.data)) {
    const pts = cur.data.map(normalizePoint).filter(Boolean) as Point[];
    if (pts.length) return [{ session_id: 'default', points: pts }];
  }
  if (Array.isArray(cur.items)) {
    const first = cur.items[0];
    if (first && (first.points || first.coords)) {
      return cur.items.map((s: any) => ({
        session_id: s.session_id ?? s.id ?? 'session',
        points: (s.points ?? []).map(normalizePoint).filter(Boolean) as Point[],
        total_distance_m: s.total_distance_m,
        started_at: s.started_at, ended_at: s.ended_at,
      })).filter((g: SessionGroup) => g.points.length > 0);
    }
    const pts = cur.items.map(normalizePoint).filter(Boolean) as Point[];
    if (pts.length) return [{ session_id: 'default', points: pts }];
  }
  if (cur.type === 'LineString' && Array.isArray(cur.coordinates)) {
    const pts = cur.coordinates.map(normalizePoint).filter(Boolean) as Point[];
    if (pts.length) return [{ session_id: 'default', points: pts }];
  }
  for (const k of Object.keys(cur)) {
    const v = cur[k];
    if (Array.isArray(v) && v.length > 0) {
      const sample = v[0];
      if (sample && typeof sample === 'object' && ('lat' in sample || 'latitude' in sample || 'lng' in sample || 'lon' in sample || 'longitude' in sample)) {
        const pts = v.map(normalizePoint).filter(Boolean) as Point[];
        if (pts.length) return [{ session_id: 'default', points: pts }];
      }
      if (sample && typeof sample === 'object' && (sample.points || sample.coords)) {
        const groups = v.map((s: any) => ({
          session_id: s.session_id ?? s.id ?? 'session',
          points: (s.points ?? s.coords ?? []).map(normalizePoint).filter(Boolean) as Point[],
          total_distance_m: s.total_distance_m,
          started_at: s.started_at, ended_at: s.ended_at,
        })).filter((g: SessionGroup) => g.points.length > 0);
        if (groups.length) return groups;
      }
    }
  }
  return [];
}

function buildLeafletHtml(points: Point[]): string {
  const safe = JSON.stringify(points.map(p => ({ lat: p.lat, lng: p.lng, t: p.recorded_at ?? '' })));
  return `<!DOCTYPE html>
<html>
<head>
<meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0">
<link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css" />
<style>
  html,body{margin:0;padding:0;height:100%;background:#E2E8F0}
  #map{height:100%;width:100%}
  .leaflet-control-attribution{font-size:9px}
</style>
</head>
<body>
<div id="map"></div>
<script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js"></script>
<script>
  const points = ${safe};
  const map = L.map('map', { zoomControl: false, attributionControl: true });
  L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
    maxZoom: 19,
    attribution: '© OpenStreetMap'
  }).addTo(map);
  L.control.zoom({ position: 'bottomright' }).addTo(map);

  if (points.length === 0) {
    map.setView([-6.2, 106.8], 11);
  } else if (points.length === 1) {
    const ll = [points[0].lat, points[0].lng];
    map.setView(ll, 16);
    L.circleMarker(ll, { radius: 8, color: '#fff', weight: 2, fillColor: '#22C55E', fillOpacity: 1 }).addTo(map);
    L.marker(ll).addTo(map).bindPopup('Titik 1');
  } else {
    const latlngs = points.map(p => [p.lat, p.lng]);
    const poly = L.polyline(latlngs, { color: '#0EA5E9', weight: 5, opacity: 0.95, lineCap: 'round', lineJoin: 'round' }).addTo(map);
    // garis putus untuk gap besar (>300m) biar kelihatan sinyal hilang
    // tetap satu polyline utama, tapi tambahkan dot
    map.fitBounds(poly.getBounds(), { padding: [32, 32] });
    if (latlngs.length <= 120) {
      latlngs.forEach((ll, i) => {
        const isStart = i === 0;
        const isEnd = i === latlngs.length - 1;
        const color = isStart ? '#22C55E' : isEnd ? '#EF4444' : '#0EA5E9';
        const r = isStart || isEnd ? 7 : 4;
        L.circleMarker(ll, { radius: r, color: '#fff', weight: 2, fillColor: color, fillOpacity: 1 }).addTo(map);
      });
    }
    const start = latlngs[0];
    const end = latlngs[latlngs.length - 1];
    L.marker(start, { title: 'Start' }).addTo(map).bindPopup('Start<br>' + (points[0].t || ''));
    L.marker(end, { title: 'End' }).addTo(map).bindPopup('End<br>' + (points[points.length-1].t || ''));
  }
</script>
</body>
</html>`;
}

export default function DetailScreen() {
  const route = useRoute<any>();
  const date: string = route.params?.date;
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [sessions, setSessions] = useState<SessionGroup[]>([]);
  const [summary, setSummary] = useState<{ total_distance_m: number; total_duration_sec?: number; started_at?: string; ended_at?: string } | null>(null);
  const [debug, setDebug] = useState<string | null>(null);
  const [webKey, setWebKey] = useState(0);

  useEffect(() => {
    (async () => {
      try {
        setLoading(true);
        setError(null);
        setDebug(null);
        const res = await api.get('/api/v1/tracking/detail', { params: { date } });
        const raw = res.data?.data ?? res.data;
        console.log('[Detail] raw response for', date, JSON.stringify(raw).slice(0, 2000));
        let groups = extractGroups(raw);
        if (groups.length === 0) {
          const keys = raw && typeof raw === 'object' ? Object.keys(raw).join(', ') : String(typeof raw);
          const sample = raw ? JSON.stringify(raw).slice(0, 800) : 'null';
          setDebug(`Tidak ada point yang bisa diparse. Keys: [${keys}]\nSample: ${sample}`);
        }
        groups = groups.map((g) => ({
          ...g,
          total_distance_m: g.total_distance_m ?? computeDistance(g.points),
        }));
        setSessions(groups);
        const totalDist = groups.reduce((a, g) => a + (g.total_distance_m ?? 0), 0);
        setSummary({
          total_distance_m: raw?.total_distance_m ?? raw?.total_distance ?? totalDist,
          total_duration_sec: raw?.total_duration_sec ?? raw?.duration,
          started_at: raw?.started_at ?? groups[0]?.points[0]?.recorded_at,
          ended_at: raw?.ended_at ?? groups[groups.length - 1]?.points.slice(-1)[0]?.recorded_at,
        });
      } catch (e: any) {
        console.log('[Detail] error', e?.response?.data || e?.message);
        setError(e?.response?.data?.message || e?.message || 'Gagal memuat detail');
      } finally {
        setLoading(false);
      }
    })();
  }, [date]);

  // Gabungkan semua sesi menjadi 1 array titik terurut waktu
  const mergedPoints = useMemo(() => {
    const all = sessions.flatMap((s) => s.points);
    all.sort((a, b) => {
      const ta = a.recorded_at ? new Date(a.recorded_at).getTime() : 0;
      const tb = b.recorded_at ? new Date(b.recorded_at).getTime() : 0;
      return ta - tb;
    });
    return all;
  }, [sessions]);
  const hasCoords = mergedPoints.length > 0;
  const leafletHtml = useMemo(() => buildLeafletHtml(mergedPoints), [mergedPoints]);

  if (loading) {
    return (
      <View style={{ flex: 1, alignItems: 'center', justifyContent: 'center', backgroundColor: '#F1F5F9' }}>
        <ActivityIndicator color="#0F172A" />
        <Text style={{ color: '#64748B', marginTop: 8 }}>Memuat rute {date}...</Text>
      </View>
    );
  }

  if (error) {
    return (
      <View style={{ flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24, backgroundColor: '#F1F5F9' }}>
        <Text style={{ color: '#DC2626', textAlign: 'center' }}>{error}</Text>
        <Text style={{ color: '#64748B', textAlign: 'center', marginTop: 8, fontSize: 12 }}>Tanggal: {date}</Text>
      </View>
    );
  }

  return (
    <View style={{ flex: 1, backgroundColor: '#F1F5F9' }}>
      {sessions.length > 1 && (
        <View style={{ margin: 12, marginBottom: 0, backgroundColor: '#EFF6FF', borderRadius: 10, padding: 10, borderWidth: 1, borderColor: '#BFDBFE' }}>
          <Text style={{ fontSize: 12, color: '#1E40AF', fontWeight: '700', textAlign: 'center' }}>
            {sessions.length} sesi digabungkan • {mergedPoints.length} titik total
          </Text>
        </View>
      )}

      <View style={{ height: 380, margin: 12, borderRadius: 16, overflow: 'hidden', borderWidth: 1, borderColor: '#E2E8F0', backgroundColor: '#E2E8F0' }}>
        {hasCoords ? (
          <WebView
            key={`merged-${webKey}`}
            originWhitelist={['*']}
            source={{ html: leafletHtml }}
            style={{ flex: 1, backgroundColor: '#E2E8F0' }}
            javaScriptEnabled
            domStorageEnabled
            mixedContentMode="always"
            allowFileAccess
            startInLoadingState
            renderLoading={() => (
              <View style={{ position: 'absolute', top: 0, left: 0, right: 0, bottom: 0, alignItems: 'center', justifyContent: 'center', backgroundColor: '#E2E8F0' }}>
                <ActivityIndicator color="#0F172A" />
              </View>
            )}
          />
        ) : (
          <View style={{ flex: 1, alignItems: 'center', justifyContent: 'center', backgroundColor: '#E2E8F0', padding: 16 }}>
            <Text style={{ color: '#64748B', fontWeight: '700' }}>Tidak ada titik untuk tanggal ini</Text>
            <Text style={{ color: '#94A3B8', fontSize: 12, textAlign: 'center', marginTop: 6 }}>{debug ?? `Tanggal: ${date} • Sesi: ${sessions.length}`}</Text>
            {debug && <Text style={{ color: '#94A3B8', fontSize: 10, textAlign: 'center', marginTop: 8 }}>{debug}</Text>}
          </View>
        )}
      </View>

      <ScrollView contentContainerStyle={{ padding: 16 }}>
        <View style={{ backgroundColor: '#fff', borderRadius: 12, padding: 16, borderWidth: 1, borderColor: '#E2E8F0' }}>
          <Text style={{ fontWeight: '800', color: '#0F172A', marginBottom: 12 }}>Ringkasan {date}</Text>
          <View style={{ flexDirection: 'row', gap: 12 }}>
            <View style={{ flex: 1, backgroundColor: '#F0F9FF', borderRadius: 10, padding: 12 }}>
              <Text style={{ fontSize: 11, color: '#64748B' }}>Total Jarak</Text>
              <Text style={{ fontWeight: '800', color: '#0F172A', marginTop: 4 }}>{formatDistance(summary?.total_distance_m ?? 0)}</Text>
            </View>
            {summary?.total_duration_sec != null && (
              <View style={{ flex: 1, backgroundColor: '#F0FDF4', borderRadius: 10, padding: 12 }}>
                <Text style={{ fontSize: 11, color: '#64748B' }}>Durasi</Text>
                <Text style={{ fontWeight: '800', color: '#0F172A', marginTop: 4 }}>{formatDuration(summary.total_duration_sec)}</Text>
              </View>
            )}
          </View>
          <View style={{ marginTop: 12 }}>
            <Text style={{ fontSize: 12, color: '#64748B' }}>Titik tercatat: {mergedPoints.length}{sessions.length > 1 ? ` dari ${sessions.length} sesi` : ''}</Text>
            {summary?.started_at && <Text style={{ fontSize: 12, color: '#64748B' }}>Mulai: {new Date(summary.started_at).toLocaleString('id-ID')}</Text>}
            {summary?.ended_at && <Text style={{ fontSize: 12, color: '#64748B' }}>Selesai: {new Date(summary.ended_at).toLocaleString('id-ID')}</Text>}
          </View>
          {hasCoords && (
            <TouchableOpacity onPress={() => setWebKey(k => k + 1)} style={{ marginTop: 12, backgroundColor: '#0F172A', borderRadius: 8, paddingVertical: 10, alignItems: 'center' }}>
              <Text style={{ color: '#fff', fontWeight: '700', fontSize: 12 }}>Muat Ulang Peta</Text>
            </TouchableOpacity>
          )}
          <Text style={{ fontSize: 11, color: '#94A3B8', marginTop: 12 }}>
            Peta pakai OpenStreetMap (Leaflet) — tidak butuh API key. Garis biru = rute gabungan semua sesi, hijau = start, merah = end.
          </Text>
          {debug && hasCoords && <Text style={{ fontSize: 10, color: '#94A3B8', marginTop: 8 }}>{debug}</Text>}
        </View>

        
      </ScrollView>
    </View>
  );
}
