import { defineConfig } from "tsdown";

export default defineConfig([
  {
    entry: ["src/index.ts"],
    format: ["esm", "cjs"],
    dts: true,
    sourcemap: true,
    clean: true,
    platform: "neutral",
    // Keep the SDK and shared utils external — they are real dependencies.
    external: [/^@asaidimu\//],
  },
]);
