import { useState, useEffect, useRef } from 'react';
import { X, Download, Trash2, RefreshCw } from 'lucide-react';
import { api } from '../api/client';

interface LogsModalProps {
  appId: string;
  onClose: () => void;
}

export default function LogsModal({ appId, onClose }: LogsModalProps) {
  const [tab, setTab] = useState<'container' | 'build'>('container');
  const [logs, setLogs] = useState('');
  const [isLoading, setIsLoading] = useState(true);
  const [autoScroll, setAutoScroll] = useState(true);
  const logsEndRef = useRef<HTMLDivElement>(null);

  const fetchLogs = async () => {
    setIsLoading(true);
    try {
      if (tab === 'container') {
        const data = await api.getLogs(appId, 200);
        setLogs(data.logs);
      } else {
        const data = await api.getBuildLogs(appId);
        setLogs(data.logs);
      }
    } catch (err) {
      setLogs('Failed to fetch logs');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchLogs();
  }, [appId, tab]);

  useEffect(() => {
    if (autoScroll && logsEndRef.current) {
      logsEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logs, autoScroll]);

  const handleDownload = () => {
    const blob = new Blob([logs], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${appId}-${tab}-logs.txt`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const handleClear = async () => {
    if (!confirm('Clear logs for this app?')) return;
    try {
      await api.clearLogs(appId);
      setLogs('');
    } catch (err) {
      alert('Failed to clear logs');
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
      <div className="bg-white dark:bg-gray-800 rounded-2xl w-full max-w-4xl h-[80vh] flex flex-col">
        <div className="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
          <div className="flex items-center gap-4">
            <h2 className="text-lg font-semibold">Logs</h2>
            <div className="flex bg-gray-100 dark:bg-gray-700 rounded-lg p-1">
              <button
                onClick={() => setTab('container')}
                className={`px-3 py-1 text-sm rounded-md transition-colors ${
                  tab === 'container'
                    ? 'bg-white dark:bg-gray-600 shadow-sm'
                    : 'text-gray-500'
                }`}
              >
                Container
              </button>
              <button
                onClick={() => setTab('build')}
                className={`px-3 py-1 text-sm rounded-md transition-colors ${
                  tab === 'build'
                    ? 'bg-white dark:bg-gray-600 shadow-sm'
                    : 'text-gray-500'
                }`}
              >
                Build
              </button>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={fetchLogs}
              className="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
              title="Refresh"
            >
              <RefreshCw className="w-4 h-4" />
            </button>
            <button
              onClick={handleDownload}
              className="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
              title="Download"
            >
              <Download className="w-4 h-4" />
            </button>
            <button
              onClick={handleClear}
              className="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg text-red-500"
              title="Clear"
            >
              <Trash2 className="w-4 h-4" />
            </button>
            <button
              onClick={onClose}
              className="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        <div className="flex items-center gap-4 px-4 py-2 border-b border-gray-200 dark:border-gray-700">
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={autoScroll}
              onChange={(e) => setAutoScroll(e.target.checked)}
              className="rounded"
            />
            Auto-scroll
          </label>
        </div>

        <div className="flex-1 overflow-auto p-4 bg-gray-900">
          {isLoading ? (
            <div className="flex justify-center py-8">
              <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-white"></div>
            </div>
          ) : logs ? (
            <pre className="log-viewer text-gray-100 whitespace-pre-wrap break-words">
              {logs}
              <div ref={logsEndRef} />
            </pre>
          ) : (
            <p className="text-gray-500 text-center py-8">No logs available</p>
          )}
        </div>
      </div>
    </div>
  );
}
