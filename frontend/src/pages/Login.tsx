import { useState } from 'react';
import { Lock } from 'lucide-react';

interface LoginProps {
  onLogin: (password: string) => Promise<void>;
}

export default function Login({ onLogin }: LoginProps) {
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setIsLoading(true);

    try {
      await onLogin(password);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center p-4">
      <div className="w-full max-w-sm">
        <div className="bg-white dark:bg-gray-800 rounded-2xl shadow-xl p-8">
          <div className="flex justify-center mb-6">
            <div className="w-16 h-16 bg-blue-500 rounded-2xl flex items-center justify-center">
              <Lock className="w-8 h-8 text-white" />
            </div>
          </div>

          <h1 className="text-2xl font-bold text-center mb-2">NAS Controller</h1>
          <p className="text-gray-500 dark:text-gray-400 text-center mb-6">
            Enter your password to continue
          </p>

          <form onSubmit={handleSubmit}>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Password"
              className="w-full px-4 py-3 rounded-lg border border-gray-300 dark:border-gray-600
                       bg-gray-50 dark:bg-gray-700 focus:outline-none focus:ring-2
                       focus:ring-blue-500 mb-4"
              autoFocus
            />

            {error && (
              <p className="text-red-500 text-sm mb-4">{error}</p>
            )}

            <button
              type="submit"
              disabled={isLoading || !password}
              className="w-full py-3 bg-blue-500 hover:bg-blue-600 disabled:bg-blue-300
                       text-white font-medium rounded-lg transition-colors"
            >
              {isLoading ? 'Logging in...' : 'Login'}
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}
