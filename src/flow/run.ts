import type { Sprite } from "@fly/sprites";
import { isAppError, cleanupError } from "../apperr/errors.js";
import type { Config } from "../cli/config.js";
import { Client, isNameCollision } from "../sprites/client.js";
import { bootstrap, transfer, gitSetup, launch } from "../remote/index.js";

const adjectives = ["amber", "brisk", "cedar", "daring", "ember", "frosty", "golden", "hazel", "ivory", "jade", "lunar", "misty", "nimble", "ochre", "quiet", "rapid", "silver", "sunny", "vivid", "wild"];
const nouns = ["badger", "canyon", "comet", "delta", "dune", "falcon", "forest", "harbor", "meadow", "mesa", "orbit", "otter", "pine", "quill", "ridge", "river", "summit", "thicket", "valley", "willow"];

export async function run(cfg: Config, stdout: NodeJS.WritableStream, stderr: NodeJS.WritableStream): Promise<void> {
  if (cfg.plain) {
    logf(stderr, "preflight", "start", "mode=plain");
  } else {
    logf(stderr, "preflight", "start", `repository=${cfg.repoName} branch=${cfg.branch}`);
  }

  const client = new Client(cfg.spritesToken, cfg.org);

  const name = await chooseName(client, cfg.namePrefix);
  logf(stderr, "sprite", "create", `name=${name}`);

  const sp = await createWithRetry(client, name, cfg.namePrefix);
  let created = true;

  const cleanupOnErr = async (cause: Error): Promise<never> => {
    if (!created) throw cause;
    logf(stderr, "cleanup", "delete", `sprite=${sp.name}`);
    try {
      await client.deleteSprite(sp.name);
    } catch (delErr) {
      throw cleanupError(`${cause.message} (cleanup failed: ${delErr})`, cause);
    }
    throw cause;
  };

  try {
    logf(stderr, "bootstrap", "install", `sprite=${sp.name}`);
    await bootstrap(sp, stdout, stderr);

    logf(stderr, "transfer", "start", `sprite=${sp.name}`);
    const workDir = await transfer(sp, {
      localAuth: cfg.localAuth,
      localEnv: cfg.localEnvFile,
      localConfigDir: cfg.localConfigDir,
    });

    if (cfg.plain) {
      logf(stderr, "git", "skip", "mode=plain");
    } else {
      logf(stderr, "git", "setup", `repo=${sanitizeRepoURL(cfg.repoURL)}`);
      await gitSetup(sp, cfg.repoURL, workDir, cfg.branch, stdout, stderr);
    }

    const org = sp.organizationName ?? client.organization ?? "";
    logf(stderr, "launch", "start", `sprite_id=${sp.name}`);
    await launch(sp, workDir, org, stdout, stderr);
  } catch (err) {
    await cleanupOnErr(err instanceof Error ? err : new Error(String(err)));
  }
}

async function createWithRetry(client: Client, firstName: string, prefix: string): Promise<Sprite> {
  let name = firstName;
  for (let attempt = 1; attempt <= 50; attempt++) {
    try {
      return await client.createSprite(name);
    } catch (createErr) {
      if (!isNameCollision(createErr)) {
        throw new Error(`create sprite: ${createErr instanceof Error ? createErr.message : String(createErr)}`);
      }
      name = randomSpriteName(prefix);
    }
  }
  throw new Error("unable to create unique sprite after 50 attempts");
}

async function chooseName(client: Client, prefix: string): Promise<string> {
  const existing = await client.listSpriteNames();
  const seen = new Set(existing);
  for (let i = 0; i < 50; i++) {
    const cand = randomSpriteName(prefix);
    if (!seen.has(cand)) return cand;
  }
  throw new Error("unable to generate a unique sprite name after 50 attempts");
}

export function randomSpriteName(prefix: string): string {
  const a = adjectives[Math.floor(Math.random() * adjectives.length)];
  const n = nouns[Math.floor(Math.random() * nouns.length)];
  if (prefix === "") return `${a}-${n}`;
  return `${prefix}-${a}-${n}`;
}

function logf(w: NodeJS.WritableStream, phase: string, step: string, msg: string): void {
  w.write(`phase=${phase} step=${step} ${msg}\n`);
}

export function sanitizeRepoURL(raw: string): string {
  try {
    const u = new URL(raw);
    u.username = "";
    u.password = "";
    return u.toString().trim();
  } catch {
    // Not a valid URL, try scp-style
  }
  const trimmed = raw.trim();
  const at = trimmed.indexOf("@");
  const colon = trimmed.indexOf(":");
  if (at > 0 && colon > at) {
    return trimmed.slice(at + 1);
  }
  return trimmed;
}

export function exitCode(err: unknown): number {
  if (err == null) return 0;
  if (isAppError(err, "usage")) return 2;
  if (isAppError(err, "preflight")) return 3;
  if (isAppError(err, "auth")) return 4;
  if (isAppError(err, "cleanup")) return 5;
  return 1;
}
