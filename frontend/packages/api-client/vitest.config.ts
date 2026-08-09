import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    environment: "node",
    globals: true,
    coverage: {
      provider: "v8",
      reporter: ["text", "html"],
      include: ["src/**/*.ts"],
      exclude: ["src/schema.ts", "src/index.ts", "**/*.d.ts", "**/*.test.ts"],
      // CLAUDE.md: nada é entregue sem pelo menos 80% de cobertura.
      thresholds: {
        lines: 80,
        functions: 80,
        branches: 80,
        statements: 80,
      },
    },
  },
});
