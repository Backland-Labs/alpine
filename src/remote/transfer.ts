import * as fs from "node:fs";
import * as path from "node:path";
import { execFileSync } from "node:child_process";
import type { Sprite } from "@fly/sprites";
import { shellQuote } from "./utils.js";

export interface TransferConfig {
  localAuth: string;
  localEnv: string;
  localConfigDir: string;
}

export async function transfer(sp: Sprite, cfg: TransferConfig): Promise<string> {
  const home = await remoteHome(sp);
  const localEnv = cfg.localEnv.trim();

  const remoteFS = sp.filesystem("/");
  const remoteAuth = path.posix.join(home, ".local", "share", "opencode", "auth.json");
  const remoteConfigParent = path.posix.join(home, ".config");
  const remoteConfigRoot = path.posix.join(remoteConfigParent, path.basename(path.resolve(cfg.localConfigDir)));
  const remoteTmpTar = `/tmp/opencode-config-${Date.now()}-${process.pid}.tar.gz`;

  await sp.exec(`sh -lc 'mkdir -p "$HOME/.local/share/opencode" "$HOME/.config"'`);

  const authBytes = fs.readFileSync(cfg.localAuth);
  await remoteFS.writeFile(remoteAuth, authBytes, { mode: 0o600 });
  await remoteFS.chmod(remoteAuth, 0o600);

  if (localEnv !== "") {
    const remoteEnv = path.posix.join(home, ".env");
    const envBytes = fs.readFileSync(localEnv);
    await remoteFS.writeFile(remoteEnv, envBytes, { mode: 0o600 });
    await remoteFS.chmod(remoteEnv, 0o600);
  }

  const configTar = packLocalConfigTar(cfg.localConfigDir);
  await remoteFS.writeFile(remoteTmpTar, configTar, { mode: 0o600 });
  try { await remoteFS.chmod(remoteTmpTar, 0o600); } catch { /* ignore */ }

  const extractResult = await sp.exec(
    `sh -lc ${shellQuote("tar -xzf " + shellQuote(remoteTmpTar) + " -C " + shellQuote(remoteConfigParent) + " && rm -f " + shellQuote(remoteTmpTar))}`,
  );
  if (extractResult.exitCode !== 0) {
    const out = String(extractResult.stderr).trim();
    throw new Error(`extract remote config tar: exit ${extractResult.exitCode}: ${out}`);
  }

  await remoteFS.stat(remoteConfigRoot);

  let workDir = home;
  if (localEnv !== "") {
    const repoName = repoNameFromLocalEnv(localEnv);
    const remoteRepoParent = path.posix.join(home, "code");
    const remoteRepoDir = path.posix.join(remoteRepoParent, repoName);
    await remoteFS.mkdir(remoteRepoParent, { recursive: true });
    workDir = remoteRepoDir;
  }

  return workDir;
}

function packLocalConfigTar(localConfigDir: string): Buffer {
  localConfigDir = path.resolve(localConfigDir);
  const configParent = path.dirname(localConfigDir);
  const configName = path.basename(localConfigDir);

  const tmpTarPath = path.join(
    (process.env["TMPDIR"] ?? "/tmp"),
    `opencode-config-${Date.now()}-${process.pid}.tar.gz`,
  );

  try {
    runTarWithFallback(configParent, configName, tmpTarPath);
    return fs.readFileSync(tmpTarPath);
  } finally {
    try { fs.unlinkSync(tmpTarPath); } catch { /* ignore */ }
  }
}

function runTarWithFallback(configParent: string, configName: string, tarPath: string): void {
  try {
    execFileSync("tar", ["--no-xattrs", "--no-mac-metadata", "-C", configParent, "-czf", tarPath, configName], {
      stdio: "pipe",
    });
    return;
  } catch {
    // fall through to basic tar
  }

  try {
    execFileSync("tar", ["-C", configParent, "-czf", tarPath, configName], {
      stdio: "pipe",
    });
  } catch (e) {
    throw new Error(`pack local config directory: ${e instanceof Error ? e.message : String(e)}`);
  }
}

export function repoNameFromLocalEnv(localEnvPath: string): string {
  if (localEnvPath === "") {
    throw new Error(`unable to derive repo name from local env path: ${localEnvPath}`);
  }
  const name = path.basename(path.dirname(path.resolve(localEnvPath)));
  if (name === "" || name === "." || name === path.sep) {
    throw new Error(`unable to derive repo name from local env path: ${localEnvPath}`);
  }
  return name;
}

async function remoteHome(sp: Sprite): Promise<string> {
  const result = await sp.exec(`sh -lc 'printf %s "$HOME"'`);
  const h = String(result.stdout).trim();
  if (h === "") {
    throw new Error("remote HOME is empty");
  }
  return h;
}
