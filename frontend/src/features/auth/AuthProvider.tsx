import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { setUnauthorizedHandler } from '../../shared/api/client';
import { TOKEN_KEY, tokenStorage } from '../../shared/auth/tokenStorage';
import { authApi } from './api';
import { AuthContext, authKeys, type AuthContextValue } from './authContext';
import type { AuthLocationState, AuthResponse } from './types';

export interface AuthProviderProps {
  children: ReactNode;
}

export function AuthProvider({ children }: AuthProviderProps) {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const location = useLocation();
  const [token, setToken] = useState<string | null>(() => tokenStorage.get());
  const [sessionExpired, setSessionExpired] = useState(false);
  const [isLoggingOut, setIsLoggingOut] = useState(false);

  const locationRef = useRef(location);
  useEffect(() => {
    locationRef.current = location;
  }, [location]);

  // Clear the logout transition flag once navigation to the public landing route commits.
  useEffect(() => {
    if (isLoggingOut && location.pathname === '/') {
      setIsLoggingOut(false);
    }
  }, [isLoggingOut, location.pathname]);

  // Stay in sync with tokenStorage mutations in this tab and cross-tab localStorage updates.
  useEffect(() => {
    const unsubscribe = tokenStorage.subscribe((nextToken) => {
      setToken(nextToken);
    });

    const handleStorage = (event: StorageEvent) => {
      if (event.key !== null && event.key !== TOKEN_KEY) {
        return;
      }
      const nextToken = tokenStorage.get();
      setToken(nextToken);
      if (!nextToken) {
        queryClient.clear();
      }
    };

    window.addEventListener('storage', handleStorage);
    return () => {
      unsubscribe();
      window.removeEventListener('storage', handleStorage);
    };
  }, [queryClient]);

  // Register global 401 handler on apiClient: token is already cleared by client.ts.
  useEffect(() => {
    setUnauthorizedHandler(() => {
      queryClient.clear();
      setSessionExpired(true);
      const current = locationRef.current;
      if (current.pathname === '/login') {
        return;
      }
      const from = `${current.pathname}${current.search}`;
      const state: AuthLocationState = {
        from: current.pathname === '/register' ? '/' : from,
        sessionExpired: true,
      };
      navigate('/login', { replace: true, state });
    });

    return () => {
      setUnauthorizedHandler(null);
    };
  }, [navigate, queryClient]);

  const meQuery = useQuery({
    queryKey: authKeys.me,
    queryFn: ({ signal }) => authApi.me(signal),
    enabled: Boolean(token),
    staleTime: 5 * 60_000,
  });

  const setSession = useCallback(
    (auth: AuthResponse) => {
      setSessionExpired(false);
      setIsLoggingOut(false);
      tokenStorage.set(auth.token);
      queryClient.setQueryData(authKeys.me, auth.user);
    },
    [queryClient],
  );

  const logout = useCallback(() => {
    setSessionExpired(false);
    setIsLoggingOut(true);
    tokenStorage.clear();
    queryClient.clear();
    navigate('/', { replace: true });
  }, [navigate, queryClient]);

  const retrySession = useCallback(() => {
    void meQuery.refetch();
  }, [meQuery]);

  const user = token ? (meQuery.data ?? null) : null;
  const isLoading = Boolean(token) && meQuery.isPending;
  const { isError, error } = meQuery;

  const value = useMemo<AuthContextValue>(() => {
    const authError =
      token && isError
        ? error instanceof Error
          ? error
          : new Error('Unable to verify session')
        : null;

    return {
      user,
      token,
      isAuthenticated: Boolean(token && user),
      isLoading,
      isLoggingOut,
      sessionExpired,
      authError,
      setSession,
      logout,
      retrySession,
    };
  }, [
    user,
    token,
    isLoading,
    isLoggingOut,
    sessionExpired,
    isError,
    error,
    setSession,
    logout,
    retrySession,
  ]);

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
