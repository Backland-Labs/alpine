#!/usr/bin/env node

import { parse } from "./cli/config.js";
import { run, exitCode } from "./flow/run.js";
import { isAppError } from "./apperr/errors.js";

async function main(): Promise<void> {
  let cfg;
  try {
    cfg = parse(process.argv.slice(2), Object.entries(process.env).map(([k, v]) => `${k}=${v ?? ""}`), process.stderr);
  } catch (err) {
    process.stderr.write(err instanceof Error ? err.message + "\n" : String(err) + "\n");
    process.exit(exitCode(err));
  }
  if (cfg.showHelp) {
    process.exit(0);
  }

  try {
    await run(cfg, process.stdout, process.stderr);
  } catch (err) {
    if (!isAppError(err, "silent")) {
      process.stderr.write(err instanceof Error ? err.message + "\n" : String(err) + "\n");
    }
    process.exit(exitCode(err));
  }
}

main();
