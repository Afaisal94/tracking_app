import React, { useEffect, useState } from 'react';
import { View, Text, TouchableOpacity, Alert, ScrollView, RefreshControl } from 'react-native';
import { useNavigation } from '@react-navigation/native';
import { useAuthStore } from '../store/authStore';
import { trackingService } from '../services/trackingService';
import { api } from '../api/client';
import { formatDistance } from '../utils/haversine';

export default function HomeScreen() {
  const navigation = useNavigation<any>();
  const { user, logout, fetchProfile } = useAuthStore();
  const [tracking, setTracking] = useState(trackingService.getState());
  const [todaySummary, setTodaySummary] = useState<{ total_distance_m: number; session_count?: number } | null>(null);
  const [refreshing, setRefreshing] = useState(false);

  useEffect(() => {
    const unsub = trackingService.subscribe(setTracking);
    return unsub;
  }, []);

  const loadSummary = async () => {
    try {
      const today = new Date().toISOString().slice(0, 10);
      const res = await api.get(`/api/v1/tracking/detail?date=${today}`).catch(() => null);
      // fallback: try history
      if (res?.data) {
        const d = res.data.data ?? res.data;
        // detail returns points; compute or use summary if available
        if (d.total_distance_m != null) setTodaySummary({ total_distance_m: d.total_distance_m });
        else if (Array.isArray(d.points) || Array.isArray(d)) {
          // no summary, keep null
        }
      }
      // also try history endpoint for today
      const hist = await api.get('/api/v1/tracking/history?limit=1').catch(() => null);
      if (hist?.data) {
        const h = hist.data.data ?? hist.data;
        const list = Array.isArray(h) ? h : h.items ?? h.history ?? [];
        const todayItem = list.find((x: any) => x.date === new Date().toISOString().slice(0, 10) || x.summary_date === new Date().toISOString().slice(0, 10));
        if (todayItem) setTodaySummary({ total_distance_m: todayItem.total_distance_m ?? todayItem.total_distance ?? 0, session_count: todayItem.session_count });
      }
    } catch {}
  };

  useEffect(() => {
    fetchProfile().catch(() => {});
    loadSummary();
  }, []);

  const onRefresh = async () => {
    setRefreshing(true);
    await Promise.all([fetchProfile().catch(() => {}), loadSummary()]);
    setRefreshing(false);
  };

  const onLogout = () => {
    Alert.alert('Konfirmasi', 'Yakin ingin logout? Tracking akan dihentikan.', [
      { text: 'Batal', style: 'cancel' },
      {
        text: 'Logout',
        style: 'destructive',
        onPress: async () => {
          try {
            await logout();
          } catch (e: any) {
            Alert.alert('Error', e.message);
          }
        },
      },
    ]);
  };

  return (
    <ScrollView
      style={{ flex: 1, backgroundColor: '#F1F5F9' }}
      contentContainerStyle={{ padding: 16 }}
      refreshControl={<RefreshControl refreshing={refreshing} onRefresh={onRefresh} />}
    >
      {/* Profile Card */}
      <View style={{ backgroundColor: '#fff', borderRadius: 16, padding: 20, elevation: 2 }}>
        <View style={{ flexDirection: 'row', alignItems: 'center' }}>
          <View style={{ width: 56, height: 56, borderRadius: 28, backgroundColor: '#0F172A', alignItems: 'center', justifyContent: 'center' }}>
            <Text style={{ color: '#fff', fontSize: 20, fontWeight: '800' }}>{(user?.name?.[0] ?? 'U').toUpperCase()}</Text>
          </View>
          <View style={{ marginLeft: 14, flex: 1 }}>
            <Text style={{ fontSize: 16, fontWeight: '700', color: '#0F172A' }}>{user?.name ?? '-'}</Text>
            <Text style={{ color: '#64748B', marginTop: 2 }}>{user?.email ?? '-'}</Text>
            {user?.phone ? <Text style={{ color: '#64748B', fontSize: 12 }}>{user.phone}</Text> : null}
          </View>
        </View>

        {/* Tracking badge */}
        <View
          style={{
            marginTop: 16,
            flexDirection: 'row',
            alignItems: 'center',
            backgroundColor: tracking.isTracking ? '#DCFCE7' : '#FEE2E2',
            borderRadius: 20,
            paddingVertical: 6,
            paddingHorizontal: 12,
            alignSelf: 'flex-start',
          }}
        >
          <View style={{ width: 8, height: 8, borderRadius: 4, backgroundColor: tracking.isTracking ? '#16A34A' : '#DC2626', marginRight: 6 }} />
          <Text style={{ color: tracking.isTracking ? '#166534' : '#991B1B', fontWeight: '700', fontSize: 12 }}>
            {tracking.isTracking ? 'Tracking Aktif' : 'Tracking Tidak Aktif'}
          </Text>
        </View>

        {todaySummary && (
          <View style={{ flexDirection: 'row', marginTop: 16, gap: 12 }}>
            <View style={{ flex: 1, backgroundColor: '#F8FAFC', borderRadius: 12, padding: 12, borderWidth: 1, borderColor: '#E2E8F0' }}>
              <Text style={{ color: '#64748B', fontSize: 12 }}>Jarak Hari Ini</Text>
              <Text style={{ fontWeight: '800', fontSize: 16, color: '#0F172A', marginTop: 4 }}>{formatDistance(todaySummary.total_distance_m)}</Text>
            </View>
            {todaySummary.session_count != null && (
              <View style={{ flex: 1, backgroundColor: '#F8FAFC', borderRadius: 12, padding: 12, borderWidth: 1, borderColor: '#E2E8F0' }}>
                <Text style={{ color: '#64748B', fontSize: 12 }}>Sesi Hari Ini</Text>
                <Text style={{ fontWeight: '800', fontSize: 16, color: '#0F172A', marginTop: 4 }}>{todaySummary.session_count}</Text>
              </View>
            )}
          </View>
        )}
      </View>

      {/* Actions */}
      <TouchableOpacity
        onPress={() => navigation.navigate('History')}
        style={{ backgroundColor: '#0F172A', borderRadius: 12, padding: 16, alignItems: 'center', marginTop: 16 }}
      >
        <Text style={{ color: '#fff', fontWeight: '700' }}>Lihat Riwayat Perjalanan</Text>
      </TouchableOpacity>

      <TouchableOpacity
        onPress={onLogout}
        style={{ backgroundColor: '#fff', borderRadius: 12, padding: 16, alignItems: 'center', marginTop: 12, borderWidth: 1, borderColor: '#FECACA' }}
      >
        <Text style={{ color: '#DC2626', fontWeight: '700' }}>Logout</Text>
      </TouchableOpacity>

      <Text style={{ color: '#94A3B8', fontSize: 11, textAlign: 'center', marginTop: 20 }}>
        Tracking berjalan otomatis saat login dan berhenti saat logout. Jarak dihitung dengan filter 10 meter.
      </Text>
    </ScrollView>
  );
}
