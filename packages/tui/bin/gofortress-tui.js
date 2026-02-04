#!/usr/bin/env node
import { spawn } from "child_process";
import { fileURLToPath } from "url";
import { dirname, join } from "path";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// Check for --legacy flag
if (process.argv.includes("--legacy")) {
  // Remove --legacy from args and spawn Go TUI
  const args = process.argv.slice(2).filter(arg => arg !== "--legacy");
  const legacyBin = join(__dirname, "../../../bin/gofortress-legacy");

  const child = spawn(legacyBin, args, {
    stdio: "inherit"
  });

  child.on("exit", (code) => {
    process.exit(code || 0);
  });
} else {
  // Run TypeScript TUI
  import("../dist/index.js");
}
