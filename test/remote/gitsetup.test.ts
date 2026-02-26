import { describe, it } from "node:test";
import * as assert from "node:assert/strict";
import * as fs from "node:fs";
import * as path from "node:path";
import * as os from "node:os";
import { execFileSync } from "node:child_process";
import { gitSetupScript } from "../../src/remote/gitsetup.js";

interface GitSetupResult {
  askpassPath: string;
  terminalPrompt: string;
  output: string;
  err: Error | null;
}

function runGitSetupScript(repoURL: string, envContent: string): GitSetupResult {
  const home = fs.mkdtempSync(path.join(os.tmpdir(), "gitsetup-test-"));
  const repoDir = path.join(home, "code", "repo");
  fs.mkdirSync(path.join(repoDir, ".git"), { recursive: true });

  if (envContent !== "") {
    fs.writeFileSync(path.join(home, ".env"), envContent);
  }

  const binDir = path.join(home, "bin");
  fs.mkdirSync(binDir, { recursive: true });

  const fakeGit = `#!/bin/sh
printf '%s' "\${GIT_ASKPASS:-}" > "$HOME/last-git-askpass"
printf '%s' "\${GIT_TERMINAL_PROMPT:-}" > "$HOME/last-git-terminal-prompt"
exit 0
`;
  fs.writeFileSync(path.join(binDir, "git"), fakeGit, { mode: 0o700 });

  let output = "";
  let err: Error | null = null;
  try {
    output = execFileSync("sh", ["-c", gitSetupScript], {
      encoding: "utf8",
      env: {
        HOME: home,
        PATH: [binDir, "/usr/bin", "/bin", "/usr/sbin", "/sbin"].join(":"),
        SPRITE_REPO_URL: repoURL,
        SPRITE_REPO_DIR: repoDir,
        SPRITE_TARGET_BRANCH: "feat/test-token-auth",
      },
    }).trim();
  } catch (e) {
    err = e instanceof Error ? e : new Error(String(e));
  }

  let askpassPath = "";
  let terminalPrompt = "";
  try { askpassPath = fs.readFileSync(path.join(home, "last-git-askpass"), "utf8").trim(); } catch { /* */ }
  try { terminalPrompt = fs.readFileSync(path.join(home, "last-git-terminal-prompt"), "utf8").trim(); } catch { /* */ }

  return { askpassPath, terminalPrompt, output, err };
}

describe("gitSetupScript", () => {
  it("uses GH_TOKEN for GitHub HTTPS", () => {
    const result = runGitSetupScript(
      "https://github.com/example/private-repo.git",
      "GH_TOKEN=test-gh-token\n",
    );
    assert.equal(result.err, null, `expected success, got: ${result.err?.message}\n${result.output}`);
    assert.ok(result.askpassPath !== "", "expected GIT_ASKPASS to be set");
    assert.equal(result.terminalPrompt, "0");

    // Verify askpass helper was cleaned up
    assert.ok(!fs.existsSync(result.askpassPath), "expected askpass helper to be cleaned up");
  });

  it("uses GITHUB_TOKEN fallback", () => {
    const result = runGitSetupScript(
      "https://github.com/example/private-repo.git",
      "GITHUB_TOKEN=test-github-token\n",
    );
    assert.equal(result.err, null, `expected success, got: ${result.err?.message}\n${result.output}`);
    assert.ok(result.askpassPath !== "", "expected GIT_ASKPASS to be set");
  });

  it("skips askpass for non-GitHub remote", () => {
    const result = runGitSetupScript(
      "https://gitlab.com/example/private-repo.git",
      "GH_TOKEN=test-gh-token\n",
    );
    assert.equal(result.err, null, `expected success, got: ${result.err?.message}\n${result.output}`);
    assert.equal(result.askpassPath, "", "expected empty GIT_ASKPASS");
    assert.equal(result.terminalPrompt, "", "expected empty GIT_TERMINAL_PROMPT");
  });
});
