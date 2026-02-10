import { useState, useEffect, useCallback } from 'react';
import { api, App } from '../api/client';

export function useApps() {
  const [apps, setApps] = useState<App[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchApps = useCallback(async () => {
    try {
      setError(null);
      const data = await api.getApps();
      setApps(data || []);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to fetch apps');
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchApps();
    // Poll every 5 seconds
    const interval = setInterval(fetchApps, 5000);
    return () => clearInterval(interval);
  }, [fetchApps]);

  return {
    apps,
    isLoading,
    error,
    refresh: fetchApps,
  };
}
