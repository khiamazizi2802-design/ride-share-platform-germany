import type { Config } from 'tailwindcss';

const config: Config = {
  content: [
    './pages/**/*{js,ts,jsx,tsx,mdx}',
    './components/**/*{js,ts,jsx,tsx,mdx}',
    './app/**/*{js,ts,jsx,tsx,mdx}',
  ],
  theme: {
    extend: {
      colors: {
        gruen: {
          50: '#f0dfd4',
          100: '#dcfce7',
          200: '#bbf7d0',
          300: '#86efac',
          400: '#4ade80',
          500: '#22c55e',
          600: '#16a34a',
          700: '#15803d',
          800: '#166534',
          900: '#14532d',
          950: '#052e16',
        },
        eco: {
          light: '#7dd87d',
          DEFAULT: '#4caf50',
          dark: '#2e7d32',
        },
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
      },
      animation: {
        'fade-': 'fadeIn 0.5s ease-in-out',
        'slide-up': 'slideUp 0.5s ease-out',
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        slideUp: {
          '0%': { transform: 'translateY(20px)', opacity: '0' },
          '100%': { transform: 'translateY(0)', opacity: '1' },
        },
      },
    },
  },
  plugins: [],
};

export default config;