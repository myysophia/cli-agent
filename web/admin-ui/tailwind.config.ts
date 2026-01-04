import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./src/**/*.{ts,tsx}"] ,
  theme: {
    extend: {
      fontFamily: {
        sans: ["Space Grotesk", "IBM Plex Sans", "SF Pro Text", "system-ui", "sans-serif"],
        mono: ["IBM Plex Mono", "SFMono-Regular", "ui-monospace", "monospace"]
      },
      colors: {
        bg: "var(--bg)",
        "bg-soft": "var(--bg-soft)",
        panel: "var(--panel)",
        ink: "var(--ink)",
        muted: "var(--muted)",
        accent: "var(--accent)",
        "accent-2": "var(--accent-2)",
        ring: "var(--ring)"
      },
      boxShadow: {
        glow: "0 20px 80px rgba(15, 23, 42, 0.18)",
        card: "0 16px 40px rgba(15, 23, 42, 0.12)"
      }
    }
  },
  plugins: []
};

export default config;
