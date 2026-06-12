import type { Config } from "tailwindcss";

const config: Config = {
  darkMode: ["class"],
  content: [
    "./app/**/*.{ts,tsx}",
    "./components/**/*.{ts,tsx}",
    "./lib/**/*.{ts,tsx}"
  ],
  theme: {
    extend: {
      colors: {
        background: "hsl(var(--background))",
        foreground: "hsl(var(--foreground))",
        muted: "hsl(var(--muted))",
        border: "hsl(var(--border))",
        surface: "hsl(var(--surface))",
        primary: "hsl(var(--primary))",
        accent: "hsl(var(--accent))",
        danger: "hsl(var(--danger))",
        info: "hsl(var(--info))"
      },
      borderRadius: {
        lg: "8px",
        md: "6px",
        sm: "4px"
      },
      boxShadow: {
        panel: "0 18px 40px rgb(16 19 22 / 0.08)"
      }
    }
  },
  plugins: []
};

export default config;

