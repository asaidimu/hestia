import { fileURLToPath } from "node:url";
import { defineConfig } from "vitest/config";

const hestiaClientDir = fileURLToPath(new URL("../core", import.meta.url));

export default defineConfig({
  resolve: {
    // Always run the tests against the SDK *source* so stack traces and
    // breakpoints stay readable. The built dist is exercised via the
    // workspace dependency in normal (non-test) usage.
    alias: {
      "@asaidimu/hestia": hestiaClientDir + "/index.ts",
    },
  },
  test: {
    include: ["test/**/*.test.ts"],
    fileParallelism: false,
    testTimeout: 20000,
    setupFiles: ["./test/setup.ts"],
  },
});
