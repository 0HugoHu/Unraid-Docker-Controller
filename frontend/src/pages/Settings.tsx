import { useState, useEffect } from 'react';
import { ArrowLeft, LogOut } from 'lucide-react';
import { api, StorageInfo } from '../api/client';

interface SettingsProps {
  onBack: () => void;
  onLogout: () => void;
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

export default function Settings({ onBack, onLogout }: SettingsProps) {
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [passwordError, setPasswordError] = useState('');
  const [passwordSuccess, setPasswordSuccess] = useState('');
  const [storage, setStorage] = useState<StorageInfo | null>(null);
  const [isPruning, setIsPruning] = useState(false);
  const [isClearingLogs, setIsClearingLogs] = useState(false);

  useEffect(() => {
    api.getStorage().then(setStorage).catch(console.error);
  }, []);

  const handleUpdatePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    setPasswordError('');
    setPasswordSuccess('');

    if (newPassword !== confirmPassword) {
      setPasswordError('Passwords do not match');
      return;
    }

    if (newPassword.length < 8) {
      setPasswordError('Password must be at least 8 characters');
      return;
    }

    try {
      await api.updatePassword(currentPassword, newPassword);
      setPasswordSuccess('Password updated successfully');
      setCurrentPassword('');
      setNewPassword('');
      setConfirmPassword('');
    } catch (err) {
      setPasswordError(err instanceof Error ? err.message : 'Failed to update password');
    }
  };

  const handlePruneImages = async () => {
    setIsPruning(true);
    try {
      const result = await api.pruneImages();
      alert(`Pruned ${formatBytes(result.spaceReclaimed)}`);
      const newStorage = await api.getStorage();
      setStorage(newStorage);
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to prune images');
    } finally {
      setIsPruning(false);
    }
  };

  const handleClearLogs = async () => {
    if (!confirm('Clear all build logs?')) return;
    setIsClearingLogs(true);
    try {
      await api.clearAllLogs();
      const newStorage = await api.getStorage();
      setStorage(newStorage);
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to clear logs');
    } finally {
      setIsClearingLogs(false);
    }
  };

  return (
    <div className="min-h-screen">
      {/* Header */}
      <header className="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 sticky top-0 z-10">
        <div className="max-w-2xl mx-auto px-4 py-4 flex items-center gap-4">
          <button
            onClick={onBack}
            className="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
          >
            <ArrowLeft className="w-5 h-5" />
          </button>
          <h1 className="text-xl font-bold">Settings</h1>
        </div>
      </header>

      <main className="max-w-2xl mx-auto px-4 py-6 space-y-6">
        {/* Password Section */}
        <section className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
          <h2 className="text-lg font-semibold mb-4">Password</h2>
          <form onSubmit={handleUpdatePassword} className="space-y-4">
            <div>
              <label className="block text-sm text-gray-500 dark:text-gray-400 mb-1">
                Current Password
              </label>
              <input
                type="password"
                value={currentPassword}
                onChange={(e) => setCurrentPassword(e.target.value)}
                className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600
                         bg-gray-50 dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <div>
              <label className="block text-sm text-gray-500 dark:text-gray-400 mb-1">
                New Password
              </label>
              <input
                type="password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600
                         bg-gray-50 dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <div>
              <label className="block text-sm text-gray-500 dark:text-gray-400 mb-1">
                Confirm New Password
              </label>
              <input
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600
                         bg-gray-50 dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            {passwordError && (
              <p className="text-red-500 text-sm">{passwordError}</p>
            )}
            {passwordSuccess && (
              <p className="text-green-500 text-sm">{passwordSuccess}</p>
            )}
            <button
              type="submit"
              className="px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white
                       font-medium rounded-lg transition-colors"
            >
              Update Password
            </button>
          </form>
        </section>

        {/* Storage Section */}
        <section className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
          <h2 className="text-lg font-semibold mb-4">Storage</h2>
          {storage && (
            <div className="space-y-3">
              <div className="flex justify-between items-center py-2 border-b border-gray-100 dark:border-gray-700">
                <span className="text-gray-500 dark:text-gray-400">Database</span>
                <span className="font-medium">{formatBytes(storage.database)}</span>
              </div>
              <div className="flex justify-between items-center py-2 border-b border-gray-100 dark:border-gray-700">
                <span className="text-gray-500 dark:text-gray-400">Repositories</span>
                <span className="font-medium">{formatBytes(storage.repositories)}</span>
              </div>
              <div className="flex justify-between items-center py-2 border-b border-gray-100 dark:border-gray-700">
                <div className="flex items-center gap-2">
                  <span className="text-gray-500 dark:text-gray-400">Logs</span>
                </div>
                <div className="flex items-center gap-3">
                  <span className="font-medium">{formatBytes(storage.logs)}</span>
                  <button
                    onClick={handleClearLogs}
                    disabled={isClearingLogs}
                    className="text-red-500 hover:text-red-600 text-sm"
                  >
                    {isClearingLogs ? 'Clearing...' : 'Clear All'}
                  </button>
                </div>
              </div>
              <div className="flex justify-between items-center py-2">
                <div className="flex items-center gap-2">
                  <span className="text-gray-500 dark:text-gray-400">Docker Images</span>
                </div>
                <div className="flex items-center gap-3">
                  <span className="font-medium">{formatBytes(storage.images)}</span>
                  <button
                    onClick={handlePruneImages}
                    disabled={isPruning}
                    className="text-red-500 hover:text-red-600 text-sm"
                  >
                    {isPruning ? 'Pruning...' : 'Prune Unused'}
                  </button>
                </div>
              </div>
            </div>
          )}
        </section>

        {/* Logout */}
        <button
          onClick={onLogout}
          className="w-full flex items-center justify-center gap-2 px-4 py-3
                   bg-red-500 hover:bg-red-600 text-white font-medium rounded-xl transition-colors"
        >
          <LogOut className="w-4 h-4" />
          Logout
        </button>
      </main>
    </div>
  );
}
