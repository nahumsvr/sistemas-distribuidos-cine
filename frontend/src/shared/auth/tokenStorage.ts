export const TOKEN_KEY = 'cine_auth_token';

type TokenListener = (token: string | null) => void;

const listeners = new Set<TokenListener>();

function notify(token: string | null): void {
  for (const listener of listeners) {
    listener(token);
  }
}

/**
 * Single point of access for the JWT stored in localStorage (see ADR-001).
 * No other module should read or write auth tokens in localStorage directly.
 */
export const tokenStorage = {
  get(): string | null {
    try {
      return window.localStorage.getItem(TOKEN_KEY);
    } catch {
      return null;
    }
  },

  set(token: string): void {
    try {
      window.localStorage.setItem(TOKEN_KEY, token);
    } finally {
      notify(token);
    }
  },

  clear(): void {
    try {
      window.localStorage.removeItem(TOKEN_KEY);
    } finally {
      notify(null);
    }
  },

  subscribe(listener: TokenListener): () => void {
    listeners.add(listener);
    return () => {
      listeners.delete(listener);
    };
  },
};
