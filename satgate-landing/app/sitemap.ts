import type { MetadataRoute } from 'next';

type SitemapEntry = {
  path: string;
  lastModified: string;
  changeFrequency: MetadataRoute.Sitemap[number]['changeFrequency'];
  priority: number;
};

const baseUrl = 'https://satgate.io';

const staticRoutes: SitemapEntry[] = [
  { path: '', lastModified: '2026-05-05', changeFrequency: 'weekly', priority: 1.0 },
  { path: '/govern', lastModified: '2026-06-01', changeFrequency: 'weekly', priority: 0.9 },
  { path: '/economic-firewall', lastModified: '2026-05-09', changeFrequency: 'weekly', priority: 0.9 },
  { path: '/partners/rails', lastModified: '2026-05-14', changeFrequency: 'weekly', priority: 0.85 },
  { path: '/policy-to-proof', lastModified: '2026-05-10', changeFrequency: 'weekly', priority: 0.95 },
  { path: '/ai-agent-cost-control', lastModified: '2026-05-05', changeFrequency: 'weekly', priority: 0.9 },
  { path: '/llm-cost-dashboard', lastModified: '2026-05-03', changeFrequency: 'monthly', priority: 0.85 },
  { path: '/agent-spending-limits', lastModified: '2026-05-03', changeFrequency: 'monthly', priority: 0.85 },
  { path: '/ai-agent-runaway-spend-benchmark', lastModified: '2026-05-03', changeFrequency: 'monthly', priority: 0.85 },
  { path: '/mcp', lastModified: '2026-09-30', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/build', lastModified: '2026-05-12', changeFrequency: 'weekly', priority: 0.95 },
  { path: '/accept-satgate-capabilities', lastModified: '2026-05-13', changeFrequency: 'weekly', priority: 0.9 },
  { path: '/capability-auth', lastModified: '2026-05-08', changeFrequency: 'weekly', priority: 0.95 },
  { path: '/agent-control-plane', lastModified: '2026-05-05', changeFrequency: 'weekly', priority: 0.9 },
  { path: '/evidence-pack-demo', lastModified: '2026-05-10', changeFrequency: 'weekly', priority: 0.95 },
  { path: '/verify-evidence-pack', lastModified: '2026-07-02', changeFrequency: 'weekly', priority: 0.95 },
  { path: '/docs', lastModified: '2026-07-02', changeFrequency: 'weekly', priority: 0.8 },
  { path: '/agent-capability-tokens', lastModified: '2026-05-10', changeFrequency: 'weekly', priority: 0.9 },
  { path: '/capability-lifecycle-demo', lastModified: '2026-05-10', changeFrequency: 'weekly', priority: 0.9 },
  { path: '/pricing', lastModified: '2026-09-27', changeFrequency: 'monthly', priority: 0.9 },
  { path: '/security', lastModified: '2026-05-03', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog', lastModified: '2026-06-04', changeFrequency: 'weekly', priority: 0.8 },
  { path: '/compare', lastModified: '2026-05-10', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/compare/aws-agentcore-payments', lastModified: '2026-05-10', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/compare/cloudflare-ai-gateway', lastModified: '2026-05-10', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/compare/langsmith-helicone-datadog', lastModified: '2026-05-10', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/compare/api-gateway-rate-limits', lastModified: '2026-05-10', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/compare/openai-anthropic-budget-controls', lastModified: '2026-05-10', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/compare/zuplo', lastModified: '2026-05-04', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/compare/bifrost', lastModified: '2026-05-04', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/compare/litellm', lastModified: '2026-05-04', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/compare/portkey', lastModified: '2026-05-04', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/compare/helicone', lastModified: '2026-05-04', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/compare/kong-ai-gateway', lastModified: '2026-05-04', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/compare/apigee', lastModified: '2026-05-04', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/compare/tyk', lastModified: '2026-05-04', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/compare/langfuse', lastModified: '2026-05-04', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/compare/cloud-native', lastModified: '2026-05-02', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/protect', lastModified: '2026-05-03', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/mint-demo', lastModified: '2026-05-03', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/sandbox', lastModified: '2026-05-03', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/runaway-agent-cost-calculator', lastModified: '2026-05-04', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/economic-firewall-readiness-grader', lastModified: '2026-05-05', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/tools', lastModified: '2026-09-30', changeFrequency: 'weekly', priority: 0.8 },
  { path: '/design-partners', lastModified: '2026-05-02', changeFrequency: 'monthly', priority: 0.6 },
  { path: '/privacy', lastModified: '2026-05-05', changeFrequency: 'yearly', priority: 0.3 },
  { path: '/terms', lastModified: '2026-05-05', changeFrequency: 'yearly', priority: 0.3 },
];

const blogRoutes: SitemapEntry[] = [
  { path: '/blog/always-on-agents-economic-authority', lastModified: '2026-06-04', changeFrequency: 'monthly', priority: 0.85 },
  { path: '/blog/ai-spend-governance', lastModified: '2026-05-22', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/why-routing-isnt-governance', lastModified: '2026-05-04', changeFrequency: 'monthly', priority: 0.6 },
  { path: '/blog/beyond-connection-economic-governance-mcp', lastModified: '2026-05-04', changeFrequency: 'monthly', priority: 0.6 },
  { path: '/blog/how-we-built-budget-enforcement-mcp', lastModified: '2026-05-04', changeFrequency: 'monthly', priority: 0.6 },
  { path: '/blog/hard-capping-mcp-tool-spend', lastModified: '2026-05-03', changeFrequency: 'monthly', priority: 0.6 },
  { path: '/blog/security-as-a-profit-center', lastModified: '2026-05-02', changeFrequency: 'monthly', priority: 0.6 },

  { path: '/blog/what-is-an-economic-firewall', lastModified: '2026-05-04', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/blog/mcp-budget-enforcement-guide', lastModified: '2026-05-04', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/agent-swarms-cost-governance', lastModified: '2026-05-04', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/blog/ai-agent-spending-limits', lastModified: '2026-05-04', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/blog/deepmind-intelligent-delegation-satgate', lastModified: '2026-05-04', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/api-gateway-for-ai-agents', lastModified: '2026-05-04', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/llm-cost-management', lastModified: '2026-06-01', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/ai-agent-api-cost-control', lastModified: '2026-05-05', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/ai-governance-api-teams', lastModified: '2026-05-04', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/blog/why-economic-firewalls-are-the-prerequisite-for-autonomous-ai-agents', lastModified: '2026-05-03', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/the-enterprise-adoption-playbook-observe-control-prove', lastModified: '2026-06-01', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/can-adversaries-game-your-economic-firewall', lastModified: '2026-05-02', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/mcp-gateway-guide', lastModified: '2026-05-04', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/api-monetization-ai', lastModified: '2026-05-04', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/why-process-wont-scale-for-ai-agent-costs', lastModified: '2026-05-02', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/macaroon-tokens-vs-api-keys', lastModified: '2026-05-04', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/http-402-payment-required-use-cases', lastModified: '2026-09-07', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/l402-protocol-explained', lastModified: '2026-05-04', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/zero-trust-for-ai-agents', lastModified: '2026-05-04', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/start-at-1-credit-economic-policy', lastModified: '2026-05-04', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/how-to-add-budget-limits-to-openai-api-calls', lastModified: '2026-08-12', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/cursor-mcp-proxy-setup-guide', lastModified: '2026-05-04', changeFrequency: 'monthly', priority: 0.8 },
];

export default function sitemap(): MetadataRoute.Sitemap {
  return [...staticRoutes, ...blogRoutes].map((entry) => ({
    url: `${baseUrl}${entry.path}`,
    lastModified: new Date(`${entry.lastModified}T00:00:00.000Z`),
    changeFrequency: entry.changeFrequency,
    priority: entry.priority,
  }));
}
