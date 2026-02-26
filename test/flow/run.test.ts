import { describe, it } from "node:test";
import * as assert from "node:assert/strict";
import { randomSpriteName, sanitizeRepoURL } from "../../src/flow/run.js";

describe("randomSpriteName", () => {
  it("includes prefix", () => {
    const name = randomSpriteName("repo-branch");
    assert.ok(name.startsWith("repo-branch-"), `expected prefix, got ${name}`);
    assert.ok(name.split("-").length >= 4, `expected adjective/noun suffix, got ${name}`);
  });

  it("works without prefix", () => {
    const name = randomSpriteName("");
    assert.ok(!name.startsWith("-"), `invalid name formatting: ${name}`);
    assert.ok(!name.endsWith("-"), `invalid name formatting: ${name}`);
    assert.equal(name.split("-").length, 2, `expected adjective-noun form, got ${name}`);
  });
});

describe("sanitizeRepoURL", () => {
  it("removes userinfo from HTTP URL", () => {
    const g = sanitizeRepoURL("https://user:secret@example.com/repo.git");
    assert.equal(g, "https://example.com/repo.git");
  });

  it("removes user from scp-style URL", () => {
    const g = sanitizeRepoURL("git@github.com:owner/repo.git");
    assert.equal(g, "github.com:owner/repo.git");
  });
});
