import React, { useState } from 'react';
import { View, Text, TextInput, TouchableOpacity, Alert, ScrollView, ActivityIndicator } from 'react-native';
import { useNavigation } from '@react-navigation/native';
import { useAuthStore } from '../store/authStore';

export default function LoginScreen() {
  const navigation = useNavigation<any>();
  const login = useAuthStore((s) => s.login);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);

  const onLogin = async () => {
    if (!email.trim() || !password.trim()) {
      Alert.alert('Validasi', 'Email dan password wajib diisi');
      return;
    }
    if (!/\S+@\S+\.\S+/.test(email)) {
      Alert.alert('Validasi', 'Format email tidak valid');
      return;
    }
    setLoading(true);
    try {
      await login(email.trim(), password);
    } catch (e: any) {
      const msg = e?.response?.data?.message || e?.response?.data?.error || e?.message || 'Login gagal';
      Alert.alert('Gagal Login', msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <ScrollView contentContainerStyle={{ flexGrow: 1, backgroundColor: '#F1F5F9', padding: 24, justifyContent: 'center' }}>
      <View style={{ alignItems: 'center', marginBottom: 32 }}>
        <View style={{ width: 64, height: 64, borderRadius: 16, backgroundColor: '#0F172A', alignItems: 'center', justifyContent: 'center' }}>
          <Text style={{ fontSize: 36 }}>📍</Text>
        </View>
        <Text style={{ fontSize: 24, fontWeight: '800', color: '#0F172A', marginTop: 12 }}>Selamat Datang</Text>
        <Text style={{ color: '#64748B', marginTop: 4 }}>Masuk untuk memulai tracking</Text>
      </View>

      <View style={{ backgroundColor: '#fff', borderRadius: 16, padding: 20, elevation: 2 }}>
        <Text style={{ fontWeight: '600', color: '#334155', marginBottom: 6 }}>Email</Text>
        <TextInput
          value={email}
          onChangeText={setEmail}
          placeholder="nama@email.com"
          keyboardType="email-address"
          autoCapitalize="none"
          style={{ borderWidth: 1, borderColor: '#E2E8F0', borderRadius: 10, padding: 12, marginBottom: 14 }}
        />
        <Text style={{ fontWeight: '600', color: '#334155', marginBottom: 6 }}>Password</Text>
        <TextInput
          value={password}
          onChangeText={setPassword}
          placeholder="••••••••"
          secureTextEntry
          style={{ borderWidth: 1, borderColor: '#E2E8F0', borderRadius: 10, padding: 12, marginBottom: 20 }}
        />
        <TouchableOpacity
          onPress={onLogin}
          disabled={loading}
          style={{ backgroundColor: '#0F172A', borderRadius: 10, padding: 14, alignItems: 'center' }}
        >
          {loading ? <ActivityIndicator color="#fff" /> : <Text style={{ color: '#fff', fontWeight: '700' }}>Masuk</Text>}
        </TouchableOpacity>

        <TouchableOpacity onPress={() => navigation.navigate('Register')} style={{ marginTop: 16, alignItems: 'center' }}>
          <Text style={{ color: '#64748B' }}>
            Belum punya akun? <Text style={{ color: '#0EA5E9', fontWeight: '700' }}>Daftar</Text>
          </Text>
        </TouchableOpacity>
      </View>
    </ScrollView>
  );
}
