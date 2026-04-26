import type { MetadataRoute } from 'next';

type SitemapEntry = {
  path: string;
  lastModified: string;
  changeFrequency: MetadataRoute.Sitemap[number]['changeFrequency'];
  priority: number;
};

const baseUrl = 'https://satgate.io';

const staticRoutes: SitemapEntry[] = [
  { path: '', lastModified: '2026-04-26', changeFrequency: 'weekly', priority: 1.0 },
  { path: '/govern', lastModified: '2026-04-24', changeFrequency: 'weekly', priority: 0.9 },
  { path: '/economic-firewall', lastModified: '2026-04-26', changeFrequency: 'weekly', priority: 0.9 },
  { path: '/ai-agent-cost-control', lastModified: '2026-04-26', changeFrequency: 'weekly', priority: 0.9 },
  { path: '/mcp-governance', lastModified: '2026-04-26', changeFrequency: 'weekly', priority: 0.9 },
  { path: '/mcp-budget-enforcement', lastModified: '2026-04-26', changeFrequency: 'weekly', priority: 0.9 },
  { path: '/agent-api-governance', lastModified: '2026-04-26', changeFrequency: 'weekly', priority: 0.9 },
  { path: '/l402-agent-payments', lastModified: '2026-04-26', changeFrequency: 'weekly', priority: 0.9 },
  { path: '/robot-customer-payments', lastModified: '2026-04-26', changeFrequency: 'weekly', priority: 0.9 },
  { path: '/satgate-for-cursor', lastModified: '2026-04-26', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/satgate-for-claude-code', lastModified: '2026-04-26', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/satgate-for-claude-desktop', lastModified: '2026-04-26', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/satgate-for-openclaw', lastModified: '2026-04-26', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/pricing', lastModified: '2026-04-12', changeFrequency: 'monthly', priority: 0.9 },
  { path: '/security', lastModified: '2026-04-12', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog', lastModified: '2026-04-26', changeFrequency: 'weekly', priority: 0.8 },
  { path: '/compare', lastModified: '2026-04-26', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/compare/zuplo', lastModified: '2026-04-12', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/compare/bifrost', lastModified: '2026-04-12', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/compare/litellm', lastModified: '2026-04-26', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/compare/portkey', lastModified: '2026-04-26', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/compare/helicone', lastModified: '2026-04-26', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/compare/cloudflare-ai-gateway', lastModified: '2026-04-26', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/compare/cloud-native', lastModified: '2026-04-12', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/protect', lastModified: '2026-04-12', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/pay', lastModified: '2026-04-12', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/mint-demo', lastModified: '2026-04-12', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/sandbox', lastModified: '2026-04-12', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/roi-calculator', lastModified: '2026-04-26', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/runaway-agent-cost-calculator', lastModified: '2026-04-26', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/openai-budget-policy-generator', lastModified: '2026-04-26', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/mcp-tool-cost-policy-generator', lastModified: '2026-04-26', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/economic-firewall-readiness-grader', lastModified: '2026-04-26', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/tools', lastModified: '2026-04-26', changeFrequency: 'weekly', priority: 0.8 },
  { path: '/design-partners', lastModified: '2026-04-12', changeFrequency: 'monthly', priority: 0.6 },
  { path: '/agents', lastModified: '2026-04-12', changeFrequency: 'monthly', priority: 0.6 },
  { path: '/privacy', lastModified: '2026-04-12', changeFrequency: 'yearly', priority: 0.3 },
  { path: '/terms', lastModified: '2026-04-12', changeFrequency: 'yearly', priority: 0.3 },
];

const blogRoutes: SitemapEntry[] = [
  { path: '/blog/why-routing-isnt-governance', lastModified: '2026-02-06', changeFrequency: 'monthly', priority: 0.6 },
  { path: '/blog/beyond-connection-economic-governance-mcp', lastModified: '2026-02-12', changeFrequency: 'monthly', priority: 0.6 },
  { path: '/blog/how-we-built-budget-enforcement-mcp', lastModified: '2026-02-13', changeFrequency: 'monthly', priority: 0.6 },
  { path: '/blog/hard-capping-mcp-tool-spend', lastModified: '2026-02-14', changeFrequency: 'monthly', priority: 0.6 },
  { path: '/blog/security-as-a-profit-center', lastModified: '2026-04-26', changeFrequency: 'monthly', priority: 0.6 },
  { path: '/blog/ai-agent-api-cost-control', lastModified: '2026-04-24', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/blog/what-is-an-economic-firewall', lastModified: '2026-04-26', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/blog/mcp-budget-enforcement-guide', lastModified: '2026-04-24', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/agent-swarms-cost-governance', lastModified: '2026-03-05', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/blog/ai-agent-spending-limits', lastModified: '2026-03-10', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/blog/deepmind-intelligent-delegation-satgate', lastModified: '2026-03-11', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/api-gateway-for-ai-agents', lastModified: '2026-04-24', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/llm-cost-management', lastModified: '2026-04-24', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/ai-governance-api-teams', lastModified: '2026-03-19', changeFrequency: 'monthly', priority: 0.7 },
  { path: '/blog/why-economic-firewalls-are-the-prerequisite-for-autonomous-ai-agents', lastModified: '2026-03-20', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/the-enterprise-adoption-playbook-observe-control-charge', lastModified: '2026-03-20', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/can-adversaries-game-your-economic-firewall', lastModified: '2026-03-23', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/mcp-gateway-guide', lastModified: '2026-03-24', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/api-monetization-ai', lastModified: '2026-04-26', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/why-process-wont-scale-for-ai-agent-costs', lastModified: '2026-04-26', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/macaroon-tokens-vs-api-keys', lastModified: '2026-04-24', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/http-402-payment-required-use-cases', lastModified: '2026-04-24', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/l402-protocol-explained', lastModified: '2026-04-26', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/zero-trust-for-ai-agents', lastModified: '2026-04-26', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/start-at-1-credit-economic-policy', lastModified: '2026-04-26', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/how-to-add-budget-limits-to-openai-api-calls', lastModified: '2026-04-24', changeFrequency: 'monthly', priority: 0.8 },
  { path: '/blog/cursor-mcp-proxy-setup-guide', lastModified: '2026-04-09', changeFrequency: 'monthly', priority: 0.8 },
];

export default function sitemap(): MetadataRoute.Sitemap {
  return [...staticRoutes, ...blogRoutes].map((entry) => ({
    url: `${baseUrl}${entry.path}`,
    lastModified: new Date(`${entry.lastModified}T00:00:00.000Z`),
    changeFrequency: entry.changeFrequency,
    priority: entry.priority,
  }));
}
