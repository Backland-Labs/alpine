import { describe, it } from "node:test";
import * as assert from "node:assert/strict";
import { repoNameFromLocalEnv } from "../../src/remote/transfer.js";

describe("repoNameFromLocalEnv", () => {
  it("extracts repo name from env path", () => {
    const name = repoNameFromLocalEnv("/Users/max/code/alpine/.env");
    assert.equal(name, "alpine");
  });

  it("rejects empty path", () => {
    assert.throws(
      () => repoNameFromLocalEnv(""),
      (err: unknown) => err instanceof Error,
    );
  });
});
