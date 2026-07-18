import { readFileSync, writeFileSync } from "fs";

function updateDistPackage() {
  const packageJson = JSON.parse(readFileSync("package.json", "utf8"));
  const distPackageJson = JSON.parse(readFileSync("dist.package.json", "utf8"));

  // Replace dependencies in dist.package.json
  distPackageJson.dependencies = packageJson.dependencies || {};

  // Write back to dist.package.json
  writeFileSync("out.package.json", JSON.stringify(distPackageJson, null, 2) + "\n");

  console.log("Updated dist.package.json successfully!");
}

updateDistPackage()
