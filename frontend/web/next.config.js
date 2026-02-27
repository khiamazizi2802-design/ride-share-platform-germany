/** @type {import('next').NextConfig} */
const nextConfig = {
  experimental: {
    appDir: true,
  },
  images: {
    domains: ['localhost'],
  },
  i18n: {
    locales: ['de', 'en'],
    defaultLocale: 'de',
  },
};

module.exports = nextConfig;