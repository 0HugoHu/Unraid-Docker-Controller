import { useState, useEffect } from 'react';
import { Settings, Plus, Database, FolderGit2, FileText, HardDrive } from 'lucide-react';
import { useApps } from '../hooks/useApps';
import { api, StorageInfo } from '../api/client';
import AppCard from '../components/AppCard';
import AddAppModal from '../components/AddAppModal';
import LogsModal from '../components/LogsModal';
import ConfigModal from '../components/ConfigModal';

interface DashboardProps {
  onSettings: () => void;
  onLogout: () => void;
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

export default function Dashboard({ onSettings }: DashboardProps) {
  const { apps, isLoading, refresh } = useApps();
  const [storage, setStorage] = useState<StorageInfo | null>(null);
  const [showAddModal, setShowAddModal] = useState(false);
  const [logsAppId, setLogsAppId] = useState<string | null>(null);
  const [configAppId, setConfigAppId] = useState<string | null>(null);

  useEffect(() => {
    const fetchStorage = async () => {
      try {
        const data = await api.getStorage();
        setStorage(data);
      } catch (e) {
        console.error('Failed to fetch storage:', e);
      }
    };
    fetchStorage();
    const interval = setInterval(fetchStorage, 30000);
    return () => clearInterval(interval);
  }, []);

  const runningCount = apps.filter((a) => a.status === 'running').length;

  return (
    <div className="min-h-screen">
      {/* Header */}
      <header className="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 sticky top-0 z-10">
        <div className="max-w-6xl mx-auto px-4 py-4 flex items-center justify-between">
          <h1 className="text-xl font-bold">NAS Controller</h1>
          <button
            onClick={onSettings}
            className="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
          >
            <Settings className="w-5 h-5" />
          </button>
        </div>
      </header>

      <main className="max-w-6xl mx-auto px-4 py-6">
        {/* Storage Bar */}
        {storage && (
          <div className="bg-white dark:bg-gray-800 rounded-xl p-4 mb-6 shadow-sm border border-gray-200 dark:border-gray-700">
            <div className="flex flex-wrap gap-4 text-sm">
              <div className="flex items-center gap-2">
                <Database className="w-4 h-4 text-blue-500" />
                <span className="text-gray-500 dark:text-gray-400">DB:</span>
                <span className="font-medium">{formatBytes(storage.database)}</span>
              </div>
              <div className="flex items-center gap-2">
                <FolderGit2 className="w-4 h-4 text-green-500" />
                <span className="text-gray-500 dark:text-gray-400">Repos:</span>
                <span className="font-medium">{formatBytes(storage.repositories)}</span>
              </div>
              <div className="flex items-center gap-2">
                <FileText className="w-4 h-4 text-yellow-500" />
                <span className="text-gray-500 dark:text-gray-400">Logs:</span>
                <span className="font-medium">{formatBytes(storage.logs)}</span>
              </div>
              <div className="flex items-center gap-2">
                <HardDrive className="w-4 h-4 text-purple-500" />
                <span className="text-gray-500 dark:text-gray-400">Images:</span>
                <span className="font-medium">{formatBytes(storage.images)}</span>
              </div>
            </div>
          </div>
        )}

        {/* Actions Bar */}
        <div className="flex items-center justify-between mb-6">
          <button
            onClick={() => setShowAddModal(true)}
            className="flex items-center gap-2 px-4 py-2 bg-blue-500 hover:bg-blue-600
                     text-white font-medium rounded-lg transition-colors"
          >
            <Plus className="w-4 h-4" />
            Add New App
          </button>
          <span className="text-sm text-gray-500 dark:text-gray-400">
            {apps.length} apps, {runningCount} running
          </span>
        </div>

        {/* Apps List */}
        {isLoading ? (
          <div className="flex justify-center py-12">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
          </div>
        ) : apps.length === 0 ? (
          <div className="text-center py-12">
            <p className="text-gray-500 dark:text-gray-400 mb-4">No apps yet</p>
            <button
              onClick={() => setShowAddModal(true)}
              className="text-blue-500 hover:underline"
            >
              Add your first app
            </button>
          </div>
        ) : (
          <div className="space-y-4">
            {apps.map((app) => (
              <AppCard
                key={app.id}
                app={app}
                onRefresh={refresh}
                onShowLogs={() => setLogsAppId(app.id)}
                onShowConfig={() => setConfigAppId(app.id)}
              />
            ))}
          </div>
        )}
      </main>

      {/* Modals */}
      {showAddModal && (
        <AddAppModal
          onClose={() => setShowAddModal(false)}
          onSuccess={() => {
            setShowAddModal(false);
            refresh();
          }}
        />
      )}

      {logsAppId && (
        <LogsModal
          appId={logsAppId}
          onClose={() => setLogsAppId(null)}
        />
      )}

      {configAppId && (
        <ConfigModal
          appId={configAppId}
          onClose={() => setConfigAppId(null)}
          onSave={() => {
            setConfigAppId(null);
            refresh();
          }}
        />
      )}
    </div>
  );
}
