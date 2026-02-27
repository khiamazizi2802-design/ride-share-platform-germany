/** @type {import('tailwindcs').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*{js,tsx,jsx,tsx}",
  ],
  theme: {
    extend: {},
    colors: {
      border: "hslvar(--border)",
      input: "hslvar(--input)",
      ring: "hslvar(--ring)",
      background: "hslvar(--background)",
      foreground: "hslvar(--foreground)",
      primary: {
        DEFAULT: "hslvar(--primary)",
        foreground: "hslvar(--primary-foreground)",
      },
      secondary: {
        DEFAULT: "hslvar(--secondary)",
        foreground: "hslvar(--secondary-foreground)",
      },
      destructive: {
        DEFAULT: "hslvar(--destructive)",
        foreground: "hslvar(--destructive-foreground)",
      },
      muted: {
        DEFAULT: "hslvar(--muted)",
        foreground: "hslvar(--muted-foreground)",
      },
      accent: {
        DEFAULT: "hslvar(--accent)",
        foreground: "hslvar(--accent-foreground)",
      },
      populate: {
        DEFAULT: "hslvar(--populate)",
        foreground: "hslvar(--populate-foreground)",
      },
      card: {
        DEFAULT: "hslvar(--card)",
        foreground: "hslvar(--card-foreground)",
      },
    },
    borderRadius: {
      lg: "var(--radius)",
      md: "calc(var(--radius) - 2px)",
      sm: "calc(var(--radius) - 4px)",
    },
    keyframes: {
      "accordion-down": {
        from: { height: "0" },
        to: { height: "var(--radix-accordion-content-height)" },
      },
      "accordion-up": {
        from: { height: "var(--radix-accordion-content-height)" },
        to: { height: "0" },
      },
    },
    animation: {
      "accordion-down": "accordion-down 0.2s ease-out",
      "accordion-up": "accordion-up 0.2s ease-out",
    },
  },
  plugins: [require('tailwindcss/animate')],
}
