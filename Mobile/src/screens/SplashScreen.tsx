import React from 'react';
import { View, Text, ActivityIndicator } from 'react-native';

export default function SplashScreen() {
  return (
    <View style={{ flex: 1, backgroundColor: '#0F172A', alignItems: 'center', justifyContent: 'center' }}>
      <View
        style={{
          width: 80,
          height: 80,
          borderRadius: 20,
          backgroundColor: '#38BDF8',
          alignItems: 'center',
          justifyContent: 'center',
          marginBottom: 16,
        }}
      >
        <Text style={{ fontSize: 36 }}>📍</Text>
      </View>
      <Text style={{ color: '#fff', fontSize: 22, fontWeight: '800', letterSpacing: 0.5 }}>Tracking App</Text>
      <Text style={{ color: '#94A3B8', marginTop: 6 }}>Merekam perjalanan Anda</Text>
      <ActivityIndicator color="#38BDF8" style={{ marginTop: 24 }} />
    </View>
  );
}
