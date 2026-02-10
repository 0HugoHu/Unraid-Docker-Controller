import { useState, useEffect } from 'react';
import { useAuth } from './hooks/useAuth';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import Settings from './pages/Settings';

type Page = 'dashboard' | 'settings';

export default function App() {
  const { isAuthenticated, isLoading, login, logout } = useAuth();
  const [currentPage, setCurrentPage] = useState<Page>('dashboard');
  const [isDark, setIsDark] = useState(() => {
    if (typeof window !== 'undefined') {
      return window.matchMedia('(prefers-color-scheme: dark)').matches;
    }
    return false;
  });

  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handler = (e: MediaQueryListEvent) => setIsDark(e.matches);
    mediaQuery.addEventListener('change', handler);
    return () => mediaQuery.removeEventListener('change', handler);
  }, []);

  useEffect(() => {
    document.documentElement.classList.toggle('dark', isDark);
  }, [isDark]);

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Login onLogin={login} />;
  }

  return (
    <div className="min-h-screen">
      {currentPage === 'dashboard' && (
        <Dashboard
          onSettings={() => setCurrentPage('settings')}
          onLogout={logout}
        />
      )}
      {currentPage === 'settings' && (
        <Settings
          onBack={() => setCurrentPage('dashboard')}
          onLogout={logout}
        />
      )}
    </div>
  );
}
