import { defineConfig } from "tsdown";

export default defineConfig({
  entry: {
    index: "index.ts",
  },
  outDir: "dist",
  format: ["esm", "cjs"],
  unbundle: false,
  // deps: {
  //   alwaysBundle: [/.*/],
  // },
  minify: true,
  clean: true,
  sourcemap: false,
  dts: true,
  platform: "neutral",
  fixedExtension: true,
});
