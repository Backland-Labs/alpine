# setup-sprite-opencode parity matrix

Date: 2026-02-21

This compares intended behavior between legacy `setup-sprite-opencode.sh` and the Go CLI `cmd/setup-sprite-opencode`.

| Scenario | Legacy script | Go CLI | Status |
|---|---|---|---|
| Help output (`--help`) | Prints usage and exits `0` | Prints usage and exits `0` | parity |
| Missing `--branch` | Usage error | Usage error (`exit 2`) | parity+ |
| Invalid branch name | Implicit later failure | Early validation via `git check-ref-format --branch` | intentional improvement |
| Local preflight checks | yes | yes | parity |
| Sprite naming | slug + adjective/noun | slug + adjective/noun | parity |
| Name collision handling | retries via list check | retries on list and create collision | parity+ |
| Core sprite operations | `sprite` CLI shell-outs | `sprites-go` SDK | intentional migration |
| Bootstrap OpenCode + ast-grep | yes | yes | parity |
| Profile idempotency | append-if-missing | append-if-missing | parity |
| Transfer auth/config/.claude/.env | yes | yes + allowlist/safety checks | parity+ |
| Git clone/checkout behavior | local/remote/default fallback | same decision table behavior | parity |
| Interactive launch | expect or direct | TTY attach via SDK | parity |
| Non-TTY behavior | fallback shell path | emits reconnect command + sprite id | intentional improvement |

`parity+` means compatibility retained with an explicit safety or reliability improvement.
