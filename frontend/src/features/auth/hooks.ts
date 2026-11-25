import { useMutation } from '@tanstack/react-query';
import { useContext } from 'react';
import { AuthContext, type AuthContextValue } from './authContext';
import { authApi } from './api';
import type { LoginInput, RegisterInput } from './types';

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}

export function useLoginMutation() {
  const { setSession } = useAuth();
  return useMutation({
    mutationFn: (input: LoginInput) => authApi.login(input),
    onSuccess: (auth) => {
      setSession(auth);
    },
  });
}

export function useRegisterMutation() {
  const { setSession } = useAuth();
  return useMutation({
    mutationFn: (input: RegisterInput) => authApi.register(input),
    onSuccess: (auth) => {
      setSession(auth);
    },
  });
}
