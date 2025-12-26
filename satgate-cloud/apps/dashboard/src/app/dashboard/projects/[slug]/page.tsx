'use client';

import { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import { getProject, getConfig, uploadConfig, validateConfig, Project, ConfigVersion } from '@/lib/api';

const STARTER_CONFIG = `version: 1

# Your upstream API
upstreams:
  my_api:
    url: https://api.example.com
    addHeaders:
      X-API-Key: "<YOUR_API_KEY>"

# Routes with L402 pricing
routes:
  - name: premium
    match:
      pathPrefix: /api/premium
    upstream: my_api
    policy:
      kind: l402
      tier: premium
      priceSats: 100
      scope: "api:premium"

  - name: basic
    match:
      pathPrefix: /api/
    upstream: my_api
    policy:
      kind: l402
      tier: basic
      priceSats: 10
      scope: "api:basic"

  # Default deny
  - name: default-deny
    match:
      path: /
    policy:
      kind: deny
      status: 403
`;

export default function ProjectPage() {
  const params = useParams();
  const router = useRouter();
  const slug = params.slug as string;

  const [project, setProject] = useState<Project | null>(null);
  const [config, setConfig] = useState<ConfigVersion | null>(null);
  const [yaml, setYaml] = useState(STARTER_CONFIG);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [validating, setValidating] = useState(false);
  const [validation, setValidation] = useState<{ valid: boolean; errors?: string[]; summary?: string } | null>(null);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  useEffect(() => {
    async function load() {
      try {
        const projectData = await getProject(slug);
        setProject(projectData.project);

        if (projectData.project.activeConfig) {
          const configData = await getConfig(slug);
          setConfig(configData.config);
          setYaml(configData.config.yaml || STARTER_CONFIG);
        }
      } catch (err) {
        setError((err as Error).message);
      } finally {
        setLoading(false);
      }
    }
    load();
  }, [slug]);

  const handleValidate = async () => {
    setValidating(true);
    setValidation(null);
    setError('');

    try {
      const result = await validateConfig(slug, yaml);
      setValidation(result);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setValidating(false);
    }
  };

  const handleSave = async () => {
    setSaving(true);
    setError('');
    setSuccess('');

    try {
      const result = await uploadConfig(slug, yaml);
      setConfig(result.config);
      setSuccess('Config saved successfully!');
      setValidation({ valid: true, summary: result.config.summary });
      setTimeout(() => setSuccess(''), 3000);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="flex justify-center py-12">
        <svg className="animate-spin h-8 w-8 text-primary-400" viewBox="0 0 24 24">
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
        </svg>
      </div>
    );
  }

  if (!project) {
    return (
      <div className="text-center py-12">
        <h1 className="text-xl font-semibold mb-2">Project not found</h1>
        <Link href="/dashboard" className="text-primary-400 hover:text-primary-300">
          Back to projects
        </Link>
      </div>
    );
  }

  return (
    <div>
      {/* Header */}
      <div className="flex items-center gap-4 mb-8">
        <Link href="/dashboard" className="text-white/60 hover:text-white transition-colors">
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
        </Link>
        <div>
          <h1 className="text-2xl font-bold">{project.name}</h1>
          <p className="text-white/60 font-mono text-sm">{project.slug}.satgate.cloud</p>
        </div>
      </div>

      {/* Quick test */}
      <div className="card p-4 mb-6 flex items-center justify-between">
        <div>
          <p className="text-sm text-white/60">Test your gateway:</p>
          <code className="text-sm font-mono text-primary-400">
            curl https://{project.slug}.satgate.cloud/api/test
          </code>
        </div>
        <button
          onClick={() => navigator.clipboard.writeText(`curl https://${project.slug}.satgate.cloud/api/test`)}
          className="btn btn-secondary text-sm"
        >
          Copy
        </button>
      </div>

      {/* Editor section */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Config editor */}
        <div className="lg:col-span-2">
          <div className="card">
            <div className="flex items-center justify-between p-4 border-b border-white/5">
              <h2 className="font-semibold">Gateway Configuration</h2>
              <div className="flex gap-2">
                <button
                  onClick={handleValidate}
                  disabled={validating}
                  className="btn btn-secondary text-sm"
                >
                  {validating ? 'Validating...' : 'Validate'}
                </button>
                <button
                  onClick={handleSave}
                  disabled={saving}
                  className="btn btn-primary text-sm"
                >
                  {saving ? 'Saving...' : 'Save & Deploy'}
                </button>
              </div>
            </div>
            <div className="p-4">
              <textarea
                value={yaml}
                onChange={(e) => setYaml(e.target.value)}
                className="w-full h-[500px] font-mono text-sm bg-surface-100 border border-white/10 rounded-lg p-4 text-white/90 resize-none focus:outline-none focus:border-primary-500"
                spellCheck={false}
              />
            </div>
          </div>
        </div>

        {/* Validation panel */}
        <div>
          <div className="card p-6">
            <h3 className="font-semibold mb-4">Validation</h3>
            
            {error && (
              <div className="bg-red-500/10 border border-red-500/30 text-red-400 rounded-lg p-3 mb-4 text-sm">
                {error}
              </div>
            )}

            {success && (
              <div className="bg-green-500/10 border border-green-500/30 text-green-400 rounded-lg p-3 mb-4 text-sm">
                {success}
              </div>
            )}

            {validation && (
              <div>
                {validation.valid ? (
                  <div className="flex items-center gap-2 text-green-400 mb-4">
                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                    <span>Valid configuration</span>
                  </div>
                ) : (
                  <div className="mb-4">
                    <div className="flex items-center gap-2 text-red-400 mb-2">
                      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                      </svg>
                      <span>Invalid configuration</span>
                    </div>
                    <ul className="text-sm text-red-400/80 space-y-1">
                      {validation.errors?.map((err, i) => (
                        <li key={i}>• {err}</li>
                      ))}
                    </ul>
                  </div>
                )}

                {validation.summary && (
                  <pre className="bg-surface-100 rounded-lg p-4 text-sm text-white/70 overflow-x-auto whitespace-pre-wrap">
                    {validation.summary}
                  </pre>
                )}
              </div>
            )}

            {!validation && (
              <p className="text-white/40 text-sm">
                Click "Validate" to check your configuration
              </p>
            )}
          </div>

          {/* Active config info */}
          {config && (
            <div className="card p-6 mt-6">
              <h3 className="font-semibold mb-4">Active Version</h3>
              <div className="text-sm text-white/60 space-y-2">
                <p>Version: <span className="text-white">{config.version}</span></p>
                <p>Deployed: <span className="text-white">{new Date(config.createdAt).toLocaleString()}</span></p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

