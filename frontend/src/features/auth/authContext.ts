import { createContext } from 'react';
import type { AuthResponse, User } from './types';

export const authKeys = {
  all: ['auth'] as const,
  me: ['auth', 'me'] as const,
};

export interface AuthContextValue {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  isLoggingOut: boolean;
  sessionExpired: boolean;
  authError: Error | null;
  setSession: (auth: AuthResponse) => void;
  logout: () => void;
  retrySession: () => void;
}

export const AuthContext = createContext<AuthContextValue | null>(null);
