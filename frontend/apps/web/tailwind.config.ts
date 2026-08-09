import type { Config } from "tailwindcss";

// Valores espelham @lifeos/ui-tokens (packages/ui-tokens/src/index.ts) — duplicados
// aqui em vez de importados porque o config do Tailwind roda fora do pipeline do
// Vite e não vale a pena resolver um pacote TS sem build só por isto. Se um valor
// mudar lá, muda aqui também no mesmo commit.
export default {
  darkMode: ["class"],
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        bg: {
          base: "#0b0d10",
          raised: "#14171c",
          overlay: "#1c2027",
        },
        border: {
          DEFAULT: "#262b33",
          subtle: "#1c2027",
        },
        fg: {
          primary: "#e8eaed",
          secondary: "#9aa2ad",
          muted: "#6b7280",
        },
        delta: {
          positive: "#4ade80",
          "positive-muted": "#1a3324",
        },
        stale: {
          fg: "#9aa2ad",
          bg: "#1c2027",
        },
        danger: {
          fg: "#f87171",
          bg: "#2a1618",
        },
      },
      borderRadius: {
        sm: "0.375rem",
        md: "0.5rem",
        lg: "0.75rem",
      },
    },
  },
  plugins: [],
} satisfies Config;
