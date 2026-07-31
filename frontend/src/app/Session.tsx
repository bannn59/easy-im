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

type SessionState = {
  user: PublicUser | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  setUser: (u: PublicUser | null) => void;
  refreshUser: () => Promise<void>;
};

const SessionContext = createContext<SessionState | null>(null);

export function SessionProvider({ children }: { children: ReactNode }) {
  const [user, setUserState] = useState<PublicUser | null>(null);
  const [loading, setLoading] = useState(true);

  const logout = useCallback(async () => {
    try {
      await authApi.logout();
    } catch {
      // cookie may already be gone; local state clears regardless
    }
    setUserState(null);
  }, []);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    authApi
      .fetchMe()
      .then((u) => {
        if (!cancelled) setUserState({ id: u.id, email: u.email, display_name: u.display_name });
      })
      .catch(() => {
        // not logged in (no/invalid cookie) — user stays null
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const setUser = useCallback((u: PublicUser | null) => setUserState(u), []);

  const refreshUser = useCallback(async () => {
    const u = await authApi.fetchMe();
    setUserState({ id: u.id, email: u.email, display_name: u.display_name });
  }, []);

  const applyAuth = useCallback(async (res: authApi.TokenResponse) => {
    setUserState(res.user);
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
    () => ({ user, loading, login, register, logout, setUser, refreshUser }),
    [user, loading, login, register, logout, setUser, refreshUser],
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
