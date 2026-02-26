import { parseArgs } from "node:util";
import * as fs from "node:fs";
import * as path from "node:path";
import { execFileSync } from "node:child_process";
import { usageError, preflightError, authError } from "../apperr/errors.js";

const usageText = `Usage:
  sc --branch <branch-name> [--org <org-name>]
  sc --plain [--org <org-name>]

Creates a Sprite environment, installs OpenCode and ast-grep, and copies
~/.local/share/opencode/auth.json plus ~/.config/opencode.

In repo mode (--branch), the CLI also copies .env, clones the current repository,
and checks out (or creates) the target branch.

In plain mode (--plain), the CLI skips all repository and git setup.

Options:
  -b, --branch <branch-name> Branch to check out/create inside sprite
  -p, --plain                Launch a plain sprite with no repository connection
  -o, --org <org-name>       Sprite organization name
  -h, --help                 Show this help`;

export interface Config {
  branch: string;
  plain: boolean;
  org: string;
  showHelp: boolean;
  repoRoot: string;
  repoURL: string;
  repoName: string;
  repoSlug: string;
  branchSlug: string;
  namePrefix: string;
  localAuth: string;
  localConfigDir: string;
  localEnvFile: string;
  spritesToken: string;
}

export function parse(args: string[], env: string[], stderr: NodeJS.WritableStream): Config {
  const cfg: Config = {
    branch: "",
    plain: false,
    org: process.env["SPRITE_ORG"] ?? "",
    showHelp: false,
    repoRoot: "",
    repoURL: "",
    repoName: "",
    repoSlug: "",
    branchSlug: "",
    namePrefix: "",
    localAuth: "",
    localConfigDir: "",
    localEnvFile: "",
    spritesToken: "",
  };

  let parsed;
  try {
    parsed = parseArgs({
      args,
      options: {
        branch: { type: "string", short: "b", default: "" },
        plain: { type: "boolean", short: "p", default: false },
        org: { type: "string", short: "o", default: cfg.org },
        help: { type: "boolean", short: "h", default: false },
      },
      strict: true,
      allowPositionals: false,
    });
  } catch (e) {
    stderr.write(usageText + "\n");
    throw usageError(e instanceof Error ? e.message : String(e));
  }

  cfg.branch = (parsed.values.branch as string) ?? "";
  cfg.plain = (parsed.values.plain as boolean) ?? false;
  cfg.org = (parsed.values.org as string) ?? "";
  cfg.showHelp = (parsed.values.help as boolean) ?? false;

  if (cfg.showHelp) {
    stderr.write(usageText + "\n");
    return cfg;
  }

  cfg.branch = cfg.branch.trim();
  if (cfg.plain && cfg.branch !== "") {
    stderr.write(usageText + "\n");
    throw usageError("--plain cannot be combined with --branch");
  }
  if (!cfg.plain && cfg.branch === "") {
    stderr.write(usageText + "\n");
    throw usageError("missing required argument --branch");
  }

  const home = process.env["HOME"] ?? "";
  cfg.localAuth = path.join(home, ".local", "share", "opencode", "auth.json");
  cfg.localConfigDir = path.join(home, ".config", "opencode");

  if (cfg.plain) {
    cfg.namePrefix = "plain";
  } else {
    const repoRoot = gitOut("rev-parse", "--show-toplevel");
    if (repoRoot === null) {
      throw preflightError("must run inside a git repository");
    }
    const repoURL = gitOut("-C", repoRoot, "config", "--get", "remote.origin.url");
    if (repoURL === null || repoURL.trim() === "") {
      throw preflightError(`unable to determine remote.origin.url for ${repoRoot}`);
    }
    const repoName = path.basename(repoURL).replace(/\.git$/, "");
    if (repoName === "" || repoName === ".") {
      throw preflightError(`unable to derive repository name from ${repoURL}`);
    }

    cfg.repoRoot = repoRoot;
    cfg.repoURL = repoURL;
    cfg.repoName = repoName;
    cfg.repoSlug = truncate(slugify(repoName), 16);
    cfg.branchSlug = truncate(slugify(cfg.branch), 16);
    cfg.namePrefix = (cfg.repoSlug || "repo") + "-" + (cfg.branchSlug || "branch");
    cfg.localEnvFile = path.join(repoRoot, ".env");
  }

  requireFile(cfg.localAuth);
  requireDir(cfg.localConfigDir);
  if (!cfg.plain) {
    requireFile(cfg.localEnvFile);
  }

  const [token, warnDifferent] = resolveToken(env, cfg.localAuth);
  if (warnDifferent) {
    stderr.write("warning: SPRITES_TOKEN differs from auth.json token; preferring SPRITES_TOKEN\n");
  }
  cfg.spritesToken = token;

  return cfg;
}

function gitOut(...args: string[]): string | null {
  try {
    const out = execFileSync("git", args, { encoding: "utf8", stdio: ["pipe", "pipe", "pipe"] });
    return out.trim();
  } catch {
    return null;
  }
}

function requireFile(filePath: string): void {
  let st;
  try {
    st = fs.statSync(filePath);
  } catch {
    throw preflightError(`missing file: ${filePath}`);
  }
  if (st.isDirectory()) {
    throw preflightError(`expected file but found directory: ${filePath}`);
  }
}

function requireDir(dirPath: string): void {
  let st;
  try {
    st = fs.statSync(dirPath);
  } catch {
    throw preflightError(`missing directory: ${dirPath}`);
  }
  if (!st.isDirectory()) {
    throw preflightError(`expected directory but found file: ${dirPath}`);
  }
}

export function resolveToken(env: string[], authPath: string): [string, boolean] {
  return resolveTokenWithSpriteLookup(env, authPath, tokenFromSpritesLogin);
}

export function resolveTokenWithSpriteLookup(
  env: string[],
  authPath: string,
  spriteLookup: (() => string | null) | null,
): [string, boolean] {
  let envToken = "";
  for (const kv of env) {
    if (kv.startsWith("SPRITES_TOKEN=")) {
      envToken = kv.slice("SPRITES_TOKEN=".length);
      break;
    }
  }
  const authToken = tokenFromAuthJSON(authPath);

  if (envToken !== "") {
    if (authToken !== "" && authToken !== envToken) {
      return [envToken, true];
    }
    return [envToken, false];
  }
  if (authToken !== "") {
    return [authToken, false];
  }
  if (spriteLookup !== null) {
    const spriteToken = spriteLookup();
    if (spriteToken !== null && spriteToken.trim() !== "") {
      return [spriteToken.trim(), false];
    }
  }
  throw authError("missing SPRITES_TOKEN and no token found in auth.json or sprites login");
}

interface SpritesConfig {
  current_selection?: {
    url?: string;
    org?: string;
  };
  urls?: Record<string, { orgs?: Record<string, SpritesOrg> }>;
  users?: Array<{ id: string }>;
  current_user?: string;
}

interface SpritesOrg {
  keyring_key?: string;
  use_keyring?: boolean;
  api_token?: string;
  token?: string;
}

function tokenFromSpritesLogin(): string | null {
  const home = (process.env["HOME"] ?? "").trim();
  if (home === "") return null;

  const configPath = path.join(home, ".sprites", "sprites.json");
  let raw: string;
  try {
    raw = fs.readFileSync(configPath, "utf8");
  } catch {
    return null;
  }

  let cfg: SpritesConfig;
  try {
    cfg = JSON.parse(raw);
  } catch {
    return null;
  }

  let url = (cfg.current_selection?.url ?? "").trim();
  if (url === "") url = "https://api.sprites.dev";

  const org = (cfg.current_selection?.org ?? "").trim();
  if (org === "") return null;

  const orgCfg = spritesOrgConfig(cfg, url, org);
  if (orgCfg === null) return null;

  const tok = firstNonEmpty(orgCfg.api_token, orgCfg.token);
  if (tok !== "") return tok;

  let userID = (cfg.current_user ?? "").trim();
  if (userID === "" && cfg.users && cfg.users.length > 0) {
    userID = (cfg.users[0].id ?? "").trim();
  }
  if (userID === "") return null;

  const service = "sprites-cli:" + userID;
  let account = (orgCfg.keyring_key ?? "").trim();
  if (account === "") {
    account = `sprites:org:${url}:${org}`;
  }

  try {
    execFileSync("which", ["security"], { stdio: "pipe" });
  } catch {
    return null;
  }

  let rawToken: string;
  try {
    rawToken = execFileSync(
      "security",
      ["find-generic-password", "-s", service, "-a", account, "-w"],
      { encoding: "utf8", stdio: ["pipe", "pipe", "pipe"] },
    ).trim();
  } catch {
    try {
      rawToken = execFileSync(
        "security",
        ["find-generic-password", "-a", account, "-w"],
        { encoding: "utf8", stdio: ["pipe", "pipe", "pipe"] },
      ).trim();
    } catch {
      return null;
    }
  }

  if (rawToken.startsWith("go-keyring-base64:")) {
    const enc = rawToken.slice("go-keyring-base64:".length);
    rawToken = Buffer.from(enc, "base64").toString("utf8");
  }
  rawToken = rawToken.trim();
  if (rawToken === "") return null;
  return rawToken;
}

function spritesOrgConfig(cfg: SpritesConfig, url: string, org: string): SpritesOrg | null {
  if (cfg.urls) {
    const byURL = cfg.urls[url];
    if (byURL?.orgs?.[org]) {
      return byURL.orgs[org];
    }
    for (const urlEntry of Object.values(cfg.urls)) {
      if (urlEntry.orgs?.[org]) {
        return urlEntry.orgs[org];
      }
    }
  }
  return null;
}

function firstNonEmpty(...values: (string | undefined)[]): string {
  for (const v of values) {
    if (v !== undefined && v.trim() !== "") return v;
  }
  return "";
}

export function tokenFromAuthJSON(authPath: string): string {
  let raw: string;
  try {
    raw = fs.readFileSync(authPath, "utf8");
  } catch {
    return "";
  }

  let generic: Record<string, unknown>;
  try {
    generic = JSON.parse(raw);
  } catch {
    return "";
  }

  const keys = ["sprites_token", "token", "access_token", "api_key"];
  for (const k of keys) {
    const v = generic[k];
    if (typeof v === "string" && v !== "") return v;
  }
  const auth = generic["auth"];
  if (auth !== null && typeof auth === "object") {
    const nested = auth as Record<string, unknown>;
    for (const k of keys) {
      const v = nested[k];
      if (typeof v === "string" && v !== "") return v;
    }
  }
  return "";
}

export function slugify(s: string): string {
  s = s.trim().toLowerCase();
  if (s === "") return "";
  let result = "";
  for (const ch of s) {
    if ((ch >= "a" && ch <= "z") || (ch >= "0" && ch <= "9")) {
      result += ch;
    } else if (ch === "-" || ch === "/" || ch === "." || ch === "_" || ch === " ") {
      result += "-";
    }
  }
  return result.replace(/^-+|-+$/g, "");
}

function truncate(s: string, n: number): string {
  if (s.length <= n) return s;
  return s.slice(0, n);
}
