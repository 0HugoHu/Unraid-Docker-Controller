const API_BASE = '/api/v1';

async function fetchAPI<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
    credentials: 'include',
  });

  if (response.status === 401) {
    window.location.href = '/';
    throw new Error('Unauthorized');
  }

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Unknown error' }));
    throw new Error(error.error || 'Request failed');
  }

  return response.json();
}

export const api = {
  // Auth
  login: (password: string) =>
    fetchAPI<{ token: string }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ password }),
    }),

  logout: () => fetchAPI('/auth/logout', { method: 'POST' }),

  checkAuth: () =>
    fetchAPI<{ authenticated: boolean }>('/auth/check'),

  updatePassword: (currentPassword: string, newPassword: string) =>
    fetchAPI('/auth/password', {
      method: 'PUT',
      body: JSON.stringify({ currentPassword, newPassword }),
    }),

  // Apps
  getApps: () => fetchAPI<App[]>('/apps'),

  getApp: (id: string) => fetchAPI<{ app: App; uptime?: string }>(`/apps/${id}`),

  cloneRepo: (repoUrl: string, branch: string) =>
    fetchAPI<CloneResult>('/apps/clone', {
      method: 'POST',
      body: JSON.stringify({ repoUrl, branch }),
    }),

  createApp: (repoUrl: string, branch: string, config: AppConfig) =>
    fetchAPI<App>('/apps', {
      method: 'POST',
      body: JSON.stringify({ repoUrl, branch, config }),
    }),

  updateApp: (id: string, config: Partial<AppConfig>) =>
    fetchAPI<App>(`/apps/${id}`, {
      method: 'PUT',
      body: JSON.stringify(config),
    }),

  deleteApp: (id: string) =>
    fetchAPI(`/apps/${id}`, { method: 'DELETE' }),

  buildApp: (id: string) =>
    fetchAPI(`/apps/${id}/build`, { method: 'POST' }),

  startApp: (id: string) =>
    fetchAPI(`/apps/${id}/start`, { method: 'POST' }),

  stopApp: (id: string) =>
    fetchAPI(`/apps/${id}/stop`, { method: 'POST' }),

  restartApp: (id: string) =>
    fetchAPI(`/apps/${id}/restart`, { method: 'POST' }),

  pullAndRebuild: (id: string) =>
    fetchAPI(`/apps/${id}/pull`, { method: 'POST' }),

  getLogs: (id: string, lines = 100) =>
    fetchAPI<{ logs: string }>(`/apps/${id}/logs?lines=${lines}`),

  getBuildLogs: (id: string) =>
    fetchAPI<{ logs: string }>(`/apps/${id}/build-logs`),

  clearLogs: (id: string) =>
    fetchAPI(`/apps/${id}/logs`, { method: 'DELETE' }),

  // System
  getSystemInfo: () => fetchAPI<SystemInfo>('/system/info'),

  getStorage: () => fetchAPI<StorageInfo>('/system/storage'),

  getPorts: () =>
    fetchAPI<{ usedPorts: number[]; range: { start: number; end: number } }>(
      '/system/ports'
    ),

  pruneImages: () => fetchAPI<{ spaceReclaimed: number }>('/system/prune', { method: 'POST' }),

  clearAllLogs: () => fetchAPI('/system/logs', { method: 'DELETE' }),
};

export interface App {
  id: string;
  name: string;
  slug: string;
  description: string;
  repoUrl: string;
  branch: string;
  lastCommit: string;
  dockerfilePath: string;
  imageName: string;
  containerName: string;
  internalPort: number;
  externalPort: number;
  env: Record<string, string>;
  status: 'stopped' | 'running' | 'building' | 'build-failed' | 'starting' | 'error';
  lastBuild: string | null;
  lastBuildDuration: string;
  lastBuildSuccess: boolean;
  imageSize: number;
  createdAt: string;
  updatedAt: string;
}

export interface CloneResult {
  slug: string;
  name: string;
  description: string;
  hasDockerfile: boolean;
  dockerfilePath: string;
  manifest: {
    name?: string;
    description?: string;
    defaultPort?: number;
    env?: Record<string, string>;
  } | null;
  suggestedPort: number;
}

export interface AppConfig {
  name?: string;
  dockerfilePath?: string;
  buildContext?: string;
  internalPort?: number;
  externalPort?: number;
  env?: Record<string, string>;
  buildArgs?: Record<string, string>;
}

export interface SystemInfo {
  version: string;
  totalApps: number;
  runningApps: number;
  docker: {
    containers: number;
    containersRunning: number;
    images: number;
    serverVersion: string;
  };
}

export interface StorageInfo {
  database: number;
  repositories: number;
  logs: number;
  images: number;
  total: number;
}
