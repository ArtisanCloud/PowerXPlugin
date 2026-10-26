import vue from "@vitejs/plugin-vue";
import { defineConfig } from "vitest/config";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";

const dir = dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  plugins: [vue()],
  test: {
    environment: "node",
    globals: true,
    setupFiles: [],
  },
  resolve: {
    alias: {
      "#imports": resolve(dir,"./tests/unit/helpers/regression-imports.ts"),
      "~": resolve(dir, "./app"),
      "@": resolve(dir, "./app"),
    },
  },
});
