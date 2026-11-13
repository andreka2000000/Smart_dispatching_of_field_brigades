import type { NextConfig } from 'next';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const nextConfig: NextConfig = {
  turbopack: { root: path.dirname(fileURLToPath(import.meta.url)) },
  async rewrites() {
    if (process.env.NODE_ENV !== 'development') return [];
    return [
      {
        source: '/api/:path*',
        destination: `${process.env.API_PROXY_TARGET || 'http://127.0.0.1:8080'}/api/:path*`,
      },
    ];
  },
};
export default nextConfig;
