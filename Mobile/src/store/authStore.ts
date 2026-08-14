import { create } from 'zustand';
import * as SecureStore from 'expo-secure-store';
import { api } from '../api/client';
import { trackingService } from '../services/trackingService';

type User = {
  id: string;
  name: string;
  email: string;
  phone?: string;
  avatar_url?: string;
};

type AuthState = {
  user: User | null;
  isAuthenticated: boolean;
  isBootstrapping: boolean;
  bootstrap: () => Promise<void>;
  login: (email: string, password: string) => Promise<void>;
  register: (data: { name: string; email: string; phone: string; password: string }) => Promise<void>;
  logout: () => Promise<void>;
  fetchProfile: () => Promise<void>;
};

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  isAuthenticated: false,
  isBootstrapping: true,

  bootstrap: async () => {
    try {
      const token = await SecureStore.getItemAsync('access_token');
      if (!token) {
        set({ isBootstrapping: false, isAuthenticated: false });
        return;
      }
      const res = await api.get('/api/v1/users/me');
      const user = res.data.data ?? res.data;
      set({ user, isAuthenticated: true, isBootstrapping: false });
      // auto-resume tracking
      trackingService.start().catch(() => {});
    } catch {
      await SecureStore.deleteItemAsync('access_token');
      await SecureStore.deleteItemAsync('refresh_token');
      set({ isAuthenticated: false, isBootstrapping: false });
    }
  },

  login: async (email, password) => {
    const res = await api.post('/api/v1/auth/login', { email, password });
    const payload = res.data.data ?? res.data;
    const access_token: string = payload.access_token;
    const refresh_token: string = payload.refresh_token;
    const user: User | undefined = payload.user;
    await SecureStore.setItemAsync('access_token', access_token);
    if (refresh_token) await SecureStore.setItemAsync('refresh_token', refresh_token);
    if (user) set({ user, isAuthenticated: true });
    else {
      // fetch profile
      try {
        const me = await api.get('/api/v1/users/me');
        set({ user: me.data.data ?? me.data, isAuthenticated: true });
      } catch {
        set({ isAuthenticated: true });
      }
    }
    // start tracking after login
    trackingService.start().catch(console.warn);
  },

  register: async (data) => {
    await api.post('/api/v1/auth/register', {
      name: data.name,
      email: data.email,
      phone: data.phone,
      password: data.password,
      confirm_password: data.password,
    });
  },

  logout: async () => {
    // flush & stop tracking first
    try {
      await trackingService.stop();
    } catch {}
    try {
      await api.post('/api/v1/auth/logout');
    } catch {}
    await SecureStore.deleteItemAsync('access_token');
    await SecureStore.deleteItemAsync('refresh_token');
    set({ user: null, isAuthenticated: false });
  },

  fetchProfile: async () => {
    const res = await api.get('/api/v1/users/me');
    set({ user: res.data.data ?? res.data });
  },
}));
