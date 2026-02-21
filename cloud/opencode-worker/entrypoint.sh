#!/bin/bash
set -euo pipefail

AUTH_DIR_USER="/home/opencode/.local/share/opencode"
AUTH_DIR_ROOT="/root/.local/share/opencode"
AUTH_FILE="auth.json"

log() {
  echo "[opencode-entrypoint] $*"
}

error_exit() {
  echo "[opencode-entrypoint] ERROR: $*" >&2
  exit 1
}

setup_auth() {
  local auth_b64="${OPENCODE_AUTH_OPENAI_B64:-}"
  
  if [[ -z "$auth_b64" ]]; then
    error_exit "Missing OPENCODE_AUTH_OPENAI_B64 environment variable. Set it to base64-encoded JSON with OpenAI credentials."
  fi
  
  local auth_json
  if ! auth_json=$(echo "$auth_b64" | base64 -d 2>/dev/null); then
    error_exit "Failed to decode OPENCODE_AUTH_OPENAI_B64. Ensure it is valid base64-encoded JSON."
  fi
  
  if ! echo "$auth_json" | jq empty 2>/dev/null; then
    error_exit "Decoded OPENCODE_AUTH_OPENAI_B64 is not valid JSON. Got: ${auth_json:0:100}..."
  fi
  
  mkdir -p "$AUTH_DIR_USER"
  echo "$auth_json" > "$AUTH_DIR_USER/$AUTH_FILE"
  chmod 600 "$AUTH_DIR_USER/$AUTH_FILE"
  chown opencode:opencode "$AUTH_DIR_USER/$AUTH_FILE"
  log "Wrote auth to $AUTH_DIR_USER/$AUTH_FILE"
  
  mkdir -p "$AUTH_DIR_ROOT"
  echo "$auth_json" > "$AUTH_DIR_ROOT/$AUTH_FILE"
  chmod 600 "$AUTH_DIR_ROOT/$AUTH_FILE"
  log "Wrote auth to $AUTH_DIR_ROOT/$AUTH_FILE (compat fallback)"
}

validate_auth() {
  if [[ ! -f "$AUTH_DIR_USER/$AUTH_FILE" ]]; then
    error_exit "Auth file not found at $AUTH_DIR_USER/$AUTH_FILE. This should not happen."
  fi
  
  if [[ ! -r "$AUTH_DIR_USER/$AUTH_FILE" ]]; then
    error_exit "Auth file at $AUTH_DIR_USER/$AUTH_FILE is not readable."
  fi
  
  local perms
  perms=$(stat -c %a "$AUTH_DIR_USER/$AUTH_FILE" 2>/dev/null || stat -f %Lp "$AUTH_DIR_USER/$AUTH_FILE")
  if [[ "$perms" != "600" ]]; then
    log "Warning: Auth file permissions are $perms, expected 600. Fixing..."
    chmod 600 "$AUTH_DIR_USER/$AUTH_FILE"
  fi
  
  log "Auth validation passed."
}

bootstrap_opencode_config() {
  local config_dir="/home/opencode/.config/opencode"
  mkdir -p "$config_dir"
  
  if [[ ! -f "$config_dir/config.json" ]]; then
    cat > "$config_dir/config.json" <<'EOF'
{
  "provider": "openai"
}
EOF
    chown opencode:opencode "$config_dir/config.json"
    log "Created default OpenCode config with OpenAI provider."
  else
    log "OpenCode config already exists, skipping bootstrap."
  fi
}

main() {
  log "Starting OpenCode sandbox initialization..."
  
  setup_auth
  validate_auth
  bootstrap_opencode_config
  
  log "Initialization complete. Starting OpenCode..."
  exec opencode serve --port "${OPENCODE_PORT:-3000}"
}

main "$@"
