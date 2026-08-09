/**
 * Design tokens do produto (06-frontend.md §7). Tema escuro por padrão — uso
 * noturno, depois de programar (05-ux.md §12.1, ponto em aberto mas é a
 * suposição de trabalho).
 *
 * Regras que os tokens existem para impor:
 * - `delta.positive` é a única cor de destaque forte do app — existe pra
 *   uma coisa só: progresso.
 * - `stale` é cinza, nunca vermelho/âmbar — não é alerta (P5).
 * - Vermelho fica reservado pra falha técnica de verdade, nunca pra
 *   estagnação ou atraso de meta.
 */
export const colors = {
  bg: {
    base: "#0b0d10",
    raised: "#14171c",
    overlay: "#1c2027",
  },
  border: {
    default: "#262b33",
    subtle: "#1c2027",
  },
  text: {
    primary: "#e8eaed",
    secondary: "#9aa2ad",
    muted: "#6b7280",
  },
  delta: {
    /** única cor de destaque forte do app — reservada para progresso */
    positive: "#4ade80",
    positiveMuted: "#1a3324",
  },
  stale: {
    /** "sem evidência recente": cinza com ⟳, nunca vermelho/âmbar (P5) */
    fg: "#9aa2ad",
    bg: "#1c2027",
  },
  danger: {
    /** reservado para falha técnica real, nunca para progresso/estagnação */
    fg: "#f87171",
    bg: "#2a1618",
  },
} as const;

export const spacing = {
  xs: "0.25rem",
  sm: "0.5rem",
  md: "1rem",
  lg: "1.5rem",
  xl: "2rem",
  "2xl": "3rem",
} as const;

export const radius = {
  sm: "0.375rem",
  md: "0.5rem",
  lg: "0.75rem",
} as const;
