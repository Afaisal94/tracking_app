import React, { useState } from 'react';
import { View, Text, TextInput, TouchableOpacity, Alert, ScrollView, ActivityIndicator } from 'react-native';
import { useNavigation } from '@react-navigation/native';
import { useAuthStore } from '../store/authStore';

export default function RegisterScreen() {
  const navigation = useNavigation<any>();
  const register = useAuthStore((s) => s.register);
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [phone, setPhone] = useState('');
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [loading, setLoading] = useState(false);

  const onRegister = async () => {
    if (!name.trim() || !email.trim() || !password) {
      Alert.alert('Validasi', 'Nama, email, dan password wajib diisi');
      return;
    }
    if (!/\S+@\S+\.\S+/.test(email)) {
      Alert.alert('Validasi', 'Format email tidak valid');
      return;
    }
    if (password.length < 6) {
      Alert.alert('Validasi', 'Password minimal 6 karakter');
      return;
    }
    if (password !== confirm) {
      Alert.alert('Validasi', 'Konfirmasi password tidak cocok');
      return;
    }
    setLoading(true);
    try {
      await register({ name: name.trim(), email: email.trim(), phone: phone.trim(), password });
      Alert.alert('Berhasil', 'Akun berhasil dibuat. Silakan login.', [{ text: 'OK', onPress: () => navigation.navigate('Login') }]);
    } catch (e: any) {
      const msg = e?.response?.data?.message || e?.response?.data?.error || e?.message || 'Registrasi gagal';
      Alert.alert('Gagal', msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <ScrollView contentContainerStyle={{ flexGrow: 1, backgroundColor: '#F1F5F9', padding: 24 }}>
      <View style={{ backgroundColor: '#fff', borderRadius: 16, padding: 20, elevation: 2, marginTop: 12 }}>
        <Text style={{ fontWeight: '600', color: '#334155', marginBottom: 6 }}>Nama Lengkap</Text>
        <TextInput value={name} onChangeText={setName} placeholder="Budi Santoso" style={{ borderWidth: 1, borderColor: '#E2E8F0', borderRadius: 10, padding: 12, marginBottom: 12 }} />

        <Text style={{ fontWeight: '600', color: '#334155', marginBottom: 6 }}>Email</Text>
        <TextInput value={email} onChangeText={setEmail} placeholder="nama@email.com" keyboardType="email-address" autoCapitalize="none" style={{ borderWidth: 1, borderColor: '#E2E8F0', borderRadius: 10, padding: 12, marginBottom: 12 }} />

        <Text style={{ fontWeight: '600', color: '#334155', marginBottom: 6 }}>No. HP</Text>
        <TextInput value={phone} onChangeText={setPhone} placeholder="0812xxxxxxx" keyboardType="phone-pad" style={{ borderWidth: 1, borderColor: '#E2E8F0', borderRadius: 10, padding: 12, marginBottom: 12 }} />

        <Text style={{ fontWeight: '600', color: '#334155', marginBottom: 6 }}>Password</Text>
        <TextInput value={password} onChangeText={setPassword} placeholder="••••••••" secureTextEntry style={{ borderWidth: 1, borderColor: '#E2E8F0', borderRadius: 10, padding: 12, marginBottom: 12 }} />

        <Text style={{ fontWeight: '600', color: '#334155', marginBottom: 6 }}>Konfirmasi Password</Text>
        <TextInput value={confirm} onChangeText={setConfirm} placeholder="••••••••" secureTextEntry style={{ borderWidth: 1, borderColor: '#E2E8F0', borderRadius: 10, padding: 12, marginBottom: 20 }} />

        <TouchableOpacity onPress={onRegister} disabled={loading} style={{ backgroundColor: '#0F172A', borderRadius: 10, padding: 14, alignItems: 'center' }}>
          {loading ? <ActivityIndicator color="#fff" /> : <Text style={{ color: '#fff', fontWeight: '700' }}>Daftar</Text>}
        </TouchableOpacity>
      </View>
    </ScrollView>
  );
}
