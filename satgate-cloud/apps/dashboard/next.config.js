/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  transpilePackages: ['@satgate/gateway-config'],
  output: 'standalone',
};

module.exports = nextConfig;

