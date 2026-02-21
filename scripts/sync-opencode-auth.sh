#!/bin/bash
set -euo pipefail

AUTH_FILE="${OPENCODE_AUTH_FILE:-$HOME/.local/share/opencode/auth.json}"
DEV_VARS_FILE="cloud/opencode-worker/.dev.vars"
VAR_NAME="OPENCODE_AUTH_OPENAI_B64"

usage() {
    cat <<EOF
Usage: $(basename "$0") [MODE]

Sync OpenAI auth from opencode to Worker environment.

Modes:
  --local    Write .dev.vars for local development
  --live     Upload secret to deployed Worker via wrangler
  --both     Perform both --local and --live

Environment:
  OPENCODE_AUTH_FILE  Override auth.json location (default: ~/.local/share/opencode/auth.json)

Security:
  Credential values are never printed to stdout/stderr.
EOF
    exit 1
}

die() {
    echo "Error: $1" >&2
    exit 1
}

validate_auth_file() {
    if [[ ! -f "$AUTH_FILE" ]]; then
        die "Auth file not found: $AUTH_FILE"
    fi

    if ! jq -e . "$AUTH_FILE" >/dev/null 2>&1; then
        die "Invalid JSON in auth file: $AUTH_FILE"
    fi

    if ! jq -e '.openai' "$AUTH_FILE" >/dev/null 2>&1; then
        die "No OpenAI auth entry found in: $AUTH_FILE"
    fi
}

extract_openai_payload() {
    jq -c '.openai' "$AUTH_FILE" 2>/dev/null || die "Failed to extract OpenAI payload"
}

encode_payload() {
    local payload="$1"
    echo -n "$payload" | base64
}

write_local() {
    local encoded="$1"

    mkdir -p "$(dirname "$DEV_VARS_FILE")"

    if [[ -f "$DEV_VARS_FILE" ]]; then
        if grep -q "^${VAR_NAME}=" "$DEV_VARS_FILE" 2>/dev/null; then
            sed -i.bak "s|^${VAR_NAME}=.*|${VAR_NAME}=${encoded}|" "$DEV_VARS_FILE" && rm -f "${DEV_VARS_FILE}.bak"
        else
            echo "${VAR_NAME}=${encoded}" >> "$DEV_VARS_FILE"
        fi
    else
        echo "${VAR_NAME}=${encoded}" > "$DEV_VARS_FILE"
    fi

    echo "Wrote $DEV_VARS_FILE"
}

upload_live() {
    local encoded="$1"

    if ! command -v wrangler &>/dev/null; then
        die "wrangler CLI not found. Install with: npm install -g wrangler"
    fi

    cd cloud/opencode-worker

    echo "$encoded" | wrangler secret put "$VAR_NAME" 2>&1 | grep -v "^$" || true

    echo "Uploaded secret to Worker"
}

main() {
    local mode="${1:-}"

    case "$mode" in
        --local|--live|--both) ;;
        *) usage ;;
    esac

    validate_auth_file

    local payload
    payload=$(extract_openai_payload)

    local encoded
    encoded=$(encode_payload "$payload")

    case "$mode" in
        --local)
            write_local "$encoded"
            ;;
        --live)
            upload_live "$encoded"
            ;;
        --both)
            write_local "$encoded"
            upload_live "$encoded"
            ;;
    esac
}

main "$@"
