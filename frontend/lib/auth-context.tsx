"use client";

import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from "react";
import { apiGetAuthed, clearTokens, getTokens } from "@/lib/api";

export type CurrentUser = {
  id: string;
  full_name: string;
  email: string;
  phone_number: string | null;
  profile_picture_url: string | null;
  preferred_currency: string;
  theme_preference: string;
  role: string;
  status: string;
  email_verified: boolean;
};

type AuthContextValue = {
  user: CurrentUser | null;
  loading: boolean;
  logout: () => void;
  refresh: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<CurrentUser | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    if (!getTokens()) {
      setUser(null);
      setLoading(false);
      return;
    }

    try {
      const payload = await apiGetAuthed<{ data: { user: CurrentUser } }>("/users/me");
      setUser(payload.data.user);
    } catch {
      clearTokens();
      setUser(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  function logout() {
    clearTokens();
    setUser(null);
    window.location.assign("/login");
  }

  return <AuthContext.Provider value={{ user, loading, logout, refresh: load }}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return ctx;
}
