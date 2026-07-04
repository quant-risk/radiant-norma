/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  experimental: {
    typedRoutes: true,
  },
  env: {
    RADIANT_API_URL: process.env.RADIANT_API_URL || 'http://localhost:8080',
  },
}

module.exports = nextConfig
