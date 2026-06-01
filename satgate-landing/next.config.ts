import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  trailingSlash: false,
  async redirects() {
    return [
      { source: '/blog/the-enterprise-adoption-playbook-observe-control-charge', destination: '/blog/the-enterprise-adoption-playbook-observe-control-prove', permanent: true },
      { source: '/blog/agent-to-agent-collaboration-security', destination: '/blog', permanent: true },
      { source: '/docs', destination: '/', permanent: true },
      { source: '/about', destination: '/', permanent: true },
      { source: '/contact', destination: '/design-partners', permanent: true },
      { source: '/demo', destination: '/sandbox', permanent: true },
      { source: '/api', destination: '/', permanent: true },
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
