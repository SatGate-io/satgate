import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  trailingSlash: false,
  async redirects() {
    return [
      { source: '/blog/the-enterprise-adoption-playbook-observe-control-charge', destination: '/blog/the-enterprise-adoption-playbook-observe-control-prove', permanent: true },
      { source: '/blog/agent-to-agent-collaboration-security', destination: '/blog', permanent: true },
      { source: '/about', destination: '/', permanent: true },
      { source: '/contact', destination: '/design-partners', permanent: true },
      { source: '/demo', destination: '/sandbox', permanent: true },
      { source: '/api', destination: '/', permanent: true },
      { source: '/dashboard', destination: '/sandbox', permanent: true },
      { source: '/crawl', destination: '/sandbox', permanent: true },
      { source: '/monetize', destination: '/pay', permanent: true },
      { source: '/seo-distribution-kit', destination: '/', permanent: true },
      { source: '/openai-budget-policy-generator', destination: '/build', permanent: true },
      { source: '/mcp-tool-cost-policy-generator', destination: '/mcp', permanent: true },
      { source: '/agent-spend-policy-template', destination: '/build', permanent: true },
      { source: '/revocable-capability-token-policy-template', destination: '/build', permanent: true },
    ];
  },
  async headers() {
    return [
      {
        source: "/(.*)",
        headers: [
          { key: "X-Content-Type-Options", value: "nosniff" },
          { key: "X-Frame-Options", value: "DENY" },
          { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
          {
            key: "Permissions-Policy",
            value: "camera=(), microphone=(), geolocation=()",
          },
        ],
      },
    ];
  },
};

export default nextConfig;
