import { useState, useEffect, useCallback } from 'react';
import { api } from '../api/client';

export function useAuth() {
  const [isAuthenticated, setIsAuthenticated] = useState<boolean | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const checkAuth = useCallback(async () => {
    try {
      const result = await api.checkAuth();
      setIsAuthenticated(result.authenticated);
    } catch {
      setIsAuthenticated(false);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    checkAuth();
  }, [checkAuth]);

  const login = async (password: string) => {
    await api.login(password);
    setIsAuthenticated(true);
  };

  const logout = async () => {
    await api.logout();
    setIsAuthenticated(false);
  };

  return {
    isAuthenticated,
    isLoading,
    login,
    logout,
    checkAuth,
  };
}
