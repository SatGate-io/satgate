/**
 * API client for SatGate Cloud control plane
 */

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3000';

interface ApiOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE';
  body?: any;
}

async function api<T>(path: string, options: ApiOptions = {}): Promise<T> {
  const { method = 'GET', body } = options;
  
  const res = await fetch(`${API_URL}/api${path}`, {
    method,
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'include',
    body: body ? JSON.stringify(body) : undefined,
  });
  
  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: 'Request failed' }));
    throw new Error(error.error || `Request failed with status ${res.status}`);
  }
  
  return res.json();
}

// Auth
export async function requestMagicLink(email: string) {
  return api<{ sent: boolean; message: string }>('/auth/magic-link', {
    method: 'POST',
    body: { email },
  });
}

export async function verifyCode(code: string) {
  return api<{ success: boolean; tenant: { slug: string } }>('/auth/verify', {
    method: 'POST',
    body: { code },
  });
}

export async function logout() {
  return api<{ success: boolean }>('/auth/logout', { method: 'POST' });
}

export async function getMe() {
  return api<{ tenant: { id: string; slug: string; email: string } }>('/auth/me');
}

// Projects
export interface Project {
  id: string;
  slug: string;
  name: string;
  host?: string;
  configCount?: number;
  createdAt: string;
  updatedAt?: string;
}

export async function listProjects() {
  return api<{ projects: Project[] }>('/projects');
}

export async function createProject(name: string) {
  return api<{ project: Project }>('/projects', {
    method: 'POST',
    body: { name },
  });
}

export async function getProject(slug: string) {
  return api<{ project: Project & { activeConfig?: { id: string; version: number } } }>(
    `/projects/${slug}`
  );
}

export async function deleteProject(slug: string) {
  return api<{ success: boolean }>(`/projects/${slug}`, { method: 'DELETE' });
}

// Config
export interface ConfigVersion {
  id: string;
  version: number;
  yaml?: string;
  summary?: string;
  createdAt: string;
}

export async function uploadConfig(projectSlug: string, yaml: string) {
  return api<{ config: ConfigVersion }>(`/projects/${projectSlug}/config`, {
    method: 'POST',
    body: { yaml },
  });
}

export async function getConfig(projectSlug: string) {
  return api<{ config: ConfigVersion }>(`/projects/${projectSlug}/config`);
}

export async function validateConfig(projectSlug: string, yaml: string) {
  return api<{ valid: boolean; errors?: string[]; summary?: string }>(
    `/projects/${projectSlug}/config/validate`,
    {
      method: 'POST',
      body: { yaml },
    }
  );
}

