#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: setup-sprite-opencode.sh --branch <branch-name> [--org <org-name>]

Creates a Sprite environment with a repo/branch-tagged random name, installs OpenCode and ast-grep inside it,
copies ~/.local/share/opencode/auth.json, copies ~/.config/opencode,
copies .env into the sprite and loads all variables,
clones the current repository inside the sprite,
checks out or creates the requested branch,
then opens sprite console and launches opencode.

Options:
  -b, --branch <branch-name> Branch to check out/create inside sprite (required)
  -o, --org <org-name>   Sprite organization name
  -h, --help             Show this help

Environment:
  SPRITE_ORG=<org-name>          Default organization when --org is omitted
EOF
}

sprite_org="${SPRITE_ORG:-}"
target_branch=""

adjectives=(
  amber
  brisk
  cedar
  daring
  ember
  frosty
  golden
  hazel
  ivory
  jade
  lunar
  misty
  nimble
  ochre
  quiet
  rapid
  silver
  sunny
  vivid
  wild
)

nouns=(
  badger
  canyon
  comet
  delta
  dune
  falcon
  forest
  harbor
  meadow
  mesa
  orbit
  otter
  pine
  quill
  ridge
  river
  summit
  thicket
  valley
  willow
)

random_sprite_name() {
  local adjective noun
  adjective="${adjectives[RANDOM % ${#adjectives[@]}]}"
  noun="${nouns[RANDOM % ${#nouns[@]}]}"
  if [[ -n "${sprite_name_prefix:-}" ]]; then
    printf '%s-%s-%s' "$sprite_name_prefix" "$adjective" "$noun"
  else
    printf '%s-%s' "$adjective" "$noun"
  fi
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    -b|--branch)
      if [[ $# -lt 2 ]]; then
        printf "Missing value for %s\n" "$1" >&2
        exit 1
      fi
      target_branch="$2"
      shift 2
      ;;
    -o|--org)
      if [[ $# -lt 2 ]]; then
        printf "Missing value for %s\n" "$1" >&2
        exit 1
      fi
      sprite_org="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      printf "Unexpected argument: %s\n" "$1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [[ -z "$target_branch" ]]; then
  printf "Missing required argument: --branch <branch-name>\n" >&2
  usage >&2
  exit 1
fi

if ! repo_root=$(git rev-parse --show-toplevel 2>/dev/null); then
  printf "This script must be run from inside a git repository.\n" >&2
  exit 1
fi

repo_url=$(git -C "$repo_root" config --get remote.origin.url || true)
if [[ -z "$repo_url" ]]; then
  printf "Unable to determine remote.origin.url for repo: %s\n" "$repo_root" >&2
  exit 1
fi

repo_name="${repo_url##*/}"
repo_name="${repo_name%.git}"
if [[ -z "$repo_name" ]]; then
  printf "Unable to derive repository name from URL: %s\n" "$repo_url" >&2
  exit 1
fi

repo_slug=$(printf '%s' "$repo_name" | tr '[:upper:]' '[:lower:]' | tr '/._ ' '-' | tr -cd 'a-z0-9-')
branch_slug=$(printf '%s' "$target_branch" | tr '[:upper:]' '[:lower:]' | tr '/._ ' '-' | tr -cd 'a-z0-9-')
repo_slug="${repo_slug:0:16}"
branch_slug="${branch_slug:0:16}"
sprite_name_prefix="${repo_slug:-repo}-${branch_slug:-branch}"

local_auth="$HOME/.local/share/opencode/auth.json"
local_config_dir="$HOME/.config/opencode"
local_env_file="$repo_root/.env"

if [[ ! -f "$local_auth" ]]; then
  printf "Missing file: %s\n" "$local_auth" >&2
  exit 1
fi

if [[ ! -d "$local_config_dir" ]]; then
  printf "Missing directory: %s\n" "$local_config_dir" >&2
  exit 1
fi

if [[ ! -f "$local_env_file" ]]; then
  printf "Missing file: %s\n" "$local_env_file" >&2
  exit 1
fi

sprite_cmd=(sprite)
if [[ -n "$sprite_org" ]]; then
  sprite_cmd+=(-o "$sprite_org")
fi

existing_sprites=()
while IFS= read -r name; do
  if [[ -n "$name" ]]; then
    existing_sprites+=("$name")
  fi
done < <("${sprite_cmd[@]}" list)

sprite_name=""
for ((attempt = 1; attempt <= 50; attempt++)); do
  candidate_name="$(random_sprite_name)"
  name_taken=0

  for existing_name in "${existing_sprites[@]-}"; do
    if [[ "$existing_name" == "$candidate_name" ]]; then
      name_taken=1
      break
    fi
  done

  if [[ $name_taken -eq 0 ]]; then
    sprite_name="$candidate_name"
    break
  fi
done

if [[ -z "$sprite_name" ]]; then
  printf "Unable to generate a unique sprite name after 50 attempts.\n" >&2
  exit 1
fi

printf "Creating sprite '%s'...\n" "$sprite_name"
"${sprite_cmd[@]}" create -skip-console "$sprite_name"

printf "Installing OpenCode and ast-grep inside sprite...\n"
"${sprite_cmd[@]}" exec -s "$sprite_name" zsh -lc '
set -e
opencode_bin_dir="$HOME/.opencode/bin"
local_bin_dir="$HOME/.local/bin"
opencode_path_line='\''export PATH="$HOME/.opencode/bin:$PATH"'\''
local_bin_path_line='\''export PATH="$HOME/.local/bin:$PATH"'\''
env_line='\''if [ -f "$HOME/.env" ]; then set -a; . "$HOME/.env"; set +a; fi'\''

mkdir -p "$local_bin_dir"
export PATH="$local_bin_dir:$opencode_bin_dir:$PATH"

if [ ! -x "$HOME/.opencode/bin/opencode" ] && ! command -v opencode >/dev/null 2>&1; then
  curl -fsSL https://opencode.ai/install | bash
fi

install_ast_grep() {
  if command -v ast-grep >/dev/null 2>&1 || command -v sg >/dev/null 2>&1; then
    return 0
  fi

  if command -v npm >/dev/null 2>&1; then
    if npm i -g @ast-grep/cli --prefix "$HOME/.local"; then
      hash -r 2>/dev/null || true
      if command -v ast-grep >/dev/null 2>&1 || command -v sg >/dev/null 2>&1; then
        return 0
      fi
    fi
  fi

  if command -v pip3 >/dev/null 2>&1; then
    if pip3 install --user ast-grep-cli; then
      hash -r 2>/dev/null || true
      if command -v ast-grep >/dev/null 2>&1 || command -v sg >/dev/null 2>&1; then
        return 0
      fi
    fi
  fi

  if command -v pip >/dev/null 2>&1; then
    if pip install --user ast-grep-cli; then
      hash -r 2>/dev/null || true
      if command -v ast-grep >/dev/null 2>&1 || command -v sg >/dev/null 2>&1; then
        return 0
      fi
    fi
  fi

  platform="$(uname -s)"
  arch="$(uname -m)"
  target=""

  if [ "$platform" = "Linux" ]; then
    case "$arch" in
      x86_64|amd64)
        target="x86_64-unknown-linux-gnu"
        ;;
      aarch64|arm64)
        target="aarch64-unknown-linux-gnu"
        ;;
    esac
  elif [ "$platform" = "Darwin" ]; then
    case "$arch" in
      x86_64|amd64)
        target="x86_64-apple-darwin"
        ;;
      aarch64|arm64)
        target="aarch64-apple-darwin"
        ;;
    esac
  fi

  if [ -z "$target" ]; then
    printf "Unable to install ast-grep for %s/%s.\n" "$platform" "$arch" >&2
    return 1
  fi

  tmp_zip="/tmp/ast-grep-$RANDOM-$RANDOM.zip"
  curl -fsSL -o "$tmp_zip" "https://github.com/ast-grep/ast-grep/releases/latest/download/app-$target.zip"

  if command -v unzip >/dev/null 2>&1; then
    unzip -qo "$tmp_zip" -d "$local_bin_dir"
  elif command -v python3 >/dev/null 2>&1; then
    python3 -c "import sys,zipfile; z=zipfile.ZipFile(sys.argv[1]); [z.extract(name, sys.argv[2]) for name in (\"ast-grep\",\"sg\")]" "$tmp_zip" "$local_bin_dir"
  else
    rm -f "$tmp_zip"
    printf "Need unzip or python3 to install ast-grep.\n" >&2
    return 1
  fi

  rm -f "$tmp_zip"
  chmod +x "$local_bin_dir/ast-grep" "$local_bin_dir/sg" 2>/dev/null || true
  hash -r 2>/dev/null || true

  command -v ast-grep >/dev/null 2>&1 || command -v sg >/dev/null 2>&1
}

install_ast_grep

for rc_file in "$HOME/.zshrc" "$HOME/.zprofile" "$HOME/.bashrc" "$HOME/.profile"; do
  [ -f "$rc_file" ] || touch "$rc_file"
  if ! grep -Fqx "$local_bin_path_line" "$rc_file"; then
    printf "\n%s\n" "$local_bin_path_line" >> "$rc_file"
  fi
  if ! grep -Fqx "$opencode_path_line" "$rc_file"; then
    printf "%s\n" "$opencode_path_line" >> "$rc_file"
  fi
  if ! grep -Fqx "$env_line" "$rc_file"; then
    printf "%s\n" "$env_line" >> "$rc_file"
  fi
done

if [ -f "$HOME/.env" ]; then
  set -a
  . "$HOME/.env"
  set +a
fi
command -v opencode >/dev/null 2>&1
command -v ast-grep >/dev/null 2>&1 || command -v sg >/dev/null 2>&1
'

remote_home=$("${sprite_cmd[@]}" exec -s "$sprite_name" sh -c 'printf %s "$HOME"')
remote_auth="$remote_home/.local/share/opencode/auth.json"
remote_env="$remote_home/.env"
remote_config_parent="$remote_home/.config"
remote_repo_parent="$remote_home/code"
remote_repo_dir="$remote_repo_parent/$repo_name"
remote_tmp_tar="/tmp/opencode-config-$RANDOM-$RANDOM.tar.gz"

printf "Preparing remote directories...\n"
"${sprite_cmd[@]}" exec -s "$sprite_name" sh -c 'mkdir -p "$HOME/.local/share/opencode" "$HOME/.config"'

config_tar=$(mktemp -t opencode-config)
cleanup() {
  rm -f "$config_tar"
}
trap cleanup EXIT

printf "Packing local config directory...\n"
COPYFILE_DISABLE=1 tar --no-xattrs --no-mac-metadata -C "$HOME/.config" -czf "$config_tar" opencode

printf "Copying auth.json...\n"
"${sprite_cmd[@]}" exec -s "$sprite_name" -file "$local_auth:$remote_auth" sh -c 'true'

printf "Copying .env...\n"
"${sprite_cmd[@]}" exec -s "$sprite_name" -file "$local_env_file:$remote_env" sh -c 'true'

printf "Copying ~/.config/opencode...\n"
"${sprite_cmd[@]}" exec -s "$sprite_name" -file "$config_tar:$remote_tmp_tar" sh -c "tar -xzf \"$remote_tmp_tar\" -C \"$remote_config_parent\" && rm -f \"$remote_tmp_tar\""

printf "Cloning repository and preparing branch inside sprite...\n"
"${sprite_cmd[@]}" exec -s "$sprite_name" -env "SPRITE_REPO_URL=$repo_url,SPRITE_REPO_DIR=$remote_repo_dir,SPRITE_TARGET_BRANCH=$target_branch" zsh -lc '
set -e

if ! command -v git >/dev/null 2>&1; then
  printf "git is required inside sprite but was not found.\n" >&2
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
  git checkout "$SPRITE_TARGET_BRANCH"
elif git ls-remote --exit-code --heads origin "$SPRITE_TARGET_BRANCH" >/dev/null 2>&1; then
  git checkout -b "$SPRITE_TARGET_BRANCH" --track "origin/$SPRITE_TARGET_BRANCH"
else
  default_remote_head="$(git symbolic-ref --quiet --short refs/remotes/origin/HEAD 2>/dev/null || true)"
  if [ -z "$default_remote_head" ]; then
    if git show-ref --verify --quiet refs/remotes/origin/main; then
      default_remote_head="origin/main"
    elif git show-ref --verify --quiet refs/remotes/origin/master; then
      default_remote_head="origin/master"
    else
      printf "Unable to determine the default branch from origin.\n" >&2
      exit 1
    fi
  fi

  git checkout -b "$SPRITE_TARGET_BRANCH" "$default_remote_head"
fi

if git ls-remote --exit-code --heads origin "$SPRITE_TARGET_BRANCH" >/dev/null 2>&1; then
  git branch --set-upstream-to="origin/$SPRITE_TARGET_BRANCH" "$SPRITE_TARGET_BRANCH" >/dev/null 2>&1 || true
fi
'

printf "Done. Sprite '%s' is ready for %s on branch '%s'.\n" "$sprite_name" "$repo_name" "$target_branch"

printf "Connecting via sprite console and launching OpenCode...\n"
if command -v expect >/dev/null 2>&1; then
  if SPRITE_NAME="$sprite_name" SPRITE_ORG="$sprite_org" SPRITE_REPO_DIR="$remote_repo_dir" expect -c '
set timeout 20
set cmd [list sprite]
if {[info exists env(SPRITE_ORG)] && $env(SPRITE_ORG) ne ""} {
  lappend cmd -o $env(SPRITE_ORG)
}
lappend cmd console -s $env(SPRITE_NAME)
spawn {*}$cmd
after 1200
send -- ". ~/.profile >/dev/null 2>&1 || true; cd \"$env(SPRITE_REPO_DIR)\" >/dev/null 2>&1 || true; hash -r 2>/dev/null || true; opencode\r"
interact
'; then
    exit 0
  fi

  printf "Automatic console launch failed, falling back to direct launch.\n" >&2
else
  printf "'expect' is not installed, falling back to direct launch.\n"
fi
exec "${sprite_cmd[@]}" exec -s "$sprite_name" -env "SPRITE_REPO_DIR=$remote_repo_dir" -tty zsh -lc '. ~/.zprofile >/dev/null 2>&1 || true; . ~/.zshrc >/dev/null 2>&1 || true; cd "$SPRITE_REPO_DIR" >/dev/null 2>&1 || true; hash -r 2>/dev/null || true; if command -v opencode >/dev/null 2>&1; then exec opencode; fi; exec "$HOME/.opencode/bin/opencode"'
