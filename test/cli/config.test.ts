import { describe, it, beforeEach } from "node:test";
import * as assert from "node:assert/strict";
import * as fs from "node:fs";
import * as path from "node:path";
import * as os from "node:os";
import { parse, slugify, resolveTokenWithSpriteLookup, tokenFromAuthJSON } from "../../src/cli/config.js";
import { AppError } from "../../src/apperr/errors.js";

const noSpriteLookup = (): string | null => null;

function makeTmpDir(): string {
  return fs.mkdtempSync(path.join(os.tmpdir(), "config-test-"));
}

describe("slugify", () => {
  it("converts mixed case with separators", () => {
    const got = slugify("Feat/My.Change Test");
    assert.equal(got, "feat-my-change-test");
  });
});

describe("resolveToken", () => {
  it("prefers env token with warning", () => {
    const dir = makeTmpDir();
    const authPath = path.join(dir, "auth.json");
    fs.writeFileSync(authPath, JSON.stringify({ token: "from-file" }));

    const [token, warn] = resolveTokenWithSpriteLookup(["SPRITES_TOKEN=from-env"], authPath, noSpriteLookup);
    assert.equal(token, "from-env");
    assert.equal(warn, true);
  });

  it("falls back to auth.json", () => {
    const dir = makeTmpDir();
    const authPath = path.join(dir, "auth.json");
    fs.writeFileSync(authPath, JSON.stringify({ token: "from-file" }));

    const [token, warn] = resolveTokenWithSpriteLookup([], authPath, noSpriteLookup);
    assert.equal(token, "from-file");
    assert.equal(warn, false);
  });

  it("errors when missing everywhere", () => {
    const dir = makeTmpDir();
    const authPath = path.join(dir, "auth.json");
    fs.writeFileSync(authPath, JSON.stringify({ other: "value" }));

    assert.throws(
      () => resolveTokenWithSpriteLookup([], authPath, noSpriteLookup),
      (err: unknown) => err instanceof AppError && err.kind === "auth",
    );
  });

  it("falls back to sprites login", () => {
    const dir = makeTmpDir();
    const authPath = path.join(dir, "auth.json");
    fs.writeFileSync(authPath, JSON.stringify({ other: "value" }));

    const [token, warn] = resolveTokenWithSpriteLookup([], authPath, () => "from-sprites-login");
    assert.equal(token, "from-sprites-login");
    assert.equal(warn, false);
  });
});

describe("tokenFromAuthJSON", () => {
  it("reads nested auth object", () => {
    const dir = makeTmpDir();
    const authPath = path.join(dir, "auth.json");
    fs.writeFileSync(authPath, JSON.stringify({ auth: { access_token: "nested-token" } }));

    const token = tokenFromAuthJSON(authPath);
    assert.equal(token, "nested-token");
  });
});

describe("parse", () => {
  let origHome: string | undefined;
  let home: string;

  beforeEach(() => {
    origHome = process.env["HOME"];
    home = makeTmpDir();
    process.env["HOME"] = home;

    const authDir = path.join(home, ".local", "share", "opencode");
    fs.mkdirSync(authDir, { recursive: true });
    fs.writeFileSync(path.join(authDir, "auth.json"), JSON.stringify({ token: "from-file" }));

    const configDir = path.join(home, ".config", "opencode");
    fs.mkdirSync(configDir, { recursive: true });
  });

  // Restore HOME after each test
  // Note: node:test beforeEach doesn't have afterEach paired, so we use a manual approach
  function restoreHome() {
    if (origHome !== undefined) {
      process.env["HOME"] = origHome;
    }
  }

  it("plain mode skips repo checks", () => {
    const devNull = { write: () => true } as unknown as NodeJS.WritableStream;
    const cfg = parse(["--plain"], ["SPRITES_TOKEN=from-env"], devNull);
    assert.equal(cfg.plain, true);
    assert.equal(cfg.namePrefix, "plain");
    assert.equal(cfg.localEnvFile, "");
    assert.equal(cfg.repoRoot, "");
    assert.equal(cfg.repoURL, "");
    assert.equal(cfg.repoName, "");
    restoreHome();
  });

  it("plain rejects branch", () => {
    const devNull = { write: () => true } as unknown as NodeJS.WritableStream;
    assert.throws(
      () => parse(["--plain", "--branch", "feat/test"], ["SPRITES_TOKEN=from-env"], devNull),
      (err: unknown) => err instanceof AppError && err.kind === "usage",
    );
    restoreHome();
  });
});
