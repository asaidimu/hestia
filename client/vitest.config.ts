import { fileURLToPath } from "node:url";
import { defineConfig } from "vitest/config";

const coreDir = fileURLToPath(new URL("./packages/core", import.meta.url));
const mockDir = fileURLToPath(new URL("./packages/mock", import.meta.url));

export default defineConfig({
  resolve: {
    alias: {
      "@asaidimu/hestia": coreDir + "/index.ts",
      "@asaidimu/hestia-mock": mockDir + "/src/index.ts",
    },
  },
  test: {
    include: [
      "packages/core/**/*.test.ts",
      "packages/mock/**/*.test.ts",
    ],
    exclude: [
      "packages/mock/node_modules/**",
      "packages/core/node_modules/**",
    ],
    globalSetup: ["./test-setup.ts"],
    fileParallelism: false,
    testTimeout: 20000,
  },
});
