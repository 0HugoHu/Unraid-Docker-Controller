import { useState, useEffect } from 'react';
import { X, Loader2 } from 'lucide-react';
import { api, App } from '../api/client';

interface ConfigModalProps {
  appId: string;
  onClose: () => void;
  onSave: () => void;
}

export default function ConfigModal({ appId, onClose, onSave }: ConfigModalProps) {
  const [_app, setApp] = useState<App | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState('');

  const [name, setName] = useState('');
  const [internalPort, setInternalPort] = useState(80);
  const [externalPort, setExternalPort] = useState(0);
  const [envVars, setEnvVars] = useState<{ key: string; value: string }[]>([]);
  const [volumes, setVolumes] = useState<{ host: string; container: string }[]>([]);

  useEffect(() => {
    const fetchApp = async () => {
      try {
        const data = await api.getApp(appId);
        setApp(data.app);
        setName(data.app.name);
        setInternalPort(data.app.internalPort);
        setExternalPort(data.app.externalPort);
        setEnvVars(
          Object.entries(data.app.env || {}).map(([key, value]) => ({
            key,
            value,
          }))
        );
        setVolumes(
          (data.app.volumes || []).map((v) => {
            const [host, container] = v.split(':');
            return { host, container: container || '' };
          })
        );
      } catch (err) {
        setError('Failed to load app');
      } finally {
        setIsLoading(false);
      }
    };
    fetchApp();
  }, [appId]);

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setIsSaving(true);

    try {
      const env: Record<string, string> = {};
      envVars.forEach(({ key, value }) => {
        if (key.trim()) {
          env[key.trim()] = value;
        }
      });

      const volumeStrings = volumes
        .filter((v) => v.host.trim() && v.container.trim())
        .map((v) => `${v.host.trim()}:${v.container.trim()}`);

      await api.updateApp(appId, {
        name,
        internalPort,
        externalPort,
        env,
        volumes: volumeStrings,
      });

      onSave();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save');
    } finally {
      setIsSaving(false);
    }
  };

  const addEnvVar = () => {
    setEnvVars([...envVars, { key: '', value: '' }]);
  };

  const updateEnvVar = (index: number, field: 'key' | 'value', value: string) => {
    const updated = [...envVars];
    updated[index][field] = value;
    setEnvVars(updated);
  };

  const removeEnvVar = (index: number) => {
    setEnvVars(envVars.filter((_, i) => i !== index));
  };

  const addVolume = () => {
    setVolumes([...volumes, { host: '', container: '' }]);
  };

  const updateVolume = (index: number, field: 'host' | 'container', value: string) => {
    const updated = [...volumes];
    updated[index][field] = value;
    setVolumes(updated);
  };

  const removeVolume = (index: number) => {
    setVolumes(volumes.filter((_, i) => i !== index));
  };

  if (isLoading) {
    return (
      <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
        <div className="bg-white dark:bg-gray-800 rounded-2xl p-8">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-gray-400"></div>
        </div>
      </div>
    );
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
      <div className="bg-white dark:bg-gray-800 rounded-2xl w-full max-w-lg max-h-[90vh] overflow-auto border border-gray-200 dark:border-gray-700">
        <div className="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-lg font-semibold">Configure App</h2>
          <button
            onClick={onClose}
            className="p-1 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg text-gray-400"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <form onSubmit={handleSave} className="p-4 space-y-4">
          <div>
            <label className="block text-sm text-gray-500 dark:text-gray-400 mb-1">
              App Name
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600
                       bg-gray-50 dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-gray-400"
              required
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-gray-500 dark:text-gray-400 mb-1">
                Internal Port
              </label>
              <input
                type="number"
                value={internalPort}
                onChange={(e) => setInternalPort(parseInt(e.target.value) || 80)}
                className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600
                         bg-gray-50 dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-gray-400"
                required
              />
            </div>
            <div>
              <label className="block text-sm text-gray-500 dark:text-gray-400 mb-1">
                External Port
              </label>
              <input
                type="number"
                value={externalPort}
                onChange={(e) => setExternalPort(parseInt(e.target.value) || 13001)}
                className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600
                         bg-gray-50 dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-gray-400"
                required
              />
            </div>
          </div>

          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-sm text-gray-500 dark:text-gray-400">
                Environment Variables
              </label>
              <button
                type="button"
                onClick={addEnvVar}
                className="text-sm text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
              >
                + Add
              </button>
            </div>
            {envVars.length > 0 ? (
              <div className="space-y-2">
                {envVars.map((env, index) => (
                  <div key={index} className="flex gap-2">
                    <input
                      type="text"
                      value={env.key}
                      onChange={(e) => updateEnvVar(index, 'key', e.target.value)}
                      placeholder="KEY"
                      className="flex-1 px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600
                               bg-gray-50 dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-gray-400 text-sm"
                    />
                    <input
                      type="text"
                      value={env.value}
                      onChange={(e) => updateEnvVar(index, 'value', e.target.value)}
                      placeholder="value"
                      className="flex-1 px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600
                               bg-gray-50 dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-gray-400 text-sm"
                    />
                    <button
                      type="button"
                      onClick={() => removeEnvVar(index)}
                      className="px-2 text-gray-400 hover:text-red-600 dark:hover:text-red-400 rounded transition-colors"
                    >
                      <X className="w-4 h-4" />
                    </button>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm text-gray-400">No environment variables</p>
            )}
          </div>

          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-sm text-gray-500 dark:text-gray-400">
                Volume Mounts
              </label>
              <button
                type="button"
                onClick={addVolume}
                className="text-sm text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
              >
                + Add
              </button>
            </div>
            {volumes.length > 0 ? (
              <div className="space-y-2">
                {volumes.map((vol, index) => (
                  <div key={index} className="flex gap-2">
                    <input
                      type="text"
                      value={vol.host}
                      onChange={(e) => updateVolume(index, 'host', e.target.value)}
                      placeholder="Host path"
                      className="flex-1 px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600
                               bg-gray-50 dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-gray-400 text-sm"
                    />
                    <input
                      type="text"
                      value={vol.container}
                      onChange={(e) => updateVolume(index, 'container', e.target.value)}
                      placeholder="Container path"
                      className="flex-1 px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600
                               bg-gray-50 dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-gray-400 text-sm"
                    />
                    <button
                      type="button"
                      onClick={() => removeVolume(index)}
                      className="px-2 text-gray-400 hover:text-red-600 dark:hover:text-red-400 rounded transition-colors"
                    >
                      <X className="w-4 h-4" />
                    </button>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm text-gray-400">No volume mounts</p>
            )}
          </div>

          {error && (
            <p className="text-red-600 dark:text-red-400 text-sm">{error}</p>
          )}

          <div className="flex gap-3 pt-4">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 py-2 border border-gray-300 dark:border-gray-600
                       text-gray-700 dark:text-gray-300 font-medium rounded-lg
                       hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSaving}
              className="flex-1 py-2 bg-gray-900 hover:bg-gray-800 disabled:bg-gray-300
                       dark:bg-white dark:hover:bg-gray-100 dark:disabled:bg-gray-600
                       text-white dark:text-gray-900 font-medium rounded-lg transition-colors
                       flex items-center justify-center gap-2"
            >
              {isSaving ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin" />
                  Saving...
                </>
              ) : (
                'Save Changes'
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
