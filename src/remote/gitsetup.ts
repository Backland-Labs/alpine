import type { Sprite } from "@fly/sprites";

export const gitSetupScript = `set -e

if [ -f "$HOME/.env" ]; then
  set -a
  . "$HOME/.env"
  set +a
fi

SPRITE_GITHUB_TOKEN="\${GH_TOKEN:-\${GITHUB_TOKEN:-}}"
SPRITE_GIT_ASKPASS=""

cleanup_git_auth() {
  if [ -n "$SPRITE_GIT_ASKPASS" ] && [ -f "$SPRITE_GIT_ASKPASS" ]; then
    rm -f "$SPRITE_GIT_ASKPASS"
  fi
}
trap cleanup_git_auth EXIT

case "$SPRITE_REPO_URL" in
  https://github.com/*|https://*.github.com/*)
    if [ -n "$SPRITE_GITHUB_TOKEN" ]; then
      SPRITE_GIT_ASKPASS="$(mktemp /tmp/git-askpass-XXXXXX)"
      cat >"$SPRITE_GIT_ASKPASS" <<'EOF'
#!/bin/sh
case "$1" in
  *sername*) printf '%s\\n' "x-access-token" ;;
  *assword*) printf '%s\\n' "$SPRITE_GITHUB_TOKEN" ;;
  *) printf '\\n' ;;
esac
EOF
      chmod 700 "$SPRITE_GIT_ASKPASS"
      export SPRITE_GITHUB_TOKEN
      export GIT_ASKPASS="$SPRITE_GIT_ASKPASS"
      export GIT_TERMINAL_PROMPT=0
    fi
    ;;
esac

if ! command -v git >/dev/null 2>&1; then
  printf "git is required inside sprite but was not found.\\n" >&2
  exit 1
fi

mkdir -p "$(dirname "$SPRITE_REPO_DIR")"
if [ ! -d "$SPRITE_REPO_DIR/.git" ]; then
  git clone "$SPRITE_REPO_URL" "$SPRITE_REPO_DIR"
fi

cd "$SPRITE_REPO_DIR"
origin_url="$(git remote get-url origin 2>/dev/null || true)"
if [ "$origin_url" != "$SPRITE_REPO_URL" ]; then
  git remote set-url origin "$SPRITE_REPO_URL"
fi

git fetch origin --prune

if git show-ref --verify --quiet "refs/heads/$SPRITE_TARGET_BRANCH"; then
  printf "branch_path=local\\n"
  git checkout "$SPRITE_TARGET_BRANCH"
elif git ls-remote --exit-code --heads origin "$SPRITE_TARGET_BRANCH" >/dev/null 2>&1; then
  printf "branch_path=remote_tracking\\n"
  git checkout -b "$SPRITE_TARGET_BRANCH" --track "origin/$SPRITE_TARGET_BRANCH"
else
  default_remote_head="$(git symbolic-ref --quiet --short refs/remotes/origin/HEAD 2>/dev/null || true)"
  if [ -n "$default_remote_head" ]; then
    printf "branch_path=from_origin_head\\n"
    git checkout -b "$SPRITE_TARGET_BRANCH" "$default_remote_head"
  elif git show-ref --verify --quiet refs/remotes/origin/main; then
    printf "branch_path=from_origin_main\\n"
    git checkout -b "$SPRITE_TARGET_BRANCH" origin/main
  elif git show-ref --verify --quiet refs/remotes/origin/master; then
    printf "branch_path=from_origin_master\\n"
    git checkout -b "$SPRITE_TARGET_BRANCH" origin/master
  else
    printf "Unable to determine base branch (origin/HEAD, origin/main, origin/master).\\n" >&2
    printf "Remediation: set origin HEAD or create origin/main or origin/master, then rerun.\\n" >&2
    exit 1
  fi
fi

if git ls-remote --exit-code --heads origin "$SPRITE_TARGET_BRANCH" >/dev/null 2>&1; then
  git branch --set-upstream-to="origin/$SPRITE_TARGET_BRANCH" "$SPRITE_TARGET_BRANCH" >/dev/null 2>&1 || true
fi
`;

export async function gitSetup(
  sp: Sprite,
  repoURL: string,
  repoDir: string,
  branch: string,
  stdout: NodeJS.WritableStream,
  stderr: NodeJS.WritableStream,
): Promise<void> {
  const cmd = sp.spawn("zsh", ["-lc", gitSetupScript], {
    env: {
      SPRITE_REPO_URL: repoURL,
      SPRITE_REPO_DIR: repoDir,
      SPRITE_TARGET_BRANCH: branch,
    },
  });
  await cmd.start();

  cmd.stdout.on("data", (chunk: Buffer) => stdout.write(chunk));
  cmd.stderr.on("data", (chunk: Buffer) => stderr.write(chunk));

  const exitCode = await cmd.wait();
  if (exitCode !== 0) {
    throw new Error(`git setup: process exited with code ${exitCode}`);
  }
}
