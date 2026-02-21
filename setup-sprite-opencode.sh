#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: setup-sprite-opencode.sh [--org <org-name>]

Creates a Sprite environment with a two-word random name, installs OpenCode inside it,
copies ~/.local/share/opencode/auth.json, copies ~/.config/opencode,
then opens sprite console and launches opencode.

Options:
  -o, --org <org-name>   Sprite organization name
  -h, --help             Show this help
EOF
}

sprite_org="${SPRITE_ORG:-}"

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
  printf '%s-%s' "$adjective" "$noun"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
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

local_auth="$HOME/.local/share/opencode/auth.json"
local_config_dir="$HOME/.config/opencode"

if [[ ! -f "$local_auth" ]]; then
  printf "Missing file: %s\n" "$local_auth" >&2
  exit 1
fi

if [[ ! -d "$local_config_dir" ]]; then
  printf "Missing directory: %s\n" "$local_config_dir" >&2
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

printf "Installing OpenCode inside sprite...\n"
"${sprite_cmd[@]}" exec -s "$sprite_name" zsh -lc '
set -e
if [ ! -x "$HOME/.opencode/bin/opencode" ] && ! command -v opencode >/dev/null 2>&1; then
  curl -fsSL https://opencode.ai/install | bash
fi
'

remote_home=$("${sprite_cmd[@]}" exec -s "$sprite_name" sh -c 'printf %s "$HOME"')
remote_auth="$remote_home/.local/share/opencode/auth.json"
remote_config_parent="$remote_home/.config"
remote_tmp_tar="/tmp/opencode-config-$RANDOM-$RANDOM.tar.gz"
remote_opencode_bin="$remote_home/.opencode/bin/opencode"

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

printf "Copying ~/.config/opencode...\n"
"${sprite_cmd[@]}" exec -s "$sprite_name" -file "$config_tar:$remote_tmp_tar" sh -c "tar -xzf \"$remote_tmp_tar\" -C \"$remote_config_parent\" && rm -f \"$remote_tmp_tar\""

printf "Done. Sprite '%s' is ready with OpenCode auth and config.\n" "$sprite_name"

printf "Connecting via sprite console and launching OpenCode...\n"
if command -v expect >/dev/null 2>&1; then
  if SPRITE_NAME="$sprite_name" SPRITE_ORG="$sprite_org" REMOTE_OPENCODE_BIN="$remote_opencode_bin" expect -c '
set timeout 20
set cmd [list sprite]
if {[info exists env(SPRITE_ORG)] && $env(SPRITE_ORG) ne ""} {
  lappend cmd -o $env(SPRITE_ORG)
}
lappend cmd console -s $env(SPRITE_NAME)
spawn {*}$cmd
after 1200
send -- "$env(REMOTE_OPENCODE_BIN)\r"
interact
'; then
    exit 0
  fi

  printf "Automatic console launch failed, falling back to direct launch.\n" >&2
else
  printf "'expect' is not installed, falling back to direct launch.\n"
fi
exec "${sprite_cmd[@]}" exec -s "$sprite_name" -tty "$remote_opencode_bin"
