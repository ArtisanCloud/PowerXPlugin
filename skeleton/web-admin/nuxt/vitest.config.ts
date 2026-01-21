import { defineConfig } from "vitest/config";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";

const dir = dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  test: {
    environment: "node",
    globals: true,
    setupFiles: [],
  },
  resolve: {
    alias: {
      "~": resolve(dir, "./app"),
      "@": resolve(dir, "./app"),
    },
  },
});
