import { useState } from 'react';
import {
  Play,
  Square,
  RefreshCw,
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
      return 'text-green-500';
    case 'building':
    case 'starting':
      return 'text-yellow-500';
    case 'build-failed':
    case 'error':
      return 'text-red-500';
    default:
      return 'text-gray-400';
  }
}

function getStatusIcon(status: App['status']): string {
  switch (status) {
    case 'running':
      return '●';
    case 'building':
    case 'starting':
      return '◐';
    case 'build-failed':
    case 'error':
      return '✕';
    default:
      return '○';
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

  const repoShort = app.repoUrl.replace('https://github.com/', '');
  const isRunning = app.status === 'running';
  const isBuilding = app.status === 'building';
  const canStart = app.status === 'stopped' || app.status === 'build-failed';

  return (
    <div className="bg-white dark:bg-gray-800 rounded-xl p-4 shadow-sm border border-gray-200 dark:border-gray-700">
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-start gap-3 flex-1 min-w-0">
          <div className="w-10 h-10 bg-gray-100 dark:bg-gray-700 rounded-lg flex items-center justify-center flex-shrink-0">
            <Box className="w-5 h-5 text-gray-500" />
          </div>
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <h3 className="font-semibold truncate">{app.name}</h3>
              <span className={`text-sm ${getStatusColor(app.status)}`}>
                {getStatusIcon(app.status)} {app.status}
              </span>
            </div>
            <div className="flex items-center gap-3 text-sm text-gray-500 dark:text-gray-400 mt-1">
              <span className="truncate flex items-center gap-1">
                <GitBranch className="w-3 h-3" />
                {repoShort}
              </span>
              <span>:{app.externalPort}</span>
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
      <div className="flex flex-wrap gap-2 mt-4 pt-4 border-t border-gray-100 dark:border-gray-700">
        {isRunning && (
          <a
            href={`http://localhost:${app.externalPort}`}
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-1 px-3 py-1.5 text-sm bg-blue-50 dark:bg-blue-900/30
                     text-blue-600 dark:text-blue-400 rounded-lg hover:bg-blue-100 dark:hover:bg-blue-900/50"
          >
            <ExternalLink className="w-3 h-3" />
            Open
          </a>
        )}

        {isRunning && (
          <button
            onClick={() => handleAction('stop', () => api.stopApp(app.id))}
            disabled={isLoading}
            className="flex items-center gap-1 px-3 py-1.5 text-sm bg-gray-100 dark:bg-gray-700
                     text-gray-600 dark:text-gray-300 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600"
          >
            <Square className="w-3 h-3" />
            {loadingAction === 'stop' ? 'Stopping...' : 'Stop'}
          </button>
        )}

        {canStart && (
          <button
            onClick={() => handleAction('start', () => api.startApp(app.id))}
            disabled={isLoading}
            className="flex items-center gap-1 px-3 py-1.5 text-sm bg-green-50 dark:bg-green-900/30
                     text-green-600 dark:text-green-400 rounded-lg hover:bg-green-100 dark:hover:bg-green-900/50"
          >
            <Play className="w-3 h-3" />
            {loadingAction === 'start' ? 'Starting...' : 'Start'}
          </button>
        )}

        {!isBuilding && (
          <button
            onClick={() => handleAction('rebuild', () => api.pullAndRebuild(app.id))}
            disabled={isLoading}
            className="flex items-center gap-1 px-3 py-1.5 text-sm bg-gray-100 dark:bg-gray-700
                     text-gray-600 dark:text-gray-300 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600"
          >
            <RefreshCw className="w-3 h-3" />
            {loadingAction === 'rebuild' ? 'Rebuilding...' : 'Rebuild'}
          </button>
        )}

        <button
          onClick={onShowLogs}
          className="flex items-center gap-1 px-3 py-1.5 text-sm bg-gray-100 dark:bg-gray-700
                   text-gray-600 dark:text-gray-300 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600"
        >
          <FileText className="w-3 h-3" />
          Logs
        </button>

        <button
          onClick={onShowConfig}
          className="flex items-center gap-1 px-3 py-1.5 text-sm bg-gray-100 dark:bg-gray-700
                   text-gray-600 dark:text-gray-300 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600"
        >
          <Settings className="w-3 h-3" />
          Config
        </button>

        <button
          onClick={handleDelete}
          disabled={isLoading}
          className="flex items-center gap-1 px-3 py-1.5 text-sm bg-red-50 dark:bg-red-900/30
                   text-red-600 dark:text-red-400 rounded-lg hover:bg-red-100 dark:hover:bg-red-900/50"
        >
          <Trash2 className="w-3 h-3" />
          {loadingAction === 'delete' ? 'Deleting...' : 'Delete'}
        </button>
      </div>
    </div>
  );
}
