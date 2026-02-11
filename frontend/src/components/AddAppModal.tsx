import { useState } from 'react';
import { X, GitBranch, Loader2 } from 'lucide-react';
import { api, CloneResult } from '../api/client';

interface AddAppModalProps {
  onClose: () => void;
  onSuccess: () => void;
}

export default function AddAppModal({ onClose, onSuccess }: AddAppModalProps) {
  const [step, setStep] = useState<'url' | 'configure'>('url');
  const [repoUrl, setRepoUrl] = useState('');
  const [branch, setBranch] = useState('main');
  const [isCloning, setIsCloning] = useState(false);
  const [isCreating, setIsCreating] = useState(false);
  const [error, setError] = useState('');
  const [cloneResult, setCloneResult] = useState<CloneResult | null>(null);

  // Config state
  const [name, setName] = useState('');
  const [internalPort, setInternalPort] = useState(80);
  const [envVars, setEnvVars] = useState<{ key: string; value: string }[]>([]);

  const handleClone = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setIsCloning(true);

    try {
      const result = await api.cloneRepo(repoUrl, branch);
      setCloneResult(result);
      setName(result.name);
      if (result.suggestedPort) {
        setInternalPort(result.suggestedPort);
      }
      if (result.manifest?.env) {
        setEnvVars(
          Object.entries(result.manifest.env).map(([key, value]) => ({
            key,
            value,
          }))
        );
      }
      setStep('configure');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to clone repository');
    } finally {
      setIsCloning(false);
    }
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setIsCreating(true);

    try {
      const env: Record<string, string> = {};
      envVars.forEach(({ key, value }) => {
        if (key.trim()) {
          env[key.trim()] = value;
        }
      });

      await api.createApp(repoUrl, branch, {
        name,
        internalPort,
        env,
      });

      onSuccess();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create app');
    } finally {
      setIsCreating(false);
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

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
      <div className="bg-white dark:bg-gray-800 rounded-2xl w-full max-w-lg max-h-[90vh] overflow-auto border border-gray-200 dark:border-gray-700">
        <div className="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-lg font-semibold">Add New App</h2>
          <button
            onClick={onClose}
            className="p-1 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg text-gray-400"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="p-4">
          {step === 'url' && (
            <form onSubmit={handleClone} className="space-y-4">
              <div>
                <label className="block text-sm text-gray-500 dark:text-gray-400 mb-1">
                  GitHub URL
                </label>
                <input
                  type="url"
                  value={repoUrl}
                  onChange={(e) => setRepoUrl(e.target.value)}
                  placeholder="https://github.com/username/repo"
                  className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600
                           bg-gray-50 dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-gray-400"
                  required
                />
              </div>

              <div>
                <label className="block text-sm text-gray-500 dark:text-gray-400 mb-1">
                  Branch
                </label>
                <div className="relative">
                  <GitBranch className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
                  <input
                    type="text"
                    value={branch}
                    onChange={(e) => setBranch(e.target.value)}
                    placeholder="main"
                    className="w-full pl-10 pr-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600
                             bg-gray-50 dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-gray-400"
                    required
                  />
                </div>
              </div>

              {error && (
                <p className="text-red-600 dark:text-red-400 text-sm">{error}</p>
              )}

              <button
                type="submit"
                disabled={isCloning || !repoUrl || !branch}
                className="w-full py-2 bg-gray-900 hover:bg-gray-800 disabled:bg-gray-300
                         dark:bg-white dark:hover:bg-gray-100 dark:disabled:bg-gray-600
                         text-white dark:text-gray-900 font-medium rounded-lg transition-colors
                         flex items-center justify-center gap-2"
              >
                {isCloning ? (
                  <>
                    <Loader2 className="w-4 h-4 animate-spin" />
                    Cloning...
                  </>
                ) : (
                  'Clone & Verify'
                )}
              </button>
            </form>
          )}

          {step === 'configure' && cloneResult && (
            <form onSubmit={handleCreate} className="space-y-4">
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

              <div className="bg-gray-50 dark:bg-gray-700/50 rounded-lg p-3">
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  Dockerfile: <span className="font-medium text-gray-900 dark:text-gray-100">{cloneResult.dockerfilePath}</span>
                </p>
              </div>

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
                <p className="text-xs text-gray-400 mt-1">
                  External port will be auto-assigned from 13001-13999
                </p>
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

              {error && (
                <p className="text-red-600 dark:text-red-400 text-sm">{error}</p>
              )}

              <div className="flex gap-3">
                <button
                  type="button"
                  onClick={() => setStep('url')}
                  className="flex-1 py-2 border border-gray-300 dark:border-gray-600
                           text-gray-700 dark:text-gray-300 font-medium rounded-lg
                           hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
                >
                  Back
                </button>
                <button
                  type="submit"
                  disabled={isCreating}
                  className="flex-1 py-2 bg-gray-900 hover:bg-gray-800 disabled:bg-gray-300
                           dark:bg-white dark:hover:bg-gray-100 dark:disabled:bg-gray-600
                           text-white dark:text-gray-900 font-medium rounded-lg transition-colors
                           flex items-center justify-center gap-2"
                >
                  {isCreating ? (
                    <>
                      <Loader2 className="w-4 h-4 animate-spin" />
                      Creating...
                    </>
                  ) : (
                    'Build & Create App'
                  )}
                </button>
              </div>
            </form>
          )}
        </div>
      </div>
    </div>
  );
}
