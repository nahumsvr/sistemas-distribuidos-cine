import { beforeEach, describe, expect, it, vi } from 'vitest';
import { tokenStorage } from './tokenStorage';

describe('tokenStorage', () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it('returns null when no token is stored', () => {
    expect(tokenStorage.get()).toBeNull();
  });

  it('stores, reads and clears a token', () => {
    tokenStorage.set('jwt-token-123');
    expect(tokenStorage.get()).toBe('jwt-token-123');

    tokenStorage.clear();
    expect(tokenStorage.get()).toBeNull();
  });

  it('notifies subscribers when the token changes', () => {
    const listener = vi.fn();
    const unsubscribe = tokenStorage.subscribe(listener);

    tokenStorage.set('abc');
    expect(listener).toHaveBeenCalledWith('abc');

    tokenStorage.clear();
    expect(listener).toHaveBeenCalledWith(null);

    unsubscribe();
    tokenStorage.set('def');
    expect(listener).toHaveBeenCalledTimes(2);
  });
});
