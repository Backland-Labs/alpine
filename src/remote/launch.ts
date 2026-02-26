import { execFileSync, spawn as cpSpawn } from "node:child_process";
import * as tty from "node:tty";
import type { Sprite } from "@fly/sprites";
import { shellQuote } from "./utils.js";

const directLaunchScript = `. ~/.zprofile >/dev/null 2>&1 || true; . ~/.zshrc >/dev/null 2>&1 || true; cd "$SPRITE_WORK_DIR" >/dev/null 2>&1 || true; hash -r 2>/dev/null || true; if command -v opencode >/dev/null 2>&1; then exec opencode; fi; exec "$HOME/.opencode/bin/opencode"`;

const expectLaunchScript = `set timeout 20
set cmd [list sprite]
if {[info exists env(SPRITE_ORG)] && $env(SPRITE_ORG) ne ""} {
  lappend cmd -o $env(SPRITE_ORG)
}
lappend cmd console -s $env(SPRITE_NAME)
spawn {*}$cmd
after 1200
send -- ". ~/.profile >/dev/null 2>&1 || true; cd \\"$env(SPRITE_WORK_DIR)\\" >/dev/null 2>&1 || true; hash -r 2>/dev/null || true; opencode\\r"
interact`;

export async function launch(
  sp: Sprite,
  workDir: string,
  org: string,
  stdout: NodeJS.WritableStream,
  stderr: NodeJS.WritableStream,
): Promise<void> {
  if (!isTTY()) {
    const reconnect = reconnectCommand(sp.name, org, workDir);
    stdout.write(`status=ready\nsprite_id=${sp.name}\nreconnect=${reconnect}\n`);
    return;
  }

  if (hasExpect()) {
    try {
      await runExpect(sp.name, org, workDir);
      return;
    } catch {
      stderr.write("Automatic console launch failed, falling back to direct launch.\n");
    }
  } else {
    stderr.write("'expect' is not installed, falling back to direct launch.\n");
  }

  const args: string[] = [];
  if (org !== "") {
    args.push("-o", org);
  }
  args.push(
    "exec",
    "-s", sp.name,
    "-env", `SPRITE_WORK_DIR=${workDir}`,
    "-tty",
    "zsh",
    "-lc",
    directLaunchScript,
  );

  await new Promise<void>((resolve, reject) => {
    const child = cpSpawn("sprite", args, {
      stdio: "inherit",
    });
    child.on("error", reject);
    child.on("close", (code) => {
      if (code !== 0) {
        reject(new Error(`launch opencode: process exited with code ${code}`));
      } else {
        resolve();
      }
    });
  });
}

function reconnectCommand(spriteName: string, org: string, workDir: string): string {
  let cmd = "sprite";
  if (org !== "") {
    cmd += " -o " + shellQuote(org);
  }
  cmd += " exec -s " + shellQuote(spriteName);
  cmd += " -env " + shellQuote("SPRITE_WORK_DIR=" + workDir);
  cmd += " -tty zsh -lc " + shellQuote(launchScriptWithWorkDir(workDir));
  return cmd;
}

function isTTY(): boolean {
  return tty.isatty(0) && tty.isatty(1);
}

function hasExpect(): boolean {
  try {
    execFileSync("which", ["expect"], { stdio: "pipe" });
    return true;
  } catch {
    return false;
  }
}

function runExpect(spriteName: string, org: string, workDir: string): Promise<void> {
  return new Promise((resolve, reject) => {
    const child = cpSpawn("expect", ["-c", expectLaunchScript], {
      stdio: "inherit",
      env: {
        ...process.env,
        SPRITE_NAME: spriteName,
        SPRITE_ORG: org,
        SPRITE_WORK_DIR: workDir,
      },
    });
    child.on("error", reject);
    child.on("close", (code) => {
      if (code !== 0) {
        reject(new Error(`expect exited with code ${code}`));
      } else {
        resolve();
      }
    });
  });
}

function launchScriptWithWorkDir(workDir: string): string {
  return `. ~/.zprofile >/dev/null 2>&1 || true; . ~/.zshrc >/dev/null 2>&1 || true; cd ${shellQuote(workDir)} >/dev/null 2>&1 || true; hash -r 2>/dev/null || true; if command -v opencode >/dev/null 2>&1; then exec opencode; fi; exec "$HOME/.opencode/bin/opencode"`;
}
