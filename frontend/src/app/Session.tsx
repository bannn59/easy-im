import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import * as authApi from '../api/auth';
import type { PublicUser } from '../api/auth';
import { ApiError } from '../api/http';

const TOKEN_KEY = 'easyim_access_token';

type SessionState = {
  token: string | null;
  user: PublicUser | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  logout: () => void;
  setUser: (u: PublicUser | null) => void;
  refreshUser: () => Promise<void>;
};

const SessionContext = createContext<SessionState | null>(null);

export function SessionProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState<string | null>(() => localStorage.getItem(TOKEN_KEY));
  const [user, setUserState] = useState<PublicUser | null>(null);
  const [loading, setLoading] = useState(Boolean(localStorage.getItem(TOKEN_KEY)));

  const logout = useCallback(() => {
    localStorage.removeItem(TOKEN_KEY);
    setToken(null);
    setUserState(null);
  }, []);

  useEffect(() => {
    if (!token) {
      setUserState(null);
      setLoading(false);
      return;
    }
    let cancelled = false;
    setLoading(true);
    authApi
      .fetchMe(token)
      .then((u) => {
        if (!cancelled) setUserState(u);
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          if (err instanceof ApiError && err.status === 401) {
            logout();
          }
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [token, logout]);

  const applyAuth = useCallback(async (res: authApi.TokenResponse) => {
    localStorage.setItem(TOKEN_KEY, res.access_token);
    setToken(res.access_token);
    setUserState(res.user);
  }, []);

  const setUser = useCallback((u: PublicUser | null) => setUserState(u), []);

  const refreshUser = useCallback(async () => {
    const t = localStorage.getItem(TOKEN_KEY);
    if (!t) return;
    const u = await authApi.fetchMe(t);
    setUserState({ id: u.id, email: u.email, display_name: u.display_name });
  }, []);

  const login = useCallback(
    async (email: string, password: string) => {
      const res = await authApi.login(email, password);
      await applyAuth(res);
    },
    [applyAuth],
  );

  const register = useCallback(
    async (email: string, password: string) => {
      const res = await authApi.register(email, password);
      await applyAuth(res);
    },
    [applyAuth],
  );

  const value = useMemo(
    () => ({ token, user, loading, login, register, logout, setUser, refreshUser }),
    [token, user, loading, login, register, logout, setUser, refreshUser],
  );

  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}

export function useSession(): SessionState {
  const ctx = useContext(SessionContext);
  if (!ctx) {
    throw new Error('useSession must be used within SessionProvider');
  }
  return ctx;
}
