import { create } from "zustand";
import { auth } from "@/lib/api";

const userFromStorage = () => {
  try {
    const u = localStorage.getItem("auth_user");
    return u ? JSON.parse(u) : null;
  } catch {
    return null;
  }
};

export const useAuthStore = create((set) => ({
  user: userFromStorage(),
  isLoading: false,
  error: null,

  login: async (email, password) => {
    set({ isLoading: true, error: null });
    try {
      const data = await auth.login({ email, password });
      localStorage.setItem("auth_user", JSON.stringify(data.user));
      set({ user: data.user, isLoading: false });
    } catch (err) {
      set({ error: err.message, isLoading: false });
      throw err;
    }
  },

  signup: async (name, email, password) => {
    set({ isLoading: true, error: null });
    try {
      const data = await auth.signup({ name, email, password });
      localStorage.setItem("auth_user", JSON.stringify(data.user));
      set({ user: data.user, isLoading: false });
    } catch (err) {
      set({ error: err.message, isLoading: false });
      throw err;
    }
  },

  logout: async () => {
    try {
      await auth.logout();
    } catch {
    } finally {
      localStorage.removeItem("auth_user");
      set({ user: null, error: null });
    }
  },

  checkAuth: async () => {
    set({ isLoading: true });
    try {
      const data = await auth.me();
      localStorage.setItem("auth_user", JSON.stringify(data.user));
      set({ user: data.user, isLoading: false });
    } catch {
      localStorage.removeItem("auth_user");
      set({ user: null, isLoading: false });
    }
  },

  clearError: () => set({ error: null }),
}));
