import { useState } from 'react';
import {
  Play,
  Square,
  RefreshCw,
  Download,
  FileText,
  Settings,
  Trash2,
  ExternalLink,
  Box,
  GitBranch,
} from 'lucide-react';
import { api, App } from '../api/client';

interface AppCardProps {
  app: App;
  onRefresh: () => void;
  onShowLogs: () => void;
  onShowConfig: () => void;
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

function getStatusColor(status: App['status']): string {
  switch (status) {
    case 'running':
      return 'text-emerald-600 dark:text-emerald-400';
    case 'building':
    case 'starting':
      return 'text-amber-600 dark:text-amber-400';
    case 'build-failed':
    case 'error':
      return 'text-red-600 dark:text-red-400';
    default:
      return 'text-gray-400';
  }
}

function getStatusDot(status: App['status']): string {
  switch (status) {
    case 'running':
      return 'bg-emerald-500';
    case 'building':
    case 'starting':
      return 'bg-amber-500';
    case 'build-failed':
    case 'error':
      return 'bg-red-500';
    default:
      return 'bg-gray-400';
  }
}

export default function AppCard({ app, onRefresh, onShowLogs, onShowConfig }: AppCardProps) {
  const [isLoading, setIsLoading] = useState(false);
  const [loadingAction, setLoadingAction] = useState<string | null>(null);

  const handleAction = async (action: string, fn: () => Promise<any>) => {
    setIsLoading(true);
    setLoadingAction(action);
    try {
      await fn();
      onRefresh();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Action failed');
    } finally {
      setIsLoading(false);
      setLoadingAction(null);
    }
  };

  const handleDelete = async () => {
    if (!confirm(`Delete ${app.name}? This will stop the container and remove all data.`)) {
      return;
    }
    await handleAction('delete', () => api.deleteApp(app.id));
  };

  const handleUpdate = async () => {
    setIsLoading(true);
    setLoadingAction('update');
    try {
      const result = await api.checkAppUpdate(app.id);
      if (!result.hasUpdate) {
        alert(`Already up to date (${result.localCommit})`);
        return;
      }
      if (!confirm(`Update available: ${result.localCommit} \u2192 ${result.remoteCommit}. Pull and rebuild?`)) {
        return;
      }
      await api.pullAndRebuild(app.id);
      onRefresh();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Update check failed');
    } finally {
      setIsLoading(false);
      setLoadingAction(null);
    }
  };

  const repoShort = app.repoUrl.replace('https://github.com/', '');
  const isRunning = app.status === 'running';
  const isBuilding = app.status === 'building';
  const canStart = app.status === 'stopped' || app.status === 'build-failed';

  return (
    <div className="bg-white dark:bg-gray-800 rounded-xl p-4 shadow-sm border border-gray-200 dark:border-gray-700">
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-start gap-3 flex-1 min-w-0">
          <div className="w-10 h-10 bg-gray-100 dark:bg-gray-700 rounded-lg flex items-center justify-center flex-shrink-0">
            <Box className="w-5 h-5 text-gray-400" />
          </div>
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <h3 className="font-semibold truncate">{app.name}</h3>
              <span className={`flex items-center gap-1.5 text-xs font-medium ${getStatusColor(app.status)}`}>
                <span className={`w-1.5 h-1.5 rounded-full ${getStatusDot(app.status)}`} />
                {app.status}
              </span>
            </div>
            <div className="flex items-center gap-3 text-sm text-gray-500 dark:text-gray-400 mt-1">
              <span className="truncate flex items-center gap-1">
                <GitBranch className="w-3 h-3" />
                {repoShort}
              </span>
              <span className="text-gray-400">:{app.externalPort}</span>
            </div>
            <div className="flex items-center gap-3 text-xs text-gray-400 mt-1">
              {app.imageSize > 0 && <span>{formatBytes(app.imageSize)}</span>}
              {isRunning && app.lastBuildDuration && (
                <span>Uptime: {app.lastBuildDuration}</span>
              )}
              {app.lastCommit && <span>#{app.lastCommit}</span>}
            </div>
          </div>
        </div>
      </div>

      {/* Actions */}
      <div className="flex flex-wrap gap-1.5 mt-4 pt-4 border-t border-gray-100 dark:border-gray-700">
        {isRunning && (
          <a
            href={`http://localhost:${app.externalPort}`}
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-1 px-3 py-1.5 text-sm
                     text-gray-600 dark:text-gray-300 rounded-lg
                     hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
          >
            <ExternalLink className="w-3 h-3" />
            Open
          </a>
        )}

        {isRunning && (
          <button
            onClick={() => handleAction('stop', () => api.stopApp(app.id))}
            disabled={isLoading}
            className="flex items-center gap-1 px-3 py-1.5 text-sm
                     text-gray-600 dark:text-gray-300 rounded-lg
                     hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors
                     disabled:opacity-50"
          >
            <Square className="w-3 h-3" />
            {loadingAction === 'stop' ? 'Stopping...' : 'Stop'}
          </button>
        )}

        {canStart && (
          <button
            onClick={() => handleAction('start', () => api.startApp(app.id))}
            disabled={isLoading}
            className="flex items-center gap-1 px-3 py-1.5 text-sm font-medium
                     text-emerald-700 dark:text-emerald-400 rounded-lg
                     hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors
                     disabled:opacity-50"
          >
            <Play className="w-3 h-3" />
            {loadingAction === 'start' ? 'Starting...' : 'Start'}
          </button>
        )}

        {!isBuilding && (
          <button
            onClick={() => handleAction('rebuild', () => api.buildApp(app.id))}
            disabled={isLoading}
            className="flex items-center gap-1 px-3 py-1.5 text-sm
                     text-gray-600 dark:text-gray-300 rounded-lg
                     hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors
                     disabled:opacity-50"
          >
            <RefreshCw className="w-3 h-3" />
            {loadingAction === 'rebuild' ? 'Rebuilding...' : 'Rebuild'}
          </button>
        )}

        {!isBuilding && (
          <button
            onClick={handleUpdate}
            disabled={isLoading}
            className="flex items-center gap-1 px-3 py-1.5 text-sm
                     text-gray-600 dark:text-gray-300 rounded-lg
                     hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors
                     disabled:opacity-50"
          >
            <Download className="w-3 h-3" />
            {loadingAction === 'update' ? 'Checking...' : 'Update'}
          </button>
        )}

        <button
          onClick={onShowLogs}
          className="flex items-center gap-1 px-3 py-1.5 text-sm
                   text-gray-600 dark:text-gray-300 rounded-lg
                   hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
        >
          <FileText className="w-3 h-3" />
          Logs
        </button>

        <button
          onClick={onShowConfig}
          className="flex items-center gap-1 px-3 py-1.5 text-sm
                   text-gray-600 dark:text-gray-300 rounded-lg
                   hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
        >
          <Settings className="w-3 h-3" />
          Config
        </button>

        <button
          onClick={handleDelete}
          disabled={isLoading}
          className="flex items-center gap-1 px-3 py-1.5 text-sm
                   text-gray-400 dark:text-gray-500 rounded-lg
                   hover:bg-gray-100 dark:hover:bg-gray-700
                   hover:text-red-600 dark:hover:text-red-400 transition-colors
                   disabled:opacity-50"
        >
          <Trash2 className="w-3 h-3" />
          {loadingAction === 'delete' ? 'Deleting...' : 'Delete'}
        </button>
      </div>
    </div>
  );
}
