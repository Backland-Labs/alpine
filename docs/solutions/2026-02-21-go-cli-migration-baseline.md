# Go CLI Migration Baseline

- Implemented `cmd/setup-sprite-opencode` with deterministic flag parsing and exit-code taxonomy.
- Added preflight checks for git context and required local files before any remote mutation.
- Migrated core flow to `sprites-go` (`CreateSprite`, `ListAllSprites`, remote `CommandContext`, and `Filesystem`).
- Added transfer allowlist, traversal rejection, symlink rejection, and backup/restore behavior.
- Added non-TTY output contract (`status`, `sprite_id`, reconnect command) and interactive TTY launch path.
