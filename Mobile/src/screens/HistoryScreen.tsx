import React, { useEffect, useState, useCallback } from 'react';
import { View, Text, FlatList, TouchableOpacity, RefreshControl, ActivityIndicator } from 'react-native';
import { useNavigation } from '@react-navigation/native';
import { api } from '../api/client';
import { formatDistance, formatDuration } from '../utils/haversine';

type HistoryItem = {
  date: string;
  summary_date?: string;
  total_distance_m: number;
  total_distance?: number;
  total_duration_sec?: number;
  session_count?: number;
};

export default function HistoryScreen() {
  const navigation = useNavigation<any>();
  const [items, setItems] = useState<HistoryItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(true);

  const fetchHistory = useCallback(
    async (pageNum: number, isRefresh = false) => {
      try {
        if (isRefresh) setRefreshing(true);
        else if (pageNum === 1) setLoading(true);
        setError(null);
        const res = await api.get(`/api/v1/tracking/history`, { params: { page: pageNum, limit: 20 } });
        const raw = res.data.data ?? res.data;
        const list: HistoryItem[] = Array.isArray(raw) ? raw : raw.items ?? raw.history ?? raw.data ?? [];
        const normalized = list.map((x: any) => ({
          date: x.date ?? x.summary_date ?? x.session_date ?? '',
          total_distance_m: x.total_distance_m ?? x.total_distance ?? 0,
          total_duration_sec: x.total_duration_sec ?? x.duration ?? undefined,
          session_count: x.session_count ?? x.sessions ?? undefined,
        }));
        if (isRefresh || pageNum === 1) setItems(normalized);
        else setItems((prev) => [...prev, ...normalized]);
        setHasMore(normalized.length >= 20);
        setPage(pageNum);
      } catch (e: any) {
        setError(e?.response?.data?.message || e?.message || 'Gagal memuat riwayat');
      } finally {
        setLoading(false);
        setRefreshing(false);
      }
    },
    []
  );

  useEffect(() => {
    fetchHistory(1);
  }, [fetchHistory]);

  const onRefresh = () => fetchHistory(1, true);
  const onEndReached = () => {
    if (!hasMore || loading || refreshing) return;
    fetchHistory(page + 1);
  };

  if (loading) {
    return (
      <View style={{ flex: 1, alignItems: 'center', justifyContent: 'center', backgroundColor: '#F1F5F9' }}>
        <ActivityIndicator color="#0F172A" />
        <Text style={{ color: '#64748B', marginTop: 8 }}>Memuat riwayat...</Text>
      </View>
    );
  }

  if (error) {
    return (
      <View style={{ flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24, backgroundColor: '#F1F5F9' }}>
        <Text style={{ color: '#DC2626', textAlign: 'center' }}>{error}</Text>
        <TouchableOpacity onPress={() => fetchHistory(1)} style={{ marginTop: 12, backgroundColor: '#0F172A', paddingHorizontal: 16, paddingVertical: 10, borderRadius: 8 }}>
          <Text style={{ color: '#fff', fontWeight: '700' }}>Coba Lagi</Text>
        </TouchableOpacity>
      </View>
    );
  }

  if (items.length === 0) {
    return (
      <View style={{ flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24, backgroundColor: '#F1F5F9' }}>
        <Text style={{ fontSize: 48 }}>🗺️</Text>
        <Text style={{ fontWeight: '700', color: '#0F172A', marginTop: 12 }}>Belum ada riwayat</Text>
        <Text style={{ color: '#64748B', textAlign: 'center', marginTop: 6 }}>Perjalanan Anda akan muncul di sini setelah tracking merekam data.</Text>
        <TouchableOpacity onPress={onRefresh} style={{ marginTop: 16, backgroundColor: '#0F172A', paddingHorizontal: 16, paddingVertical: 10, borderRadius: 8 }}>
          <Text style={{ color: '#fff', fontWeight: '700' }}>Refresh</Text>
        </TouchableOpacity>
      </View>
    );
  }

  return (
    <FlatList
      data={items}
      keyExtractor={(item, idx) => `${item.date}-${idx}`}
      contentContainerStyle={{ padding: 16, backgroundColor: '#F1F5F9', flexGrow: 1 }}
      refreshControl={<RefreshControl refreshing={refreshing} onRefresh={onRefresh} />}
      onEndReached={onEndReached}
      onEndReachedThreshold={0.5}
      renderItem={({ item }) => (
        <TouchableOpacity
          onPress={() => navigation.navigate('Detail', { date: item.date })}
          style={{ backgroundColor: '#fff', borderRadius: 12, padding: 16, marginBottom: 12, elevation: 1, borderWidth: 1, borderColor: '#E2E8F0' }}
        >
          <View style={{ flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' }}>
            <Text style={{ fontWeight: '700', color: '#0F172A' }}>{item.date}</Text>
            <Text style={{ color: '#0EA5E9', fontWeight: '700' }}>›</Text>
          </View>
          <View style={{ flexDirection: 'row', marginTop: 8, gap: 12 }}>
            <View style={{ backgroundColor: '#F0F9FF', borderRadius: 8, paddingHorizontal: 10, paddingVertical: 6 }}>
              <Text style={{ fontSize: 12, color: '#64748B' }}>Jarak</Text>
              <Text style={{ fontWeight: '700', color: '#0F172A' }}>{formatDistance(item.total_distance_m)}</Text>
            </View>
            {item.total_duration_sec != null && (
              <View style={{ backgroundColor: '#F0FDF4', borderRadius: 8, paddingHorizontal: 10, paddingVertical: 6 }}>
                <Text style={{ fontSize: 12, color: '#64748B' }}>Durasi</Text>
                <Text style={{ fontWeight: '700', color: '#0F172A' }}>{formatDuration(item.total_duration_sec)}</Text>
              </View>
            )}
            {item.session_count != null && (
              <View style={{ backgroundColor: '#FEFCE8', borderRadius: 8, paddingHorizontal: 10, paddingVertical: 6 }}>
                <Text style={{ fontSize: 12, color: '#64748B' }}>Sesi</Text>
                <Text style={{ fontWeight: '700', color: '#0F172A' }}>{item.session_count}</Text>
              </View>
            )}
          </View>
        </TouchableOpacity>
      )}
      ListFooterComponent={hasMore ? <ActivityIndicator color="#94A3B8" style={{ marginTop: 12 }} /> : null}
    />
  );
}
